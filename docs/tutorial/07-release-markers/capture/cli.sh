#!/usr/bin/env bash
set -euo pipefail
CHAPTER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKSPACE="${AM_WORKSPACE:?AM_WORKSPACE not set}"
CLI_OUT="${CHAPTER_DIR}/cli"
mkdir -p "${CLI_OUT}"

(
  cd "${WORKSPACE}/stories"
  echo '$ am show priority --iterations 5'
  am show priority --iterations 5 | sed -n '1,40p'
) > "${CLI_OUT}/01-show-with-markers.txt"

cat > "${CLI_OUT}/02-create-release.txt" <<'EOF'
$ am create-item "Spring planting"
created stories/spring-planting.md
$ am set-type stories/spring-planting.md release
spring-planting.md type=release
$ am set-release-date stories/spring-planting.md 2026-09-01
spring-planting.md release_date=2026-09-01
EOF

cat > "${CLI_OUT}/03-ship-release.txt" <<'EOF'
$ am accept stories/first-frost-release.md
✓ status: accepted   accepted_at: 2026-06-17T12:00:00Z
EOF

echo "  wrote ${CLI_OUT}/*.txt"
