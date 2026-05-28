// Web-target replacement for vscode-api.ts. The extension build uses
// the VS Code postMessage API; the laboratory build uses HTTP + a
// snapshot WebSocket. Components import `vscode` from the same path
// (./vscode-api), webpack's resolve.alias maps it to this file when
// AM_TARGET=web.
//
// The contract preserved here:
//   - `vscode.postMessage(msg)` dispatches a write to the backend.
//   - The backend emits a new snapshot which lands as a
//     `window.message` event of type `{type: 'snapshot', payload}`.
//   - Extra messages the App listens for (visible-columns, project,
//     whoami, detail, refresh, error) are synthesized here in the
//     web shim so the React tree behaves the same way.
//
// Busy tracking mirrors the extension's vscode-api: counts outstanding
// writes and dispatches `am:busy` so the topbar's "Saving…" pill works
// without any change to App.tsx.

declare global {
  interface Window {
    __amWebBoot?: () => Promise<void>;
  }
}

type Msg = { type: string; payload?: any };

const apiBase = '/api';
const wsSnapshotUrl = locationWs('/ws/snapshot');
const wsLogUrl = locationWs('/ws/log');
void wsLogUrl;

let pending = 0;
const SETTLE_MS = 80;

function emit() {
  window.dispatchEvent(new CustomEvent('am:busy', { detail: { pending } }));
}
function bump() { pending++; emit(); }
function settle() {
  // Each bump gets its own decrement timer. A single shared timer
  // would leak: when two writes overlap, the second settle() clears
  // the first's timer, so one decrement is lost and `pending` sticks
  // above zero, the "Saving…" pill then never clears.
  setTimeout(() => { pending = Math.max(0, pending - 1); emit(); }, SETTLE_MS);
}

// Translation table: webview message type → HTTP route + body shape.
// The extension's webview emits these messages; in the lab they get
// dispatched directly to the Go backend.
const routes: Record<string, { path: string; body?(p: any): any }> = {
  'load':                  { path: '/snapshot' }, // GET, handled specially
  'set-status':            { path: '/set-status' },
  'set-estimate':          { path: '/set-estimate' },
  'set-assigned':          { path: '/set-assigned' },
  'set-tags':              { path: '/set-tags' },
  'set-epic':              { path: '/set-epic' },
  'set-type':              { path: '/set-type' },
  'set-release-date':      { path: '/set-release-date' },
  'set-description':       { path: '/set-description' },
  'add-comment':           { path: '/comment' },
  'comment':               { path: '/comment' },
  'add-task':              { path: '/task' },
  'task-add':              { path: '/task' },
  'set-task-done':         { path: '/task', body: (p: any) => ({ path: p.path, idx: p.idx ?? p.index }) },
  'task-tick':             { path: '/task' },
  'create-item':           { path: '/create-item' },
  'delete-item':           { path: '/delete-item' },
  'rank-item':             { path: '/rank' },
  'move-to-icebox':        { path: '/move-to-icebox' },
  'move-to-priority':      { path: '/move-to-priority' },
  'pull-from-icebox':      { path: '/move-to-priority',
                              body: (p: any) => ({ backlog: p.backlog, item_paths: [p.item_path], position: 'top' }) },
  'block-item':            { path: '/block' },
  'block':                 { path: '/block' },
  'unblock-item':          { path: '/unblock' },
  'unblock':               { path: '/unblock' },
  'reject-item':           { path: '/reject' },
  'reject':                { path: '/reject' },
  'sync':                  { path: '/sync' },
};

async function call(msg: Msg): Promise<any> {
  // open-detail: the extension routes this through the extension
  // host; the lab needs to fetch the item itself and synthesize the
  // `detail` message the DetailPanel listens for. Keeps the React
  // tree unchanged.
  if (msg.type === 'open-detail') {
    const path = msg.payload?.path;
    if (!path) return null;
    bump();
    try {
      const r = await fetch(`${apiBase}/get-item?path=${encodeURIComponent(path)}`);
      if (!r.ok) throw new Error(await r.text());
      const body = await r.json();
      synth('detail', { ...body, autoAction: msg.payload?.autoAction ?? null });
      return body;
    } finally {
      settle();
    }
  }
  if (msg.type === 'detail-closed') {
    // No backend op; the App clears the panel on its own. Swallow.
    return null;
  }
  const route = routes[msg.type];
  if (!route) {
    // Messages the lab doesn't need to round-trip (e.g. search-only
    // host events). Handled locally.
    return null;
  }
  bump();
  try {
    if (msg.type === 'load') {
      const r = await fetch(apiBase + route.path);
      if (!r.ok) throw new Error(await r.text());
      const snap = await r.json();
      synthSnapshot(snap);
      return snap;
    }
    const body = route.body ? route.body(msg.payload) : msg.payload;
    const r = await fetch(apiBase + route.path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body || {}),
    });
    if (!r.ok) throw new Error(await r.text());
    return await r.json();
  } finally {
    settle();
  }
}

