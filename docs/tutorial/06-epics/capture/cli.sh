#!/usr/bin/env bash
set -euo pipefail
CHAPTER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKSPACE="${AM_WORKSPACE:?AM_WORKSPACE not set}"
CLI_OUT="${CHAPTER_DIR}/cli"
mkdir -p "${CLI_OUT}"

(
  cd "${WORKSPACE}/stories"
  echo '$ am show epic the-button-quest'
  am show epic the-button-quest
) > "${CLI_OUT}/01-finished-epic.txt"

(
  cd "${WORKSPACE}/stories"
  echo '$ am show epic the-garden-of-patience'
  am show epic the-garden-of-patience
) > "${CLI_OUT}/02-in-flight-epic.txt"

(
  cd "${WORKSPACE}/stories"
  echo '$ am show epic the-moon-watch'
  am show epic the-moon-watch
) > "${CLI_OUT}/03-not-started-epic.txt"

echo "  wrote ${CLI_OUT}/*.txt"
