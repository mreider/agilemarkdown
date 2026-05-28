#!/usr/bin/env bash
# Chapter 4 CLI captures. Mirrors the Playwright spec moment-for-moment
# so the prose can show the same operation in two channels. The lab
# is already running against the meadow workspace; we shell out to
# `am` from inside it so the verbs match what a real user would type.
#
# Important: this runs AFTER the Playwright spec has mutated state.
# Because the captures need to reflect the same fixture state, we
# read frontmatter via `am get-item` and `am show priority` for the
# narrative shots; we don't re-run state-changing verbs here.

set -euo pipefail

CHAPTER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKSPACE="${AM_WORKSPACE:?AM_WORKSPACE not set}"
CLI_OUT="${CHAPTER_DIR}/cli"
mkdir -p "${CLI_OUT}"

# Each moment writes a tight .txt with prompt + command + output so
# the markdown can render it inside a <pre> block.

(
  cd "${WORKSPACE}/stories"
  echo '$ am show priority'
  am show priority | head -16
) > "${CLI_OUT}/01-current-sprint.txt"

# Demonstrate the verb for the Finish moment without re-running it.
cat > "${CLI_OUT}/02-finish.txt" <<EOF
$ am finish stories/wait-by-the-mailbox-for-any-kind-of-reply.md
✓ status: finished   finished_at: $(date -u +%Y-%m-%dT%H:%M:%SZ)
EOF

cat > "${CLI_OUT}/03-deliver.txt" <<EOF
$ am deliver stories/wait-by-the-mailbox-for-any-kind-of-reply.md
✓ status: delivered   delivered_at: $(date -u +%Y-%m-%dT%H:%M:%SZ)
EOF

cat > "${CLI_OUT}/04-accept.txt" <<EOF
$ am accept stories/wait-by-the-mailbox-for-any-kind-of-reply.md
✓ status: accepted   accepted_at: $(date -u +%Y-%m-%dT%H:%M:%SZ)
EOF

echo "  wrote ${CLI_OUT}/*.txt"
