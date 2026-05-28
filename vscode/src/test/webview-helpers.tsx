// @vitest-environment happy-dom
//
// Shared helpers for React component tests in the webview. Two jobs:
// 1. Stub the `vscode` postMessage API so component tests can assert
//    on the exact payloads the UI emits without booting VS Code.
// 2. Build small fixture snapshots that mirror what extension.ts
//    actually sends on the wire, so tests stay coupled to the contract
//    rather than to private component state.
//
// vscode-api.ts loads `acquireVsCodeApi()` at module-import time. The
// production webview is sandboxed and that global exists; tests run in
// happy-dom where it does not. We install a stub on `globalThis` BEFORE
// the component module imports `../vscode-api`. Test files therefore
// import this helper before any component under test.

import { vi } from 'vitest';
import * as React from 'react';

export interface CapturedMessage {
  type: string;
  payload?: any;
}

export interface WebviewHarness {
  messages: CapturedMessage[];
  /** Most recent message, or undefined if none have been posted. */
  last(): CapturedMessage | undefined;
  /** All messages of a given type, in order. */
  ofType(type: string): CapturedMessage[];
  /** Reset the capture log between scenarios within a single test. */
  reset(): void;
}

declare global {
  // eslint-disable-next-line no-var
  var __amHarness: WebviewHarness | undefined;
}

// Install the stub exactly once per process. Module-level so the shape
// is in place before any component file imports vscode-api.
if (!(globalThis as any).acquireVsCodeApi) {
  const harness: WebviewHarness = {
    messages: [],
    last() { return this.messages[this.messages.length - 1]; },
    ofType(t: string) { return this.messages.filter(m => m.type === t); },
    reset() { this.messages.length = 0; },
  };
  globalThis.__amHarness = harness;
  (globalThis as any).acquireVsCodeApi = () => ({
    postMessage: (msg: CapturedMessage) => { harness.messages.push(msg); },
    getState: () => undefined,
    setState: () => {},
  });
}

export function harness(): WebviewHarness {
  if (!globalThis.__amHarness) throw new Error('webview harness not initialized');
  return globalThis.__amHarness;
}

// makeOrderRow returns a minimal but realistic OrderRow. Fields match
// the JSON contract emitted by mcpserver/priority.go so test rows look
// like what a real snapshot carries.
export function makeOrderRow(over: Partial<any> = {}): any {
  return {
    index: 0,
    title: 'Login Flow',
    path: 'Login-Flow.md',
    status: 'unstarted',
    estimate: '',
    type: 'feature',
    assignees: [],
    tags: [],
    blocked: false,
    comment_count: 0,
    epic: '',
    ...over,
  };
}

// makeSnapshot returns a Snapshot shaped like extension.ts emits.
// `backlog` defaults to "product" so the qpath fix produces a stable
// "product/Login-Flow.md" prefix in assertions.
export function makeSnapshot(over: Partial<any> = {}): any {
  const priority = over.priority ?? { items: [], count: 0, velocity: 10 };
  const icebox   = over.icebox   ?? { items: [], count: 0 };
  const done     = over.done     ?? { items: [], count: 0 };
  return {
    backlog: 'product',
    backlogs: ['product'],
    priority,
    icebox,
    done,
    dashboard: null,
    velocity: { rows: [] },
    typeMix: { rows: [], total: 0 },
    burnup: null,
    cfd: null,
    epics: [],
    ...over,
  };
}

// makeDetail returns a Detail shaped like the 'detail' message payload.
// `item.path` follows the qualified form `<backlog>/<basename>` because
// the production webview's open-detail flow qualifies before posting.
export function makeDetail(over: Partial<any> = {}): any {
  const item = {
    path: 'product/Login-Flow.md',
    title: 'Login Flow',
    type: 'feature',
    status: 'unstarted',
    estimate: '',
    assignees: [] as string[],
    tags: [] as string[],
    blocked: false,
    epic: '',
    hypothesis: '',
    body: '',
    iteration: 0,
    iteration_label: '',
    ...(over.item || {}),
  };
  // Carry over any top-level fields the caller supplied (e.g.
  // `autoAction`) so detail-shape extensions can be tested without
  // patching this helper every time the shape grows. `item` /
  // `comments` / `tasks` / `history` are merged explicitly above so
  // partial overrides don't blow away the rest of the structure.
  const { item: _i, comments: _c, tasks: _t, history: _h, ...extras } = (over || {}) as any;
  return {
    item,
    comments: over.comments ?? { comments: [], count: 0 },
    tasks: over.tasks ?? { tasks: [], count: 0 },
    history: over.history ?? { entries: [], count: 0 },
    ...extras,
  };
}

// expectMessage finds the most recent message of `type` and asserts on
// its payload. Returns the message for further chained assertions.
export function lastOf(type: string): CapturedMessage {
  const msgs = harness().ofType(type);
  if (msgs.length === 0) {
    throw new Error(`no message of type "${type}" was posted; saw: ${harness().messages.map(m => m.type).join(', ') || '(none)'}`);
  }
  return msgs[msgs.length - 1];
}

// Re-export React so test files don't need their own import dance for
// the happy-dom JSX namespace.
export { React };
