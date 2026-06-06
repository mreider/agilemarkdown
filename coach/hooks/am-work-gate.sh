#!/usr/bin/env bash
# Generalized agilemarkdown work-gate (cwd-independent).
#
# A Claude Code PreToolUse hook on Edit/Write/NotebookEdit that enforces the
# Pivotal rule "no code changes without a pulled story" for ANY am-managed
# repo — regardless of which directory the agent session was launched from.
# `am init` installs this into the user's global ~/.claude so it fires for
# every session; the per-repo coach gate only fires when the session runs
# from that repo. This one keys off the edited file's path instead.
#
# Decision:
#   - Not an Edit/Write/NotebookEdit          -> allow
#   - Edited file is not inside an am repo     -> allow (non-am project)
#   - Edited file is backlog / agent infra     -> allow (managing the work)
#   - The repo has no stories backlog yet      -> allow (not managed yet)
#   - The repo has >=1 story in `started`      -> allow
#   - Otherwise                                -> BLOCK (exit 2)
#
# An am repo is identified by a `.am/` directory at its root; its stories
# live in the directory containing `_priority.md` (root or a sub-folder).
set -euo pipefail

input="$(cat)"

field() {
  printf '%s' "$input" | python3 -c "import json,sys
try: d=json.load(sys.stdin)
except Exception: print(''); sys.exit()
$1" 2>/dev/null || true
}

tool_name="$(field 'print(d.get("tool_name",""))')"
case "$tool_name" in
  Edit | Write | NotebookEdit) ;;
  *) exit 0 ;;
esac

file_path="$(field 'ti=d.get("tool_input",{}); print(ti.get("file_path") or ti.get("notebook_path") or "")')"
[ -n "$file_path" ] || exit 0

# Walk up from the edited file to the nearest directory containing `.am/`.
repo=""
dir="$(dirname "$file_path")"
while [ "$dir" != "/" ] && [ -n "$dir" ]; do
  if [ -d "$dir/.am" ]; then
    repo="$dir"
    break
  fi
  dir="$(dirname "$dir")"
done
# Not inside an am-managed repo — leave it alone.
[ -n "$repo" ] || exit 0

# Always-allowed paths inside the repo (the backlog itself + agent infra,
# which you must be able to edit to manage and pull the work).
case "$file_path" in
  "$repo"/.am/* | "$repo"/.claude/* | "$repo"/.cursor/*) exit 0 ;;
  "$repo"/CLAUDE.md | "$repo"/AGENTS.md | "$repo"/.gitignore) exit 0 ;;
  "$repo"/.github/copilot-instructions.md) exit 0 ;;
esac

# Find the stories directory (the one holding `_priority.md`).
stories="$(find "$repo" -maxdepth 2 -name _priority.md -not -path '*/.am/*' 2>/dev/null | head -1)"
stories="${stories%/_priority.md}"
# A bare `.am/` with no stories backlog isn't a managed backlog yet — don't
# gate it (avoids bricking edits on partially-initialized repos).
[ -n "$stories" ] || exit 0
# Story files themselves are always editable (managing the work).
case "$file_path" in
  "$stories"/*.md) exit 0 ;;
esac

# Require at least one started story in this repo's backlog.
started=0
if compgen -G "$stories/*.md" >/dev/null; then
  started=$( { grep -l '^status: started$' "$stories"/*.md 2>/dev/null || true; } | wc -l | tr -d ' ')
fi

if [ "${started:-0}" -eq 0 ]; then
  rel="${file_path#"$repo"/}"
  {
    echo "Coach refused: $tool_name on $rel"
    echo
    echo "Rule: code changes happen on a started story. No story is in progress"
    echo "      in $(basename "$repo")'s backlog."
    echo "Next: pull one first —"
    echo "  am next        # show the top of priority"
    echo "  am pull        # next + start the top"
    echo "  am start <story.md>"
    echo
    echo "(Backlog files, .am/, .claude/ are always editable so you can manage the work.)"
  } >&2
  exit 2
fi

exit 0
