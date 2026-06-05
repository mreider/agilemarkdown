#!/usr/bin/env bash
# Agile Markdown SessionStart hook. Installed by `am init` /
# `am create-backlog` and wired into .claude/settings.json. Prints the
# project brief (vision + dashboard + next pull + WIP + working
# agreements) into the agent's context at the start of every session, so
# a fresh agent — or a fresh clone on a new machine — begins with the same
# shared picture the PM and the dev pair have. Output goes to stdout,
# which Claude Code injects as a system reminder.
#
# This file is shipped with agilemarkdown. It carries no hardcoded paths,
# so it works in any clone at any location. Edit freely; `am init` never
# overwrites an existing copy.

set -uo pipefail

# Resolve the repo root from this script's own location (works regardless
# of the cwd Claude Code invokes the hook from), falling back to the git
# toplevel.
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." 2>/dev/null && pwd)"
ROOT="${ROOT:-$(git rev-parse --show-toplevel 2>/dev/null)}"
[ -n "${ROOT}" ] && cd "${ROOT}" 2>/dev/null || true

am_bin="${AM_BIN:-am}"
if ! command -v "${am_bin}" >/dev/null 2>&1; then
  echo "agilemarkdown (am) is not on PATH. Install it for backlog context:"
  echo "  go install github.com/mreider/agilemarkdown@latest"
  exit 0
fi

# `am brief` is the one-shot onboarding blob. Older am builds don't have
# it; fall back to dashboard + next so the hook still surfaces context.
brief_out="$("${am_bin}" brief 2>/dev/null || true)"
if [ -n "${brief_out}" ]; then
  printf '%s\n' "${brief_out}"
else
  echo "## Backlog state"
  echo '```'
  "${am_bin}" dashboard 2>/dev/null || true
  echo
  "${am_bin}" next 2>/dev/null || true
  echo '```'
fi

exit 0
