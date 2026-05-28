#!/usr/bin/env bash
set -euo pipefail
CHAPTER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLI_OUT="${CHAPTER_DIR}/cli"
mkdir -p "${CLI_OUT}"

cat > "${CLI_OUT}/01-reject.txt" <<EOF
$ am reject stories/sort-the-unread-letters-into-three-quiet-piles.md \\
    --reason "PM noticed the third pile is mislabeled."
✓ rejected, back to started
EOF

cat > "${CLI_OUT}/02-unstart.txt" <<EOF
$ am unstart stories/yell-at-the-seedling-to-grow-faster.md
✓ status: unstarted   started/finished/delivered/accepted cleared
EOF

cat > "${CLI_OUT}/03-block.txt" <<EOF
$ am block stories/pace-exactly-seven-times-up-and-down-the-path.md \\
    --reason "Path is wet from the rain. Waiting for it to dry."
✓ blocked
EOF

echo "  wrote ${CLI_OUT}/*.txt"
