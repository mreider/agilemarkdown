#!/usr/bin/env bash
# Verify that the four coach-mode templates carry identical body
# content. Format-specific headers (Cursor's frontmatter, etc.) are
# stripped before diffing.
#
# Used by CI and the e2e suite. Catches drift when one template gets
# edited without the other three.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${REPO_ROOT}"

TEMPLATES=(
  "coach/agilemarkdown-coach.md"
  "coach/AGENTS.md"
  "coach/copilot-instructions.md"
  "coach/cursor-coach.mdc"
)

for f in "${TEMPLATES[@]}"; do
  if [ ! -f "${f}" ]; then
    echo "missing template: ${f}" >&2
    exit 1
  fi
done

# Strip per-format headers so the bodies can be compared directly.
#   - Cursor `.mdc` files start with a YAML frontmatter block fenced by
#     `---` lines; drop that block.
#   - Other files have no format-specific framing.
strip_frontmatter() {
  awk '
    BEGIN { skip=0; started=0 }
    NR==1 && $0=="---" { skip=1; next }
    skip==1 && $0=="---" { skip=0; next }
    skip==1 { next }
    !started && $0=="" { next }
    { started=1; print }
  ' "$1"
}

tmp="$(mktemp -d)"
trap 'rm -rf "${tmp}"' EXIT

for f in "${TEMPLATES[@]}"; do
  out="${tmp}/$(basename "${f%.*}").body"
  strip_frontmatter "${f}" > "${out}"
done

base="${tmp}/agilemarkdown-coach.body"
fail=0
for f in "${TEMPLATES[@]}"; do
  out="${tmp}/$(basename "${f%.*}").body"
  if ! diff -q "${base}" "${out}" > /dev/null; then
    echo "drift: ${f} differs from coach/agilemarkdown-coach.md (after stripping frontmatter)" >&2
    diff -u "${base}" "${out}" | head -40 >&2
    fail=1
  fi
done

if [ "${fail}" -ne 0 ]; then
  echo
  echo "Re-sync the coach templates. Edit coach/agilemarkdown-coach.md (the canonical body), then mirror to AGENTS.md, copilot-instructions.md, and cursor-coach.mdc." >&2
  exit 1
fi

# Repo-local Claude config under .claude/ must match the canonical
# coach/ templates byte-for-byte. The drift surfaces when an edit
# lands in coach/ but the repo's own Claude session keeps reading
# the stale projection.
LOCAL_PAIRS=(
  ".claude/agilemarkdown-coach.md::coach/agilemarkdown-coach.md"
  ".claude/hooks/coach-gate.sh::coach/hooks/coach-gate.sh"
)

for pair in "${LOCAL_PAIRS[@]}"; do
  local_path="${pair%%::*}"
  canon_path="${pair##*::}"
  if [ ! -f "${local_path}" ] || [ ! -f "${canon_path}" ]; then
    continue
  fi
  if ! diff -q "${canon_path}" "${local_path}" > /dev/null; then
    echo "drift: ${local_path} differs from ${canon_path}" >&2
    diff -u "${canon_path}" "${local_path}" | head -40 >&2
    echo >&2
    echo "Re-sync the repo-local .claude/ files. Run: cp ${canon_path} ${local_path}" >&2
    exit 1
  fi
done

echo "ok: 4 coach body projections in sync; .claude/ matches coach/"