function synthSnapshot(payload: any) {
  window.dispatchEvent(new MessageEvent('message', { data: { type: 'snapshot', payload } }));
}

function synth(type: string, payload: any) {
  window.dispatchEvent(new MessageEvent('message', { data: { type, payload } }));
}

// Connect the snapshot WebSocket: every push from the backend lands
// as a `snapshot` message just like in the extension. Reconnect on
// drop so casual disconnects don't break the UI.
let snapWs: WebSocket | null = null;
function connectSnapshotWs() {
  snapWs = new WebSocket(wsSnapshotUrl);
  snapWs.onmessage = (e) => {
    try {
      const snap = JSON.parse(e.data);
      synthSnapshot(snap);
    } catch (err) {
      console.error('snapshot ws parse:', err);
    }
  };
  snapWs.onclose = () => {
    setTimeout(connectSnapshotWs, 1000);
  };
  snapWs.onerror = () => { try { snapWs?.close(); } catch {} };
}

// Connect the CommandLog WebSocket. Republishes each event as an
// `am:log` CustomEvent on `window` so any UI component (the
// CommandLog tab, plus anything else that wants to listen) can
// subscribe without sharing the connection.
let logWs: WebSocket | null = null;
function connectLogWs() {
  logWs = new WebSocket(wsLogUrl);
  logWs.onmessage = (e) => {
    try {
      const ev = JSON.parse(e.data);
      window.dispatchEvent(new CustomEvent('am:log', { detail: ev }));
    } catch (err) {
      console.error('log ws parse:', err);
    }
  };
  logWs.onclose = () => {
    setTimeout(connectLogWs, 1000);
  };
  logWs.onerror = () => { try { logWs?.close(); } catch {} };
}

// Boot the web shim once. Called from index.tsx after React mounts.
window.__amWebBoot = async () => {
  connectSnapshotWs();
  connectLogWs();
  // Project label so the topbar reads sensibly before the first
  // snapshot lands.
  synth('project', { name: location.host.split(':')[0] || 'laboratory', folderName: 'laboratory' });
  synth('whoami', { name: 'lab', email: 'lab@local' });
};

function locationWs(path: string): string {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${proto}//${location.host}${path}`;
}

// Export the same shape vscode-api.ts exports so the rest of the
// codebase compiles against either build target. The `vscode` symbol
// has `.postMessage(msg)` and that's it.
//
// Calls are serialized through a promise chain. The extension host
// processes postMessage sequentially; the lab must too, or multi-step
// flows race. The New Story modal, for instance, fires create-item →
// set-type → set-release-date; without serialization set-release-date
// can land before create-item and write a stub file with no type or
// title (release markers then never render).
let chain: Promise<unknown> = Promise.resolve();
export const vscode = {
  postMessage: (msg: Msg) => {
    chain = chain.then(() => call(msg)).catch((e) => {
      console.error('lab call failed:', msg, e);
      synth('error', { message: String(e?.message || e) });
    });
  },
};

// Re-export the busy-pill hook used by App.tsx. It must return a
// boolean to match vscode-api.ts's `useBusy(): boolean`, BusyPill
// does `const busy = useBusy(); if (!busy && !loading) return null`.
// Returning an object here made `busy` always truthy, so the pill
// never unmounted and the lab showed "Saving…" forever.
import { useEffect, useState } from 'react';
export function useBusy(): boolean {
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    const on = (e: any) => setBusy((e?.detail?.pending ?? 0) > 0);
    window.addEventListener('am:busy', on as any);
    return () => window.removeEventListener('am:busy', on as any);
  }, []);
  return busy;
}
