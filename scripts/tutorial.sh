#!/usr/bin/env bash
# Tutorial capture pipeline.
#
# Boots `am serve` against two workspaces (a scratch one for the
# first two chapters, a Lobel-fixture one for the rest), runs the
# Playwright spec to capture UI screenshots, runs the per-chapter
# capture-cli.sh to record CLI output, and runs the capture-mcp Go
# helper to write MCP wire traces. Outputs land in
# docs/tutorial/<chapter>/{images,cli,mcp}/.
#
# Usage:
#   scripts/tutorial.sh            run the whole tutorial
#   scripts/tutorial.sh 04         only chapter 04
#
# Determinism: each chapter starts from a known fixture state. The
# scratch workspace is wiped + reseeded by hand; the meadow workspace
# is wiped + reseeded by scripts/build-demo.sh at the start of
# Chapter 3. State then evolves naturally across the chapter so the
# story stays coherent across pages.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${REPO_ROOT}"

CHAPTER="${1:-all}"

LAB_SCRATCH="/tmp/am-tut-scratch"
LAB_MEADOW="/tmp/am-tut-meadow"
LAB_PORT_SCRATCH="8090"
LAB_PORT_MEADOW="8091"

# Build everything we need before doing anything.
echo "▶ building am + lab bundle"
go build -o "${REPO_ROOT}/bin/am" .
(cd vscode && npm run build:web >/dev/null)
go build -o "${REPO_ROOT}/bin/am-server" ./cmd/am-server/
export PATH="${REPO_ROOT}/bin:${PATH}"

# Helper: wait for a URL to respond 200.
wait_for() {
  local url="$1"
  for _ in $(seq 1 60); do
    if curl -s -f -o /dev/null "${url}"; then
      return 0
    fi
    sleep 1
  done
  echo "timed out waiting for ${url}" >&2
  return 1
}

# Helper: boot a lab pointed at a given workspace + port. Echoes
# the PID so the caller can shut it down.
boot_lab() {
  local root="$1" port="$2" reset="$3" fixture="$4"
  local flags=("--root" "${root}" "--port" "${port}")
  if [[ "${reset}" == "noreset" ]]; then
    flags+=("--no-reset")
  fi
  if [[ -n "${fixture}" ]]; then
    flags+=("--fixture" "${fixture}")
  fi
  am-server "${flags[@]}" >"/tmp/am-lab-${port}.log" 2>&1 &
  echo $!
}

# Helper: kill a lab cleanly.
kill_lab() {
  local pid="$1"
  kill "${pid}" 2>/dev/null || true
  wait "${pid}" 2>/dev/null || true
}

# Helper: run a chapter's CLI capture + Playwright spec + MCP capture.
# Caller must have a lab booted on the right port before invoking.
run_chapter() {
  local chapter="$1" port="$2" workspace="$3"
  echo "▶ chapter ${chapter} → port ${port} workspace ${workspace}"

  local chapter_dir="docs/tutorial/${chapter}"
  if [[ -x "${chapter_dir}/capture/cli.sh" ]]; then
    AM_LAB_URL="http://localhost:${port}" \
    AM_WORKSPACE="${workspace}" \
    bash "${chapter_dir}/capture/cli.sh"
  fi

  if [[ -f "${chapter_dir}/playwright/spec.ts" ]]; then
    AM_LAB_URL="http://localhost:${port}" \
    AM_CHAPTER_DIR="${REPO_ROOT}/${chapter_dir}" \
    npx playwright test --config="${REPO_ROOT}/docs/tutorial/playwright/playwright.config.ts" \
                        "${REPO_ROOT}/${chapter_dir}/playwright/spec.ts" \
                        --reporter=line || true
  fi

  if [[ -f "${chapter_dir}/capture/mcp.yaml" ]]; then
    AM_LAB_URL="http://localhost:${port}" \
    "${REPO_ROOT}/bin/capture-mcp" \
      -plan "${chapter_dir}/capture/mcp.yaml" \
      -out "${chapter_dir}/mcp"
  fi
}

# ---- Chapters 1-2 run against the scratch workspace ----

run_scratch() {
  rm -rf "${LAB_SCRATCH}"
  mkdir -p "${LAB_SCRATCH}"
  # No fixture, empty workspace. The lab's auto-reset would try to
  # re-seed; --no-reset preserves whatever state Chapter 1's CLI
  # script builds up.
  local pid; pid=$(boot_lab "${LAB_SCRATCH}" "${LAB_PORT_SCRATCH}" "noreset" "")
  trap "kill_lab ${pid}" EXIT
  wait_for "http://localhost:${LAB_PORT_SCRATCH}/"

  for ch in 01-inception 02-first-story; do
    if [[ "${CHAPTER}" == "all" || "${CHAPTER}" == "${ch%%-*}" ]]; then
      run_chapter "${ch}" "${LAB_PORT_SCRATCH}" "${LAB_SCRATCH}"
    fi
  done

  kill_lab "${pid}"
  trap - EXIT
}

# ---- Chapters 3-11 run against the meadow workspace ----

run_meadow() {
  rm -rf "${LAB_MEADOW}"
  local pid; pid=$(boot_lab "${LAB_MEADOW}" "${LAB_PORT_MEADOW}" "reset" "${REPO_ROOT}/scripts/build-demo.sh")
  trap "kill_lab ${pid}" EXIT
  wait_for "http://localhost:${LAB_PORT_MEADOW}/api/snapshot"

  for ch in 03-meadow-intro 04-current-sprint 05-when-things-go-wrong 06-epics \
            07-release-markers 08-done 09-search 10-comments-tasks 11-mcp-and-agents; do
    if [[ "${CHAPTER}" == "all" || "${CHAPTER}" == "${ch%%-*}" ]]; then
      run_chapter "${ch}" "${LAB_PORT_MEADOW}" "${LAB_MEADOW}"
    fi
  done

  kill_lab "${pid}"
  trap - EXIT
}

# Build the MCP capture binary once.
go build -o "${REPO_ROOT}/bin/capture-mcp" "${REPO_ROOT}/docs/tutorial/capture/capture-mcp.go"

run_scratch
run_meadow

echo
echo "▶ rendering chapters → docs/tutorial-*.html"
go run "${REPO_ROOT}/scripts/build-site"

echo
echo "✓ tutorial captures + static site up to date"
echo "  artifacts: docs/tutorial/"
echo "  pages:     docs/tutorial.html + docs/tutorial-NN-*.html"
