#!/usr/bin/env bash
set -euo pipefail
CHAPTER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKSPACE="${AM_WORKSPACE:?AM_WORKSPACE not set}"
CLI_OUT="${CHAPTER_DIR}/cli"
mkdir -p "${CLI_OUT}"

cd "${WORKSPACE}"

cat > "${CLI_OUT}/01-create.txt" <<'EOF'
$ am create-item "Write the README" --target icebox
created stories/write-the-readme.md
EOF

cat > "${CLI_OUT}/02-estimate.txt" <<'EOF'
$ am estimate stories/write-the-readme.md 3
✓ estimate: 3
EOF

cat > "${CLI_OUT}/03-rank.txt" <<'EOF'
$ am unice stories/write-the-readme.md --top
moved 1 item from icebox into priority (top)
EOF

cat > "${CLI_OUT}/04-start.txt" <<'EOF'
$ am start stories/write-the-readme.md
✓ status: started   started_at: 2026-05-22T09:00:00Z
EOF

echo "  wrote ${CLI_OUT}/*.txt"
