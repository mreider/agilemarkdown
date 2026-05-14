#!/usr/bin/env bash
# Agile Markdown coach gate. Wired as a Claude Code PreToolUse hook on
# the mcp__agilemarkdown__set_status and mcp__agilemarkdown__set_estimate
# tools. Reads the hook's JSON envelope from stdin, runs `am coach-check`
# against the planned action, and exits non-zero with a message when the
# action breaks a hard rule.
#
# Exit codes:
#   0  allowed (Claude Code proceeds with the tool call)
#   2  blocked (Claude Code shows the message to the user and aborts)
#
# This hook is shipped with agilemarkdown and installed by
# `am create-backlog` at .claude/hooks/coach-gate.sh. Edits to the
# hard rules live in mcpserver/coach.go; this hook is just a transport.

set -euo pipefail

input="$(cat)"

tool_name="$(printf '%s' "${input}" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("tool_name",""))' 2>/dev/null || true)"
tool_input="$(printf '%s' "${input}" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(json.dumps(d.get("tool_input",{})))' 2>/dev/null || true)"

if [ -z "${tool_name}" ]; then
  exit 0
fi

am_bin="${AM_BIN:-am}"
if ! command -v "${am_bin}" >/dev/null 2>&1; then
  # No am on PATH. Don't block the workflow on a missing tool.
  exit 0
fi

run_check() {
  set +e
  out="$("${am_bin}" "$@" 2>&1)"
  rc=$?
  set -e
  if [ "${rc}" -ne 0 ]; then
    {
      echo "Coach refused: ${tool_name}"
      echo
      echo "${out}"
    } >&2
    exit 2
  fi
}

case "${tool_name}" in
  mcp__agilemarkdown__set_status)
    path="$(printf '%s' "${tool_input}" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("path",""))' 2>/dev/null || true)"
    status="$(printf '%s' "${tool_input}" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("status",""))' 2>/dev/null || true)"
    if [ -z "${path}" ] || [ -z "${status}" ]; then
      exit 0
    fi
    run_check coach-check set_status --path "${path}" --status "${status}"
    # On a started transition for a feature, also gate on the pull check.
    # A feature without acceptance bullets is the genie-builds-wrong-thing
    # failure mode; refuse here so the agent surfaces the rule.
    if [ "${status}" = "started" ]; then
      run_check coach-check pull --path "${path}"
    fi
    exit 0
    ;;
  mcp__agilemarkdown__set_estimate)
    path="$(printf '%s' "${tool_input}" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("path",""))' 2>/dev/null || true)"
    estimate="$(printf '%s' "${tool_input}" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("estimate",""))' 2>/dev/null || true)"
    if [ -z "${path}" ] || [ -z "${estimate}" ]; then
      exit 0
    fi
    run_check coach-check set_estimate --path "${path}" --estimate "${estimate}"
    exit 0
    ;;
esac

exit 0
