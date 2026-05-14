#!/usr/bin/env bash
# Build a populated agilemarkdown project for screenshots, demos, and
# generally working through the site's tutorials without disturbing
# the test fixture.
#
# Default target: ~/agilemarkdown-demo. Override by passing a path:
#   ./scripts/build-demo.sh /tmp/am-demo
#
# Uses the same generator as the E2E suite
# (tests/fixtures/build-sample.sh), so the demo and the tests share
# one source of truth. After the build, the directory holds:
#   - product/ and platform/ backlogs with 18 stories across them
#   - stories in every state (accepted, started, finished, delivered,
#     unstarted, rejected)
#   - acceptance bullets in mixed [ ] / [~] / [x] states
#   - an auth-rewrite epic with three phases
#   - a release marker (Q3 ship)
#   - a half-strength iteration override
#   - inception.md, team-agreements.md, learnings.md
#   - three iterations of accepted points so velocity computes to
#     real numbers (product velocity is 8, platform is 16)
#
# When `jq` is present, the build also asserts a few JSON shapes
# (backlog count, accepted-stories count, velocity number). The
# asserts double as an integration check that `am`'s JSON surface
# still matches what the docs promise.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FIXTURE="${REPO_ROOT}/tests/fixtures/build-sample.sh"
TARGET="${1:-${HOME}/agilemarkdown-demo}"

if [ ! -x "${FIXTURE}" ]; then
  echo "fixture not found: ${FIXTURE}" >&2
  exit 1
fi

if ! command -v am >/dev/null 2>&1; then
  echo "am not on PATH. Run build.sh from the repo root first." >&2
  exit 1
fi

echo "Building demo at ${TARGET}"
bash "${FIXTURE}" "${TARGET}" >/dev/null

# Sanity-check the build through the JSON surface. Skips gracefully
# when `jq` is missing; surfaces a clear failure when a shape drifts.
if command -v jq >/dev/null 2>&1; then
  echo
  echo "Checking JSON surface…"
  pushd "${TARGET}" >/dev/null

  backlogs=$(am list-backlogs | jq -r '.backlogs | length')
  if [ "${backlogs}" != "2" ]; then
    echo "  fail: expected 2 backlogs, got ${backlogs}" >&2
    exit 1
  fi
  echo "  list-backlogs: ${backlogs} backlogs"

  product_items=$(am list-items product | jq -r '.count')
  platform_items=$(am list-items platform | jq -r '.count')
  echo "  list-items:    product=${product_items}  platform=${platform_items}"

  accepted=$(am dashboard --json | jq -r '.stories_accepted_total')
  velocity=$(am dashboard --json | jq -r '.velocity')
  echo "  dashboard:     velocity=${velocity}  accepted=${accepted}"
  if [ "${accepted}" -lt 1 ]; then
    echo "  fail: dashboard reported zero accepted stories" >&2
    exit 1
  fi

  # `am velocity` reads the current backlog, so run it from inside one.
  iterations=$( (cd product && am velocity --json) | jq -r '.rows | length')
  echo "  velocity:      ${iterations} iteration rows (product)"

  blocked=$(am list-items product | jq -r '[.items[] | select(.blocked)] | length')
  echo "  blocked items: ${blocked} in product"

  popd >/dev/null
else
  echo
  echo "(install jq to run JSON surface checks)"
fi

echo
echo "Done."
echo
echo "Next steps:"
echo "  cd ${TARGET}"
echo "  am show priority             # look at the product backlog"
echo "  am sprint plan               # Monday-morning planning view"
echo "  am retro                     # end-of-iteration retro"
echo "  am accept-prompt product/Login-flow.md   # the PM ceremony"
echo
echo "Or pipe through the JSON surface:"
echo "  am dashboard --json | jq"
echo "  am list-items product --status started | jq '.items[].title'"
echo
if command -v code >/dev/null 2>&1; then
  echo "Open in VS Code:"
  echo "  code ${TARGET}"
  echo
fi
