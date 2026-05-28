# Laboratory plan

A small hosted deployment of agilemarkdown that demonstrates every concept
three ways simultaneously, web UI, CLI terminal, MCP console, backed by the
real `am` binary. Same React tree as the VS Code extension, zero duplication.
Single static binary. Drives the tutorial screenshot pipeline and doubles as a
"try agilemarkdown without installing VS Code" deployment for end users.

This document is the source of truth for the build. Decisions noted inline.

## Goal

Three-channel pedagogy. For every concept in the tutorial, the same outcome
is reached three different ways and shown side by side. The reader sees:

  1. The web UI (Playwright screenshot)
  2. The CLI verb (vhs terminal recording)
  3. The MCP JSON-RPC dialog (captured request + response)

Same concept, three frames, every time. The laboratory is the host that
generates all three from one running backend so they stay coherent.

## What ships

A new `am serve` subcommand on the existing CLI:

```
am serve --port 8080 --root /path/to/project
```

Embeds a web bundle. Exposes HTTP + WebSocket. Watches the project. Hosts
three browser panes:

  - **Board** (top): the existing React tree from `vscode/src/webview/`.
    Search / Backlog / Done / Icebox / Epics columns. StoryCard, DetailPanel,
    NewStoryModal, filter columns, every existing component. Unchanged.
  - **Bottom panel** (tabs): one of {CommandLog, MCP Console, Terminal}
    visible at a time. Tabs were chosen over a split for screenshot clarity.
  - **Live mutating backend**: every UI action hits the real `am` semantics,
    file watcher broadcasts the resulting snapshot, every other connected
    client sees the change.

The web build reuses the React tree wholesale. No components are forked.

## Decisions (answers to the five planning questions)

  1. **Read-write, no auth.** It's a lab. Anyone with the URL can mutate
     state. Single visitor at a time is the assumed mode.
  2. **Real terminal in v1.** xterm.js client + `creack/pty` server-side.
     Pseudo-terminal attached to a bash shell rooted at the project dir.
  3. **Fixture resets on every server start.** `am serve` wipes the
     workspace and reseeds from `scripts/build-demo.sh` so the lab always
     boots into the same deterministic 54-story state. Screenshots stay
     reproducible.
  4. **Bottom panel uses tabs.** CommandLog / MCP / Terminal, one at a
     time. Easier to screenshot one channel at a time without sibling
     panes leaking into the frame.
  5. **One shared MCP child process.** A single `am mcp` is spawned on
     start; the server multiplexes `/ws/mcp` connections by request id.
     Saves memory; simpler lifecycle.

## Repo layout

```
cmd/am-server/
  main.go              entry, flags, embed.FS over vscode/dist/web/
  routes.go            HTTP handler table
  ws.go                /ws/snapshot, /ws/log, /ws/mcp, /ws/pty
  watcher.go           fsnotify-based debounced snapshot push (200ms)
  log.go               structured CLI-equivalent log emitter
  mcp_bridge.go        single am mcp child, request multiplexer
  pty_bridge.go        creack/pty backed terminal channel
  fixture.go           wipes + re-seeds /path/to/project on startup

commands/serve_command.go   am serve wiring

vscode/src/webview/    UNCHANGED, the React tree
vscode/src/web/        NEW, web-mode bootstrap and shim only
  index.tsx
  web-api.ts           re-export module; webpack resolves to one impl
  web-api.impl.ts      HTTP + WebSocket implementation
  LabLayout.tsx        composes <Board/> + <BottomPanel/>
  BottomPanel.tsx      tab switcher
  CommandLog.tsx       reads /ws/log
  McpConsole.tsx       reads tool manifest, POSTs /api/mcp/*
  Terminal.tsx         xterm.js wrapper, reads /ws/pty

vscode/webpack.config.js       extension target (unchanged)
vscode/webpack.web.config.js   web target, same React source

deploy/
  Caddyfile.example
  am-serve.service              systemd unit
  bringup.sh                    droplet bootstrap script
```

## React-tree reuse (no duplication)

The components in `vscode/src/webview/` import `vscode` from
`./vscode-api`. Refactor that single module to a re-export:

