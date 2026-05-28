#!/usr/bin/env bash
set -euo pipefail
CHAPTER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKSPACE="${AM_WORKSPACE:?AM_WORKSPACE not set}"
CLI_OUT="${CHAPTER_DIR}/cli"
mkdir -p "${CLI_OUT}"

cat > "${CLI_OUT}/01-add-comment.txt" <<'EOF'
$ am comment stories/plan-the-moons-very-first-birthday-party.md \
    "Maybe Owl should bake the cake?" --author frog
✓ comment added
EOF

cat > "${CLI_OUT}/02-add-task.txt" <<'EOF'
$ am task add stories/plan-the-moons-very-first-birthday-party.md \
    "Find a stand for the cake"
✓ task added
EOF

cat > "${CLI_OUT}/03-tick-task.txt" <<'EOF'
$ am task tick stories/plan-the-moons-very-first-birthday-party.md 2
✓ task toggled
EOF

echo "  wrote ${CLI_OUT}/*.txt"
