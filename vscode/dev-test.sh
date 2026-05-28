#!/usr/bin/env bash
# Prepares the Agile Markdown VS Code extension for a local F5 test.
#
# What it does:
#   1. Builds the agilemarkdown CLI binary so the extension's MCP
#      session has a working `am`.
#   2. Installs npm deps + runs the webpack production build so
#      dist/extension.js and dist/webview.js are fresh.
#   3. Builds a deterministic test workspace at /tmp/am-vscode-test
#      with one backlog ("notes"), three stories at points 3/3/3, real
#      acceptance criteria in each body, an inception.md, and a tiny
#      team-agreements.md. The workspace is wiped and rebuilt every
#      run.
#   4. Prints the next-step instructions for the F5 launch.
#
# After this script:
#   - Open the agilemarkdown repo (the directory containing this file's
#     parent) in VS Code.
#   - Inside VS Code, press F5 (or Run > Start Debugging). VS Code spawns
#     the Extension Development Host window.
#   - In the dev host window, open the prepared workspace at
#     /tmp/am-vscode-test (File > Open Folder...).
#   - Cmd+Shift+P (or Ctrl+Shift+P) -> "Agile Markdown: Open Board".
#
# Usage:
#   bash vscode/dev-test.sh
#
# Optional env:
#   AM_BIN  path to the am binary (default: $REPO_ROOT/agilemarkdown)
#   WORK    workspace directory (default: /tmp/am-vscode-test)

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${HERE}/.." && pwd)"
AM_BIN="${AM_BIN:-${REPO_ROOT}/agilemarkdown}"
WORK="${WORK:-/tmp/am-vscode-test}"

step() { printf "\n→ %s\n" "$*"; }

step "Building agilemarkdown CLI"
cd "${REPO_ROOT}"
go build -o "${AM_BIN}" .
"${AM_BIN}" --version

step "Installing extension deps"
cd "${HERE}"
if [ ! -d node_modules ]; then
  npm install
else
  echo "(node_modules present; skipping npm install)"
fi

step "Building extension bundle (webpack)"
npx webpack --mode production

step "Type checking"
npx tsc -p . --noEmit

step "Packaging VSIX"
rm -f agilemarkdown-*.vsix
npx vsce package --out agilemarkdown.vsix
VSIX="${HERE}/agilemarkdown.vsix"

step "Building test workspace at ${WORK}"
rm -rf "${WORK}"
mkdir -p "${WORK}"
cd "${WORK}"

git init -q
git config user.email pm@example.com
git config user.name "PM"

# Drop the coach files + sample backlog. am init writes the four
# projections, the canonical body, the five skills, the hook, and
# settings.json.
"${AM_BIN}" init >/dev/null

# Seed the inception so the dev pair has context.
cat > inception.md <<'INCEPTION'
# Inception

## The user

A solo developer who keeps daily notes in a markdown vault and runs
everything from the keyboard.

## The goal

A command palette inside the notes app. cmd-k opens the palette and
the user can jump to a note, run a recent search, or trigger a small
set of actions without touching the mouse.

## The reason

The vault is at 1200 notes. Mouse navigation no longer scales.

## Success

End-of-day usage shows the palette opens at least 5 times a session
and the median action latency is below 800 ms.

## Constraints

- one-person team
- no third-party search dependency
- no breaking changes to existing keybindings

## Out of scope

- multi-vault support
- web/cloud version
- mobile palette UI
INCEPTION

# Backlog + three stories with concrete acceptance criteria.
"${AM_BIN}" create-backlog notes >/dev/null
rm -f notes/Sample-*.md

write_story() {
  local title="$1"; shift
  local file="${title// /-}.md"
  cd "${WORK}/notes"
  "${AM_BIN}" create-item "${title}" >/dev/null
  "${AM_BIN}" estimate "${file}" 3 >/dev/null
  python3 - "${WORK}/notes/${file}" "$@" <<'PY'
import sys, re
fp = sys.argv[1]
bullets = sys.argv[2:]
s = open(fp).read()
ac = "\n## Acceptance\n\n" + "\n".join(f"- {b}" for b in bullets) + "\n"
s = s.replace("## Acceptance\n\n- (replace with a concrete, PM-checkable bullet)\n- (one user-visible behavior per bullet)\n", ac.lstrip("\n"))
open(fp, "w").write(s)
PY
  cd "${WORK}"
}