```ts
// vscode/src/webview/vscode-api.ts
import { impl } from './vscode-api.impl';
export const vscode = impl;
export type * from './vscode-api.types';
```

Two impl files, one per build target. Webpack picks one via `resolve.alias`:

```ts
// vscode/src/webview/vscode-api.impl.ts        (extension)
export const impl = acquireVsCodeApi();

// vscode/src/web/web-api.impl.ts                (web, aliased over the
//                                                above by webpack.web.config.js)
export const impl = createHttpWsShim();
```

Components in `vscode/src/webview/` continue to compile against both targets
with no changes. A bugfix to Board.tsx lands in the extension and the
laboratory at once.

## Backend HTTP surface

Every route wraps an existing function in `mcpserver/api.go` or a single
CLI verb. JSON in, JSON out. No auth.

```
GET  /api/snapshot              priority + icebox + done + dashboard + epics + velocity
GET  /api/mcp-tools             registry of tool name → arg schema (for the Console)
GET  /api/me                    current git user

POST /api/set-status            {path, status}
POST /api/rank                  {item_path, top|bottom|after|before}
POST /api/ice                   {item_path}
POST /api/unice                 {item_paths[], top|bottom|after}
POST /api/move-to-icebox
POST /api/move-to-priority
POST /api/reject                {path, reason, author}
POST /api/comment               {path, text, author}
POST /api/task                  {path, text} or {path, idx} for tick
POST /api/create-item           {backlog, title, target}
POST /api/set-type
POST /api/set-release-date
POST /api/set-estimate
POST /api/set-assigned
POST /api/set-tags
POST /api/set-epic
POST /api/block / /unblock
POST /api/delete-item
POST /api/sync                  runs am sync (regenerate + commit + push)
POST /api/mcp/:tool             generic MCP tool passthrough (drives McpConsole)
```

WebSocket channels:

```
/ws/snapshot       full snapshot pushed on file-watcher event, 200ms debounce
/ws/log            CommandLog entries (one per write)
/ws/mcp            JSON-RPC frames to/from the single am mcp child process
/ws/pty            xterm.js terminal bytes ↔ pty
```

## Bottom panel, three tabs

### 1. CommandLog

Read-only terminal-styled stream. Every write route on the server emits a
structured event before responding:

```json
{
  "ts": "2026-05-21T13:42:11Z",
  "verb": "am start stories/wait-by-the-mailbox-for-any-kind-of-reply.md",
  "ok": true,
  "summary": "status: started   started_at: 2026-05-21T13:42:11Z"
}
```

Renders as `$ <verb>` followed by the summary, ✓ on success, ✗ on failure.
Scrollback. Copyable text.

Purpose: the tutorial reader sees "click in the UI → this CLI verb is
the same thing." Pedagogical bridge between channels.

### 2. MCP Console

Two halves split vertically inside the tab.

**Top:** tool invoker.
  - Dropdown listing every MCP tool (server emits the registry on
    `/api/mcp-tools`, sourced from the same `buildServer` registration
    used by the production MCP server so it stays in sync).
  - Form generated from each tool's argument JSON schema.
  - "Send" button → POST `/api/mcp/:tool` → result JSON pretty-printed
    below.

**Bottom:** wire view toggle. Shows the raw JSON-RPC envelope:

```json
{
  "jsonrpc": "2.0",
  "id": 17,
  "method": "tools/call",
  "params": { "name": "set_status", "arguments": { "path": "...", "status": "started" } }
}
```

next to the rendered result. This is what an AI agent actually sends, 
the tutorial needs this frame to demystify MCP for readers.

### 3. Terminal

xterm.js on the client; `creack/pty` spawning bash on the server. Wired
via `/ws/pty`. CWD is the project root. Users can type `am show priority`,
`am sync`, `git log`, anything. Terminal output is what gets captured by
vhs for the CLI tutorial frames (when running locally) or just used live
for exploration.

The terminal also doubles as the natural "deploy this on your own droplet"
affordance, a hosted agilemarkdown user can drive the project from the
shell without SSH.

