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
#   - a single stories/ backlog with 54 stories in a Lobel-flavored
#     meadow (Frog, Toad, Mouse, Owl, Snail, Sparrow, Bear, Lizard)
#   - stories in every state (accepted, started, finished, delivered,
#     unstarted, rejected)
#   - five epics (two finished, two in flight, one not yet started)
#   - two release markers (First-frost and The Long-Sleep)
#   - a half-strength iteration override
#   - .am/inception.md, .am/team-agreements.md, .am/learnings.md
#   - six iterations of accepted points so velocity computes to a
#     real, stable number
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
  if [ "${backlogs}" != "1" ]; then
    echo "  fail: expected 1 backlog, got ${backlogs}" >&2
    exit 1
  fi
  echo "  list-backlogs: ${backlogs} backlog"

  items=$(am list-items stories | jq -r '.count')
  echo "  list-items:    stories=${items}"
  if [ "${items}" -ne 54 ]; then
    echo "  fail: expected 54 stories, got ${items}" >&2
    exit 1
  fi

  accepted=$(am dashboard --json | jq -r '.stories_accepted_total')
  velocity=$(am dashboard --json | jq -r '.velocity')
  echo "  dashboard:     velocity=${velocity}  accepted=${accepted}"
  # 18 accepted in prior iterations + 3 accepted in the current
  # sprint that haven't rolled over yet = 21.
  if [ "${accepted}" -ne 21 ]; then
    echo "  fail: expected 21 accepted stories, got ${accepted}" >&2
    exit 1
  fi

  iterations=$( (cd stories && am velocity --json) | jq -r '.rows | length')
  echo "  velocity:      ${iterations} iteration rows"

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
echo "  am show priority             # look at the current iteration"
echo "  am sprint plan               # Monday-morning planning view"
echo "  am retro                     # end-of-iteration retro"
echo
echo "Or pipe through the JSON surface:"
echo "  am dashboard --json | jq"
echo "  am list-items stories --status started | jq '.items[].title'"
echo
if command -v code >/dev/null 2>&1; then
  echo "Open in VS Code:"
  echo "  code ${TARGET}"
  echo
fi
