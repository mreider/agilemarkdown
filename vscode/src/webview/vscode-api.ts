// VS Code injects `acquireVsCodeApi` into the webview iframe and only
// allows one call per page. Multiple modules calling it themselves
// throws "An instance of the VS Code API has already been acquired"
// and stops the bundle dead. Acquire once here, import everywhere.
//
// Wrap postMessage so the rest of the app can tell when the
// extension is busy. Every CLI write shells out through `am`, and the
// round trip takes long enough that an unindicated click feels broken.
// The wrapper counts outstanding writes and dispatches `am:busy` on
// the window whenever the count crosses zero. The App listens and
// renders a "Saving…" pill in the topbar; per-button consumers can
// subscribe to the same event for inline spinners.
//
// A burst of writes (e.g. set-tags then a snapshot fan-out) all share
// one busy window, so the UI stays calm instead of flickering once per
// CLI call.

const raw = (typeof acquireVsCodeApi === 'function') ? acquireVsCodeApi() : null;

let pending = 0;
let settleTimer: ReturnType<typeof setTimeout> | null = null;
let fallbackTimer: ReturnType<typeof setTimeout> | null = null;

const SETTLE_MS = 80;
// Hard upper bound on a busy window. Some message types don't yield
// an incoming reply (detail-closed, error-swallowed writes, etc.); the
// settle timer alone never fires for those and the indicator gets
// stuck. The fallback clears pending unconditionally after this many
// ms of no further posts.
const FALLBACK_MS = 3000;

function emit() {
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent('am:busy', { detail: { pending } }));
  }
}

function clearBusy() {
  pending = 0;
  if (settleTimer) { clearTimeout(settleTimer); settleTimer = null; }
  if (fallbackTimer) { clearTimeout(fallbackTimer); fallbackTimer = null; }
  emit();
}

if (typeof window !== 'undefined') {
  // Incoming message → schedule a settle 80ms out. Bursts of fan-out
  // messages (snapshot + detail) keep resetting the timer so the
  // indicator stays on for the whole burst.
  window.addEventListener('message', () => {
    if (pending === 0) return;
    if (settleTimer) clearTimeout(settleTimer);
    settleTimer = setTimeout(clearBusy, SETTLE_MS);
  });
}

export const vscode = raw ? {
  postMessage(msg: any) {
    pending = Math.max(0, pending + 1);
    emit();
    // Cancel any pending settle from a previous burst; the fresh post
    // means the user is still asking for work and we're not idle yet.
    if (settleTimer) { clearTimeout(settleTimer); settleTimer = null; }
    // Re-arm the fallback. If no incoming message lands within 3s,
    // assume the operation produced no reply (or quietly failed) and
    // force-clear the spinner so the user isn't stuck looking at a
    // permanent "Saving…".
    if (fallbackTimer) clearTimeout(fallbackTimer);
    fallbackTimer = setTimeout(clearBusy, FALLBACK_MS);
    raw.postMessage(msg);
  },
  getState: raw.getState?.bind(raw),
  setState: raw.setState?.bind(raw),
} : null;

// useBusy returns the current busy flag as React state, subscribing
// to the `am:busy` event for updates. Components import this to show
// inline spinners; the topbar pill uses it for the global indicator.
//
// MIN_VISIBLE_MS keeps the spinner on screen long enough to register
// visually even when the CLI returns very fast. A click → 50ms shell
// out → reply is too brief to perceive a flicker, so we floor every
// busy window at 250ms. The pending counter clears as soon as the
// response lands, but the hook holds the UI state down for the rest
// of the window.
import { useEffect, useState, useRef } from 'react';
const MIN_VISIBLE_MS = 250;

export function useBusy(): boolean {
  const [busy, setBusy] = useState(false);
  const becameBusyAtRef = useRef<number>(0);
  const offTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  useEffect(() => {
    function onBusy(e: Event) {
      const ce = e as CustomEvent<{ pending: number }>;
      const nowBusy = (ce.detail?.pending ?? 0) > 0;
      if (nowBusy) {
        if (offTimerRef.current) {
          clearTimeout(offTimerRef.current);
          offTimerRef.current = null;
        }
        if (becameBusyAtRef.current === 0) becameBusyAtRef.current = Date.now();
        setBusy(true);
        return;
      }
      // Pending dropped to zero. Hold the spinner on for at least
      // MIN_VISIBLE_MS since it first appeared so quick operations
      // still produce a visible flash.
      const elapsed = Date.now() - (becameBusyAtRef.current || Date.now());
      const remaining = Math.max(0, MIN_VISIBLE_MS - elapsed);
      if (offTimerRef.current) clearTimeout(offTimerRef.current);
      offTimerRef.current = setTimeout(() => {
        becameBusyAtRef.current = 0;
        offTimerRef.current = null;
        setBusy(false);
      }, remaining);
    }
    window.addEventListener('am:busy', onBusy);
    return () => {
      window.removeEventListener('am:busy', onBusy);
      if (offTimerRef.current) clearTimeout(offTimerRef.current);
    };
  }, []);
  return busy;
}
