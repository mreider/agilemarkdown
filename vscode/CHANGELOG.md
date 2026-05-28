# Changelog

## 0.1.4

- New foundation field `started:` is stamped automatically on the
  first transition out of `unstarted`, preserved across rejected /
  restart cycles, and cleared by a full reset to `unstarted`. Schema,
  `am start` and forward transitions back-fill it when skipping
  states. Enables cycle time from started to accepted and the
  in-flight band of the cumulative-flow chart.
- 4-band cumulative flow: `am cfd --json` and the Analytics chart now
  carry accepted / in-flight / backlog counts per day.
- Per-item History panel in DetailPanel, fed by a new
  `am history ITEM --json` verb that wraps `git log --follow`.
- Search: ⌘K-focused input in the subbar, substring search across
  title / tags / path / body with ranked hits and snippets, backed by
  `am search QUERY --json` and the new `search` MCP tool.
- Filter state machinery: the internal sidebar's Views and Labels
  rows are now clickable. "My work" filters by the current git user
  (new `am whoami --json` verb). Labels are aggregated from tags
  across priority + icebox.
- Per-item Iteration field in DetailPanel. Server-side number from
  timestamps; the extension fills in the band for unstarted items
  using priority position + rolling velocity.
- Reporter row in DetailPanel pulled from item frontmatter `author:`.
- Help button in the subbar opens the agilemarkdown docs site.

## 0.1.3

- Switched from a long-lived `am mcp` child to per-verb shell-out: each
  panel action runs a single `execFile` against the `am` CLI. No JSON-RPC
  framing, no initialize handshake, no swallowed errors. Requires `am`
  with the per-verb JSON surface (`--json` on read views plus
  `list-backlogs`, `list-items`, `get-item`, `get-comments`, `type-mix`,
  `set-description`).
- Welcome view state machine: `noFolder` (open a folder),
  `uninitialized` (initialize Agile Markdown here, which runs `git init`
  then `am init`), and `ready` (open the board, new backlog). A
  filesystem watcher on `.am/config.yaml` flips the state automatically.
- Pre-React shell so a blank panel can never happen silently. If the
  webview bundle fails to load (CSP, bad path, corrupt bundle), the
  panel shows a "Loading…" line and surfaces any thrown error.
- Build fix: `npm run package` now runs `webpack --mode production`
  before `vsce package`, so the vsix can no longer ship with stale
  `dist/` bundles.

## 0.1.0

- Initial release.
- Read-only board with Current, Icebox, and Epics columns sourced from
  `priority_list`, `icebox_list`, and `epic_progress`.
- Detail panel: state machine, points picker, owner editor, tags,
  epic, description, tasks, comments, blocked toggle.
- Drag-and-drop reorder within priority, plus moves between priority
  and icebox.
- Analytics tab: KPI cards from `dashboard`, velocity chart from
  `velocity_history`, story type mix from `type_mix`.
- CLI auto-detect: PATH, `~/go/bin/am`, `~/.am/bin/am`. Prompts to
  download the matching binary from the latest GitHub release if
  nothing is found.
- Multi-owner stories: up to three assignees per story.
- Single MCP session per workspace. All reads and writes go through
  the `am mcp` server.
