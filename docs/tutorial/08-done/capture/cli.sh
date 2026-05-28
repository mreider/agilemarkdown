#!/usr/bin/env bash
set -euo pipefail
CHAPTER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKSPACE="${AM_WORKSPACE:?AM_WORKSPACE not set}"
CLI_OUT="${CHAPTER_DIR}/cli"
mkdir -p "${CLI_OUT}"

(
  cd "${WORKSPACE}/stories"
  echo '$ am show done'
  am show done | sed -n '1,20p'
) > "${CLI_OUT}/01-show-done.txt"

(
  cd "${WORKSPACE}/stories"
  echo '$ am retro'
  am retro
  echo
  echo '$ am record-learning "Pointing the gnarly one first kept the rest honest."'
  am record-learning "Pointing the gnarly one first kept the rest honest."
) > "${CLI_OUT}/02-retro.txt"

echo "  wrote ${CLI_OUT}/*.txt"