write_story "Open palette on cmd-k" \
  "cmd-k opens the palette overlay" \
  "the palette captures keyboard focus on open" \
  "escape closes it"

write_story "Fuzzy match note titles" \
  "typing 'wkly' returns 'weekly notes' in the top three" \
  "search latency p95 under 300 ms" \
  "empty query shows recent notes"

write_story "Recent searches dropdown" \
  "recent searches show below the input on focus" \
  "clicking a recent re-runs it" \
  "history caps at 10 entries"

# Promote into priority so the iteration plan has content to render.
"${AM_BIN}" sync </dev/null >/dev/null
cd "${WORK}/notes"
"${AM_BIN}" unice Open-palette-on-cmd-k.md --top >/dev/null
"${AM_BIN}" unice Fuzzy-match-note-titles.md >/dev/null
"${AM_BIN}" unice Recent-searches-dropdown.md >/dev/null

# A tiny seed of working agreements so the coach surface has something.
cd "${WORK}"
"${AM_BIN}" team-agreements --add "Bugs go at the top of priority unless intentionally deprioritized." >/dev/null
"${AM_BIN}" team-agreements --add "Every feature carries an `## Acceptance` section before pulling." >/dev/null

cat <<MSG

✓ Test workspace ready at: ${WORK}
✓ VSIX built at: ${VSIX}

To install the packaged extension (no F5 needed):

  code --install-extension "${VSIX}" --force

Then open ${WORK} in VS Code and run "Agile Markdown: Open Board".

Or, for F5 dev-host iteration:

  1. Open the extension folder in VS Code (NOT the repo root):
       code "${HERE}"

     The launch config at vscode/.vscode/launch.json points
     debugger args at this folder. Opening the repo root will
     not surface the F5 launch.

  2. Press F5 (Run → Start Debugging). VS Code runs the build
     task and then spawns an "Extension Development Host" window
     with the prepared workspace ${WORK} already opened.

  3. Cmd+Shift+P (or Ctrl+Shift+P) → run:
       Agile Markdown: Open Board

What to verify (each interaction should re-render text on screen):

  • Stories panel shows three cards in Current. Click one → detail
    panel opens with Acceptance bullets at the top.
  • Click a points pip (1, 2, 3, 5, 8). Card's points badge updates.
  • Owners input "alice, bob" then blur. Avatars render on the card.
  • Tags input "q3, palette" then blur. Label chips appear.
  • Click "Block" → enter "waiting on design" → blocked badge appears.
  • Click "Blocked · click to clear" → badge clears.
  • Click "started" on the state machine → status pill flips.
  • Click "delivered" → flips. Click "accepted" → inline PM ceremony
    panel opens with the criteria + Yes / Cancel. Click Yes → flips.
  • For another story: click "rejected" → inline reason textarea →
    type a reason → Reject with reason. Body picks up "## Rejection
    notes" with the reason.
  • Drag a card from Current down or up. Order in _priority.md updates.
  • Drag a card from Current into Icebox. _icebox.md picks it up.
  • Drag from Icebox back into Current.
  • Click "+" in the Current column header → inline title input → type
    a title → Enter. The new card appears immediately (the create_item
    tool now stages into icebox automatically).
  • Click the "Plan" tab in the topbar. The iteration plan renders
    with Committed + Below the line + Warnings.
  • Click the "Analytics" tab. KPI cards render from the dashboard
    tool.

If anything does not re-render after an interaction, check the VS Code
Developer Tools (Help → Toggle Developer Tools) for webview console
errors, and check the "Agile Markdown" output channel for extension
messages.

To rebuild after edits:
  bash $(pwd)/dev-test.sh

MSG