## Lifecycle (`am serve`)

  1. Parse flags: `--port`, `--root`, optional `--no-reset` for users
     who want state to persist between restarts.
  2. By default (per decision 3), wipe `--root` and re-seed via
     `scripts/build-demo.sh`. The lab boots into known state every time.
  3. Start `am mcp` as a single child process; keep its stdin/stdout
     ready for the multiplexer.
  4. Start fsnotify watcher rooted at the project dir; debounce events
     by 200ms.
  5. Mount HTTP routes + WS channels.
  6. Serve the embedded web bundle at `/`.
  7. Block on signal.

The wipe-and-reseed happens at startup, not at every snapshot, visitors
can mutate freely during the session.

## Deployment

Single $5/mo DigitalOcean droplet:

```
/usr/local/bin/am                  the binary
/var/lib/am/demo/                   workspace (wiped + reseeded on start)
/etc/systemd/system/am-serve.service
/etc/caddy/Caddyfile                 auto-HTTPS reverse-proxy to localhost:8080
```

`am-serve.service` runs `/usr/local/bin/am serve --port 8080 --root /var/lib/am/demo`.

Caddy fronts with TLS via Let's Encrypt. Cloudflare DNS points at the
droplet IP. Optional Cloudflare Tunnel variant keeps the IP private.

Deploy cycle: `scp am root@droplet:/usr/local/bin/`, `ssh root@droplet systemctl restart am-serve`. The fixture reseeds on each restart so screenshot determinism survives any deploy.

## Three-channel mapping for the tutorial

The tutorial is rewritten later. This is the shape it will take:

| Concept                  | UI                                     | CLI                                                       | MCP                                              |
|--------------------------|-----------------------------------------|-----------------------------------------------------------|--------------------------------------------------|
| Start a story            | Click **Start** on a card               | `am start stories/wait.md`                                | `set_status {path, status: "started"}`           |
| Reject delivered         | Red **Reject** on a delivered card      | `am reject stories/yell.md --reason "..."`                | `reject_item {path, reason, author}`             |
| Search by owner          | Search column, Owner=toad               | `am list-items stories --assignee toad`                   | `list_items {assignee: "toad"}`                  |
| Epic ETA                 | Epics card "ETA Jul 13"                 | `am show epic the-garden-of-patience`                     | `epic_progress {slug}`                           |
| Ship a release           | Click **Ship** on a release bar         | `am accept stories/first-frost.md`                        | `set_status {path, status: "accepted"}`          |
| Done view                | Open Done column                        | `am show done`                                            | `done_list {backlog: "stories"}`                 |
| Current sprint           | Iteration 1 · Current band              | `am show priority`                                        | `priority_list {backlog: "stories"}`             |
| New release marker       | + button → type=release → date picker   | `am create-item ...` + `am set-release-date ... 2026-09-01` | `create_item` + `set_release_date`             |

Roughly ten concepts in the final tutorial. Each one gets three rendered
artifacts; all three are captured deterministically from the same lab
state.

## Phasing

| Phase | Scope                                                         | Effort  |
|-------|---------------------------------------------------------------|---------|
| 1     | `cmd/am-server` skeleton, `/api/snapshot`, `/api/set-status`, `/ws/snapshot`, web build target, mount Board against real backend in a local browser | ~half day |
| 2     | All HTTP write routes; CommandLog panel + `/ws/log`           | ~half day |
| 3     | MCP Console + xterm.js Terminal pane + `/ws/mcp` + `/ws/pty`  | ~one day  |
| 4     | Caddyfile + systemd unit + droplet bring-up                   | ~half day |
| 5     | Playwright spec + vhs tapes + MCP capture script → `docs/tutorial/` | ~half day |

Total: roughly 2.5–3 days of build. Tutorial content rewrite happens after.

## Open dependencies to install

  - Go: `github.com/creack/pty`, `github.com/fsnotify/fsnotify`,
    `github.com/coder/websocket` or `nhooyr.io/websocket`.
  - npm: `xterm`, `@xterm/addon-fit`. Already have `react`, `webpack`.

## Non-goals (out of scope for v1)

  - Multi-user concurrent editing
  - Authentication or per-session workspaces
  - WASM frontend (would let us drop the server entirely; deferred)
  - Hosted PaaS install path (Helm chart, docker-compose, etc.)
  - Persistent state across restarts (fixture resets every boot by design)
