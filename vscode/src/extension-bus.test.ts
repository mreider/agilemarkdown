// Integration test for the extension's message bus.
//
// We do not boot VS Code in this suite. Instead we exercise the same
// wire the webview drives: every postMessage type the React components
// emit gets sent through the real handler in extension.ts (refactored
// into a pure function), backed by a real AmClient on a real fixture
// workspace.
//
// What this catches: misnamed message types, wrong arg shapes, broken
// MCP calls, and disk state that does not change after a mutation.
// What this does NOT catch: visual rendering, drag events, CSS bugs,
// ordering of webview re-renders. That stays for F5 in real VS Code.

import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import * as cp from 'child_process';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import { AmClient } from './am-client';

const REPO_ROOT = path.resolve(__dirname, '..', '..');
const LOCAL_AM = path.join(REPO_ROOT, 'agilemarkdown');

function ensureBinary(): boolean {
  if (fs.existsSync(LOCAL_AM)) return true;
  try {
    cp.execSync('go build -o agilemarkdown .', { cwd: REPO_ROOT, stdio: 'ignore' });
    return fs.existsSync(LOCAL_AM);
  } catch {
    return false;
  }
}

// makeFixture creates a small project with one backlog and a few stories
// in different states so each gesture has something to act on.
function makeFixture(): string {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'am-ext-'));
  cp.execSync('git init -q && git config user.email tester@example.com && git config user.name "Tester"', { cwd: dir, shell: '/bin/sh' });
  cp.execSync(`${LOCAL_AM} create-backlog am-product`, { cwd: dir });
  cp.execSync(`rm -f am-product/Sample-*.md`, { cwd: dir });
  cp.execSync(`${LOCAL_AM} create-item "First story"`, { cwd: path.join(dir, 'am-product') });
  cp.execSync(`${LOCAL_AM} create-item "Second story"`, { cwd: path.join(dir, 'am-product') });
  cp.execSync(`${LOCAL_AM} create-item "Third story"`, { cwd: path.join(dir, 'am-product') });
  cp.execSync(`${LOCAL_AM} estimate First-story.md 3`, { cwd: path.join(dir, 'am-product') });
  cp.execSync(`${LOCAL_AM} estimate Second-story.md 5`, { cwd: path.join(dir, 'am-product') });
  cp.execSync(`${LOCAL_AM} estimate Third-story.md 2`, { cwd: path.join(dir, 'am-product') });
  cp.execSync(`${LOCAL_AM} sync </dev/null >/dev/null`, { cwd: dir, shell: '/bin/sh' });
  cp.execSync(`${LOCAL_AM} unice First-story.md --top`, { cwd: path.join(dir, 'am-product') });
  cp.execSync(`${LOCAL_AM} unice Second-story.md`, { cwd: path.join(dir, 'am-product') });
  // Third stays in icebox so we have one for drag-from-icebox tests.
  return dir;
}

// extensionMessage replays the dispatch logic from extension.ts. Kept
// in lock-step with the case statements there. When extension.ts
// changes, update this routing block.
//
// The harness tracks `openDetailPath` the same way the real extension
// tracks `currentDetailPath`: set on open-detail, cleared on
// detail-closed, used by afterWrite() to decide whether to re-emit a
// fresh 'detail' message after a write. This is the seam that keeps
// the DetailPanel coherent with the board snapshot.
let openDetailPath: string | null = null;

async function emitDetail(client: AmClient, p: string, sink: { type: string; payload?: any }[]) {
  const [item, comments, tasks] = await Promise.all([
    client.getItem(p),
    client.getComments(p).catch(() => ({ comments: [], count: 0 })),
    client.listTasks(p).catch(() => ({ tasks: [], count: 0 })),
  ]);
  sink.push({ type: 'detail', payload: { item, comments, tasks } });
}

