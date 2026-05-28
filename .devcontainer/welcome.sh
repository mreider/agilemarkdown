#!/usr/bin/env bash
# Codespaces postStart: print a tiny how-to so first-time visitors
# know what to do next. Runs every time the codespace starts.

cat <<'BANNER'

═══════════════════════════════════════════════════════════════════════
  agilemarkdown, try it out

  You're inside the demo project: 54 stories, 5 epics, 2 releases,
  six iterations of velocity history.

  Click the agilemarkdown icon in the activity bar (left edge) to
  see Search / Backlog / Done / Icebox / Epics.

  CLI:    am show priority     |    am show done     |    am dashboard
  Docs:   https://agilemarkdown.com
  Source: /workspaces/agilemarkdown  (File ▸ Open Folder…)
═══════════════════════════════════════════════════════════════════════

BANNER
