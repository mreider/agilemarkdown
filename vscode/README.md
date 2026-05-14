# Agile Markdown for VS Code

A Pivotal-style backlog board, detail panel, and analytics for any
agilemarkdown repo. Drives the project through MCP, so every action is
also reachable from the `am` CLI and from any other MCP client.

## Requirements

- VS Code 1.85 or later
- The `am` CLI v4.3.0 or later. The extension prompts to download it on
  first run if it is not on PATH.

## Features

- Read-only board with priority, icebox, and epic columns
- Story detail panel: state machine, points picker, owner editor,
  tags, epic, description, tasks, comments, blocked toggle
- Drag-and-drop within priority and between priority and icebox
- Analytics view: KPI cards from `dashboard`, velocity chart from
  `velocity_history`, story type mix from `type_mix`
- Multi-owner: assigns up to three names per story

## Settings

| Setting | Default | Notes |
|---|---|---|
| `agilemarkdown.cliPath` | _(empty)_ | Absolute path to `am`. When empty, the extension searches PATH, `~/go/bin/am`, `~/.am/bin/am`, then prompts. |

## Marketplace

The same `.vsix` ships to both the Microsoft Marketplace (publisher
`mreider`) and the Open VSX Registry. CI runs on `vscode-v*` tags.

The publish workflow needs two repository secrets:

- `VSCE_PAT`: Azure DevOps PAT with `Marketplace > Manage` scope on the
  `mreider` publisher.
- `OVSX_PAT`: open-vsx.org token for the `mreider` namespace.

## Development

```
cd vscode
npm install
npm run build
```

Open the `vscode/` folder in VS Code and press F5 to launch a host
window with the extension loaded.

## Architecture

The extension spawns a single `am mcp` child process per workspace and
holds a stdio JSON-RPC session over its lifetime. Every read and write
is an MCP tool call. There is no direct file IO from the extension.
