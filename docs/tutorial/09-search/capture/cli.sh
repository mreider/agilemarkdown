#!/usr/bin/env bash
set -euo pipefail
CHAPTER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKSPACE="${AM_WORKSPACE:?AM_WORKSPACE not set}"
CLI_OUT="${CHAPTER_DIR}/cli"
mkdir -p "${CLI_OUT}"

(
  cd "${WORKSPACE}"
  echo '$ am search "seedling"'
  am search "seedling" --limit 5
) > "${CLI_OUT}/01-search-text.txt"

(
  cd "${WORKSPACE}"
  echo '$ am list-items stories --tag garden | jq ".items[] | {title, status}" | head -20'
  am list-items stories --tag garden | head -20
) > "${CLI_OUT}/02-list-by-tag.txt"

echo "  wrote ${CLI_OUT}/*.txt"