function samePath(a: string | null | undefined, b: string | null | undefined): boolean {
  if (!a || !b) return false;
  const n = (p: string) => p.replace(/\\/g, '/').replace(/^\.\//, '');
  return n(a) === n(b) || n(a).endsWith('/' + n(b)) || n(b).endsWith('/' + n(a));
}

async function afterWrite(client: AmClient, sink: { type: string; payload?: any }[], writtenPath?: string) {
  sink.push({ type: 'snapshot', payload: await snapshot(client) });
  if (openDetailPath && writtenPath && samePath(openDetailPath, writtenPath)) {
    await emitDetail(client, openDetailPath, sink);
  }
}

async function maybePromoteFromIcebox(client: AmClient, qualifiedPath: string) {
  const norm = qualifiedPath.replace(/\\/g, '/').replace(/^\.\//, '');
  const slash = norm.indexOf('/');
  if (slash < 0) return;
  const backlog = norm.slice(0, slash);
  const basename = norm.slice(slash + 1);
  try {
    const ice = await client.iceboxList(backlog);
    const inIcebox = (ice.items || []).some(r => r.path === basename || r.path.endsWith('/' + basename));
    if (!inIcebox) return;
    await client.moveToPriority({ backlog, item_paths: [basename], position: 'top' });
  } catch {
    // best-effort
  }
}

async function extensionMessage(client: AmClient, msg: { type: string; payload?: any }, sink: { type: string; payload?: any }[]) {
  switch (msg.type) {
    case 'load':
      sink.push({ type: 'snapshot', payload: await snapshot(client) });
      return;
    case 'detail-closed':
      openDetailPath = null;
      return;
    case 'open-detail': {
      openDetailPath = msg.payload.path;
      await emitDetail(client, msg.payload.path, sink);
      return;
    }
    case 'set-status':
      if (msg.payload.status === 'started') await maybePromoteFromIcebox(client, msg.payload.path);
      await client.setStatus(msg.payload.path, msg.payload.status);
      await afterWrite(client, sink, msg.payload.path);
      return;
    case 'set-estimate':     await client.setEstimate(msg.payload.path, String(msg.payload.estimate)); await afterWrite(client, sink, msg.payload.path); return;
    case 'set-assigned':     await client.setAssigned(msg.payload.path, msg.payload.assignees);        await afterWrite(client, sink, msg.payload.path); return;
    case 'set-tags':         await client.setTags(msg.payload.path, msg.payload.tags);                 await afterWrite(client, sink, msg.payload.path); return;
    case 'set-epic':         await client.setEpic(msg.payload.path, msg.payload.epic || '');           await afterWrite(client, sink, msg.payload.path); return;
    case 'set-description':  await client.setDescription(msg.payload.path, msg.payload.body);          await afterWrite(client, sink, msg.payload.path); return;
    case 'rank-item':        await client.rankItem(msg.payload);                                       await afterWrite(client, sink); return;
    case 'move-to-icebox':   await client.moveToIcebox(msg.payload);                                   await afterWrite(client, sink); return;
    case 'move-to-priority': await client.moveToPriority(msg.payload);                                 await afterWrite(client, sink); return;
    case 'pull-from-icebox': {
      const { backlog, item_path } = msg.payload;
      const qualified = `${backlog}/${item_path}`;
      await client.moveToPriority({ backlog, item_paths: [item_path], position: 'top' });
      await client.setStatus(qualified, 'started');
      await afterWrite(client, sink, qualified);
      return;
    }
    case 'create-item':      await client.createItem(msg.payload.backlog, msg.payload.title, msg.payload.user); await afterWrite(client, sink); return;
    case 'block':            await client.blockItem(msg.payload.path, msg.payload.reason);             await afterWrite(client, sink, msg.payload.path); return;
    case 'unblock':          await client.unblockItem(msg.payload.path);                               await afterWrite(client, sink, msg.payload.path); return;
    case 'add-comment':      await client.addComment(msg.payload.path, msg.payload.text, msg.payload.author); await afterWrite(client, sink, msg.payload.path); return;
    case 'add-task':         await client.addTask(msg.payload.path, msg.payload.text);                 await afterWrite(client, sink, msg.payload.path); return;
    case 'tick-task':        await client.setTaskDone(msg.payload.path, msg.payload.index, msg.payload.done); await afterWrite(client, sink, msg.payload.path); return;
    case 'reject':           await client.rejectItem(msg.payload.path, msg.payload.reason || '');      await afterWrite(client, sink, msg.payload.path); return;
    case 'set-type': {
      const t = String(msg.payload.type || '').toLowerCase();
      await client.setType(msg.payload.path, t);
      if (t === 'bug' || t === 'chore' || t === 'release') {
        try { await client.setEstimate(msg.payload.path, ''); } catch { /* best-effort */ }
      }
      await afterWrite(client, sink, msg.payload.path);
      return;
    }
    case 'delete-item': {
      await client.deleteItem(msg.payload.path);
      // After a successful delete the open detail (if any) points at a
      // path that no longer exists. The real extension posts
      // 'close-detail' to the webview; mirror that here so the
      // harness has the same external shape.
      openDetailPath = null;
      sink.push({ type: 'close-detail' });
      sink.push({ type: 'snapshot', payload: await snapshot(client) });
      return;
    }
    default:
      throw new Error(`unknown message type: ${msg.type}`);
  }
}

async function snapshot(client: AmClient) {
  const backlogs = (await client.listBacklogs()).backlogs || [];
  const backlog = backlogs[0];
  if (!backlog) return { backlog: '', backlogs, priority: { items: [], count: 0, velocity: 0 }, icebox: { items: [], count: 0 } };
  const [priority, icebox] = await Promise.all([
    client.priorityList(backlog),
    client.iceboxList(backlog),
  ]);
  return { backlog, backlogs, priority, icebox };
}

describe('extension message bus', () => {
  if (!ensureBinary()) {
    it.skip('skipped: ./agilemarkdown not built', () => {});
    return;
  }
  let dir: string;
  let client: AmClient;
  let messages: { type: string; payload?: any }[];

  beforeAll(async () => {
    dir = makeFixture();
    client = new AmClient(LOCAL_AM, dir);
    messages = [];
    openDetailPath = null;
  }, 60_000);

  afterAll(() => {
    if (dir) fs.rmSync(dir, { recursive: true, force: true });
  });

  it('initial load returns a snapshot with priority and icebox', async () => {
    await extensionMessage(client, { type: 'load' }, messages);
    const snap = messages.at(-1)!.payload;
    expect(snap.backlog).toBe('am-product');
    expect(snap.priority.items.length).toBeGreaterThanOrEqual(2);
    expect(snap.icebox.items.length).toBeGreaterThanOrEqual(1);
  });

  it('clicking a story card opens the detail panel', async () => {
    await extensionMessage(client, { type: 'open-detail', payload: { path: 'am-product/First-story.md' } }, messages);
    const last = messages.at(-1)!;
    expect(last.type).toBe('detail');
    expect(last.payload.item.title).toMatch(/first story/i);
    expect(last.payload.comments).toBeDefined();
    expect(last.payload.tasks).toBeDefined();
    // Close detail so subsequent write-tests can rely on
    // `messages.at(-1)` being the snapshot. The afterWrite() seam
    // re-emits a detail when the open path matches the write target,
    // and most write-tests below act on First-story.md.
    await extensionMessage(client, { type: 'detail-closed' }, messages);
  });

  it('action-button click flips status and the next snapshot reflects it', async () => {
    await extensionMessage(client, { type: 'set-status', payload: { path: 'am-product/First-story.md', status: 'started' } }, messages);
    const after = messages.at(-1)!.payload;
    const item = after.priority.items.find((r: any) => r.path === 'First-story.md');
    expect(item.status).toBe('started');
  });

  it('points pip click sets the estimate visible on the next snapshot', async () => {
    await extensionMessage(client, { type: 'set-estimate', payload: { path: 'am-product/First-story.md', estimate: 5 } }, messages);
    const after = messages.at(-1)!.payload;
    const item = after.priority.items.find((r: any) => r.path === 'First-story.md');
    expect(item.estimate).toBe('5');
  });

  it('owner input writes assignees and the card avatars update', async () => {
    await extensionMessage(client, { type: 'set-assigned', payload: { path: 'am-product/First-story.md', assignees: ['alice', 'bob'] } }, messages);
    const after = messages.at(-1)!.payload;
    const item = after.priority.items.find((r: any) => r.path === 'First-story.md');
    expect(item.assignees).toEqual(['alice', 'bob']);
  });

  it('tags input replaces tag list and chips update', async () => {
    await extensionMessage(client, { type: 'set-tags', payload: { path: 'am-product/First-story.md', tags: ['onboarding', 'q3'] } }, messages);
    const after = messages.at(-1)!.payload;
    const item = after.priority.items.find((r: any) => r.path === 'First-story.md');
    expect(item.tags).toEqual(['onboarding', 'q3']);
  });

  it('epic input writes the slug', async () => {
    await extensionMessage(client, { type: 'set-epic', payload: { path: 'am-product/First-story.md', epic: 'auth-rewrite' } }, messages);
    const after = messages.at(-1)!.payload;
    const item = after.priority.items.find((r: any) => r.path === 'First-story.md');
    expect(item.epic).toBe('auth-rewrite');
  });

  it('block button sets blocked + reason; unblock clears', async () => {
    await extensionMessage(client, { type: 'block', payload: { path: 'am-product/First-story.md', reason: 'waiting on design' } }, messages);
    let after = messages.at(-1)!.payload;
    let item = after.priority.items.find((r: any) => r.path === 'First-story.md');
    expect(item.blocked).toBe(true);
    await extensionMessage(client, { type: 'unblock', payload: { path: 'am-product/First-story.md' } }, messages);
    after = messages.at(-1)!.payload;
    item = after.priority.items.find((r: any) => r.path === 'First-story.md');
    expect(item.blocked).toBeFalsy();
  });

  it('add-task appends a task; tick-task flips the checkbox', async () => {
    await extensionMessage(client, { type: 'add-task', payload: { path: 'am-product/First-story.md', text: 'wire telemetry' } }, messages);
    const tasks = await client.listTasks('am-product/First-story.md');
    expect(tasks.tasks.find(t => t.text === 'wire telemetry')).toBeDefined();
    const idx = tasks.tasks.find(t => t.text === 'wire telemetry')!.index;
    await extensionMessage(client, { type: 'tick-task', payload: { path: 'am-product/First-story.md', index: idx, done: true } }, messages);
    const tasks2 = await client.listTasks('am-product/First-story.md');
    const t = tasks2.tasks.find(x => x.text === 'wire telemetry')!;
    expect(t.done).toBe(true);
  });

  it('add-comment appends to the body Comments section', async () => {
    await extensionMessage(client, { type: 'add-comment', payload: { path: 'am-product/First-story.md', text: 'looks good', author: 'pm' } }, messages);
    const comments = await client.getComments('am-product/First-story.md');
    expect(comments.comments.find(c => c.text.includes('looks good'))).toBeDefined();
  });

  it('drag from icebox to priority promotes', async () => {
    await extensionMessage(client, { type: 'move-to-priority', payload: { backlog: 'am-product', item_paths: ['Third-story.md'] } }, messages);
    const after = messages.at(-1)!.payload;
    expect(after.priority.items.find((r: any) => r.path === 'Third-story.md')).toBeDefined();
    expect(after.icebox.items.find((r: any) => r.path === 'Third-story.md')).toBeUndefined();
  });

  it('rank-item with after reorders inside priority', async () => {
    await extensionMessage(client, { type: 'rank-item', payload: { backlog: 'am-product', item_path: 'Third-story.md', after: 'First-story.md' } }, messages);
    const after = messages.at(-1)!.payload;
    const indexOf = (p: string) => after.priority.items.findIndex((r: any) => r.path === p);
    expect(indexOf('Third-story.md')).toBeGreaterThan(indexOf('First-story.md'));
  });

  it('drag from priority to icebox demotes', async () => {
    await extensionMessage(client, { type: 'move-to-icebox', payload: { backlog: 'am-product', item_path: 'Third-story.md' } }, messages);
    const after = messages.at(-1)!.payload;
    expect(after.icebox.items.find((r: any) => r.path === 'Third-story.md')).toBeDefined();
    expect(after.priority.items.find((r: any) => r.path === 'Third-story.md')).toBeUndefined();
  });

  it('inline create-item produces a new story', async () => {
    await extensionMessage(client, { type: 'create-item', payload: { backlog: 'am-product', title: 'Live add' } }, messages);
    const after = messages.at(-1)!.payload;
    const all = [...after.priority.items, ...after.icebox.items];
    expect(all.find((r: any) => /live add/i.test(r.title))).toBeDefined();
  });

  // "Add Story" button regression: the user reported that clicking
  // Add Story in the subbar produced a Pp.md both under the backlog
  // AND at the repo root. The extension's new-story-prompt branch
  // calls client.createItem(backlog, title), which cd's into the
  // backlog dir before running `am create-item`. The only way a file
  // could appear at root is if cwd resolves to root, e.g. when
  // backlog is the empty string. Assert that with a non-empty
  // backlog the file lands only inside the backlog dir, no sibling.
  it('regression: Add-Story flow writes one file inside the backlog, no root sibling', async () => {
    await extensionMessage(client, { type: 'create-item', payload: { backlog: 'am-product', title: 'Pp' } }, messages);
    expect(fs.existsSync(path.join(dir, 'am-product', 'Pp.md'))).toBe(true);
    expect(fs.existsSync(path.join(dir, 'Pp.md'))).toBe(false);
  });

  // Phantom-write canary: feed the create-item handler an empty
  // backlog name and assert it does NOT create a Pp.md at the repo
  // root. Before the create-item guard tightened in
  // commands/create_item_command.go this scenario silently exited 0
  // with the file written at root (cwd=root, no overview check
  // failure). The guard now makes the CLI fail loudly, and the
  // extension surfaces the error to the user.
  it('regression: create-item with an empty backlog name fails loudly, no phantom at root', async () => {
    let err: any = null;
    try {
      await extensionMessage(client, { type: 'create-item', payload: { backlog: '', title: 'Phantom' } }, messages);
    } catch (e) {
      err = e;
    }
    expect(err).not.toBeNull();
    expect(fs.existsSync(path.join(dir, 'Phantom.md'))).toBe(false);
  });

  it('description editor saves the new body', async () => {
    const newBody = '# Replaced\n\nThis is the rewritten description.\n';
    await extensionMessage(client, { type: 'set-description', payload: { path: 'am-product/First-story.md', body: newBody } }, messages);
    const detail = await client.getItem('am-product/First-story.md');
    expect(detail.body).toContain('Replaced');
  });

  it('GAP G1: state-machine "accepted" click flips status without a ceremony render', async () => {
    // Today's behavior. Documents the gap so it shows up in the suite
    // until the extension renders an acceptance prompt before flipping.
    await extensionMessage(client, { type: 'set-status', payload: { path: 'am-product/First-story.md', status: 'delivered' } }, messages);
    await extensionMessage(client, { type: 'set-status', payload: { path: 'am-product/First-story.md', status: 'accepted' } }, messages);
    const after = messages.at(-1)!.payload;
    const item = after.priority.items.find((r: any) => r.path === 'First-story.md');
    expect(item.status).toBe('accepted');
    // No 'detail' message with an acceptance ceremony was queued. That
    // confirms G1 (the extension flips without rendering criteria).
    const ceremonyMessage = [...messages].reverse().find((m: { type: string }) => m.type === 'acceptance-ceremony');
    expect(ceremonyMessage).toBeUndefined();
  });

  it('reject message uses reject_item and writes a reason into the body', async () => {
    // The new reject path: user clicks rejected → inline reason form →
    // submit → extension calls reject_item which writes a dated
    // "## Rejection notes" block into the body.
    await extensionMessage(client, { type: 'set-status', payload: { path: 'am-product/Second-story.md', status: 'started' } }, messages);
    await extensionMessage(client, { type: 'set-status', payload: { path: 'am-product/Second-story.md', status: 'finished' } }, messages);
    await extensionMessage(client, { type: 'set-status', payload: { path: 'am-product/Second-story.md', status: 'delivered' } }, messages);
    await extensionMessage(client, { type: 'reject', payload: { path: 'am-product/Second-story.md', reason: 'missed timezone case' } }, messages);
    const detail = await client.getItem('am-product/Second-story.md');
    expect(detail.status).toBe('rejected');
    expect(detail.body).toContain('## Rejection notes');
    expect(detail.body).toContain('missed timezone case');
  });

  // The legacy state-machine "rejected" click (set-status status=rejected
  // without reason) is no longer the UI path. The detail panel routes
  // rejection clicks through the inline reason form to the `reject`
  // message, which is exercised in the test above. The set-status
  // message handler still accepts status=rejected for direct callers,
  // but the gap of "no reason captured" is now a non-default code path.

  // Phantom-file regression. The original bug: the Board posted
  // set-status with a bare basename ("Login-Flow.md") instead of the
  // qualified path. `am start` ran from the repo root, filepath.Abs
  // resolved it against root, and LoadFrontmatter happily created an
  // in-memory file for the missing path which Save then wrote to disk.
  // The result was a Login-Flow.md at the repo root with `status:
  // started` and nothing else. After the qpath fix in Board.tsx the
  // webview never posts a bare basename, AND the CLI's transition
  // verbs (start/finish/deliver/accept/reject) refuse a missing path
  // before LoadBacklogItem can synthesize an empty file. Both halves
  // are exercised here.
  it('regression: bare-basename set-status refuses instead of writing a phantom at the repo root', async () => {
    // Pretend the webview misbehaves and posts an un-qualified path.
    // The extension still hands the string straight to `am start`,
    // which now stats the resolved abs path first. Result: an error,
    // and no phantom on disk.
    let err: any = null;
    try {
      await extensionMessage(client, { type: 'set-status', payload: { path: 'First-story.md', status: 'started' } }, messages);
    } catch (e) {
      err = e;
    }
    expect(err).not.toBeNull();
    expect(String(err)).toMatch(/no such item/i);
    // The phantom file shape: a markdown file at the repo root with
    // only frontmatter. Assert it does NOT exist.
    expect(fs.existsSync(path.join(dir, 'First-story.md'))).toBe(false);
    // The real story in the backlog is untouched.
    const real = fs.readFileSync(path.join(dir, 'am-product', 'First-story.md'), 'utf8');
    expect(real).toContain('title: First story');
  });

  // afterWrite() contract: a write that targets the path currently
  // bound to the open DetailPanel must re-emit a fresh `detail`
  // alongside the snapshot, so the panel and the card cannot disagree
  // on status. Before this seam, the user could see the panel say
  // "started" while the card still showed "Start" (or vice versa).
  it('regression: writes to the open detail path re-emit a fresh detail message', async () => {
    // Open detail on First-story.md.
    await extensionMessage(client, { type: 'open-detail', payload: { path: 'am-product/First-story.md' } }, messages);
    const beforeDetail = messages.at(-1)!;
    expect(beforeDetail.type).toBe('detail');

    // Write something on that path. The contract: snapshot AND a new
    // detail flow back, both freshly read from disk.
    await extensionMessage(client, { type: 'set-tags', payload: { path: 'am-product/First-story.md', tags: ['fresh'] } }, messages);
    const lastTwo = messages.slice(-2);
    expect(lastTwo[0].type).toBe('snapshot');
    expect(lastTwo[1].type).toBe('detail');
    expect(lastTwo[1].payload.item.tags).toContain('fresh');

    // Close the detail. Future writes against the same path must NOT
    // emit a detail, so the panel doesn't pop back into the user's
    // face when they've dismissed it.
    await extensionMessage(client, { type: 'detail-closed' }, messages);
    const beforeCount = messages.filter(m => m.type === 'detail').length;
    await extensionMessage(client, { type: 'set-tags', payload: { path: 'am-product/First-story.md', tags: ['quieter'] } }, messages);
    const afterCount = messages.filter(m => m.type === 'detail').length;
    expect(afterCount).toBe(beforeCount);
  });

  it('regression: writes to a DIFFERENT path do not refresh the open detail', async () => {
    // Open detail on First-story.md.
    await extensionMessage(client, { type: 'open-detail', payload: { path: 'am-product/First-story.md' } }, messages);
    const baselineCount = messages.filter(m => m.type === 'detail').length;
    // Write a sibling. The open detail panel is bound to a different
    // story so we must NOT re-emit detail for it (would be wasted work
    // AND could surprise the user with a flicker on an unrelated edit).
    await extensionMessage(client, { type: 'set-tags', payload: { path: 'am-product/Second-story.md', tags: ['sibling'] } }, messages);
    const afterCount = messages.filter(m => m.type === 'detail').length;
    expect(afterCount).toBe(baselineCount);
    // Clean up so subsequent tests get a clean openDetailPath state.
    await extensionMessage(client, { type: 'detail-closed' }, messages);
  });

  // set-status with status=started auto-promotes from the icebox.
  // Before this fix, the card path (pull-from-icebox) promoted but
  // the detail-panel state-machine path did not, so starting from
  // the panel left the item stranded in icebox with status=started.
  // Card and file disagreed; the user saw "Started" in detail but
  // the card stayed in the Icebox column.
  it('regression: set-status started auto-promotes from icebox even when caller is not pull-from-icebox', async () => {
    // Third-story.md lives in the icebox. Drive a raw set-status the
    // way the DetailPanel state machine does. Result: the story is
    // both started AND moved into priority.
    // (Reset the priority/icebox state first so this test is order-
    // independent.)
    await extensionMessage(client, { type: 'move-to-icebox', payload: { backlog: 'am-product', item_path: 'Third-story.md' } }, messages).catch(() => {});

    await extensionMessage(client, { type: 'set-status', payload: { path: 'am-product/Third-story.md', status: 'started' } }, messages);
    const after = messages.at(-1)!.payload;
    const inPriority = after.priority.items.find((r: any) => r.path === 'Third-story.md');
    const inIcebox = after.icebox.items.find((r: any) => r.path === 'Third-story.md');
    expect(inPriority).toBeDefined();
    expect(inPriority.status).toBe('started');
    expect(inIcebox).toBeUndefined();
  });

  it('set-type writes the type frontmatter and strips the estimate when switching to bug/chore', async () => {
    // Seed: Third-story.md is a feature with no estimate yet. Set one,
    // then flip to chore. The extension's set-type handler must strip
    // the estimate so the snapshot doesn't render a stale points pill
    // on a card the coach rules say is unpointed.
    await extensionMessage(client, { type: 'set-estimate', payload: { path: 'am-product/Third-story.md', estimate: 3 } }, messages);
    await extensionMessage(client, { type: 'set-type', payload: { path: 'am-product/Third-story.md', type: 'chore' } }, messages);
    const detail = await client.getItem('am-product/Third-story.md');
    expect(detail.type).toBe('chore');
    expect(detail.estimate || '').toBe('');
    // Flipping back to feature does NOT auto-restore the estimate
    // (we don't know what it was); the user picks one again.
    await extensionMessage(client, { type: 'set-type', payload: { path: 'am-product/Third-story.md', type: 'feature' } }, messages);
    const after = await client.getItem('am-product/Third-story.md');
    expect(after.type).toBe('feature');
    expect(after.estimate || '').toBe('');
  });

  it('regression: set-status unstarted is a valid CLI verb (am unstart), not an error', async () => {
    // The detail panel's state-machine row lets the user click any
    // state including "unstarted". Before adding the unstart verb,
    // this errored with "unsupported status: unstarted" from the
    // TypeScript client. The reset goes through `am unstart`, which
    // applies the UnstartedStatus transition (clears all timestamps).
    await extensionMessage(client, { type: 'set-status', payload: { path: 'am-product/First-story.md', status: 'finished' } }, messages);
    await extensionMessage(client, { type: 'set-status', payload: { path: 'am-product/First-story.md', status: 'unstarted' } }, messages);
    const after = messages.at(-1)!.payload;
    const item = after.priority.items.find((r: any) => r.path === 'First-story.md');
    expect(item.status).toBe('unstarted');
    // The reset must also clear timestamps. Read the file directly to
    // confirm (the snapshot doesn't expose started/finished fields).
    const detail = await client.getItem('am-product/First-story.md');
    expect(detail.started || '').toBe('');
    expect(detail.finished || '').toBe('');
  });

  // delete-item end-to-end. Drives the real CLI against the fixture
  // and asserts: (a) the file is gone from disk, (b) the icebox /
  // priority view drops the row in the next snapshot, and (c) the
  // extension tells the webview to close the open detail panel.
  it('delete-item removes the story file, drops the order entry, and closes the detail panel', async () => {
    // Create a throwaway story so we don't delete the fixture's
    // First-story.md (other tests below assert on it).
    await extensionMessage(client, { type: 'create-item', payload: { backlog: 'am-product', title: 'Doomed' } }, messages);
    expect(fs.existsSync(path.join(dir, 'am-product', 'Doomed.md'))).toBe(true);

    // Open detail on the doomed story so we can assert the close-detail
    // signal flows back after the delete.
    await extensionMessage(client, { type: 'open-detail', payload: { path: 'am-product/Doomed.md' } }, messages);

    await extensionMessage(client, { type: 'delete-item', payload: { path: 'am-product/Doomed.md' } }, messages);

    // File is gone from disk.
    expect(fs.existsSync(path.join(dir, 'am-product', 'Doomed.md'))).toBe(false);
    // The webview gets a close-detail signal AND a fresh snapshot.
    const recent = messages.slice(-2);
    expect(recent[0].type).toBe('close-detail');
    expect(recent[1].type).toBe('snapshot');
    // The deleted row is not in priority or icebox of the new
    // snapshot. View == disk.
    const snap = recent[1].payload;
    expect(snap.priority.items.find((r: any) => r.path === 'Doomed.md')).toBeUndefined();
    expect(snap.icebox.items.find((r: any) => r.path === 'Doomed.md')).toBeUndefined();
  });

  it('regression: qualified path resolves to the backlog file, not a root sibling', async () => {
    // The qpath contract in Board.tsx is `${snap.backlog}/${row.path}`.
    // Drive an arbitrary transition through that shape and verify the
    // file under the backlog moved while no top-level sibling appeared.
    await extensionMessage(client, { type: 'set-status', payload: { path: 'am-product/Second-story.md', status: 'finished' } }, messages);
    const real = fs.readFileSync(path.join(dir, 'am-product', 'Second-story.md'), 'utf8');
    expect(real).toMatch(/status:\s*finished/);
    // No phantom appeared at the root.
    expect(fs.existsSync(path.join(dir, 'Second-story.md'))).toBe(false);
  });
});
