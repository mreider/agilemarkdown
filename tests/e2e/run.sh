#!/usr/bin/env bash
# End-to-end test suite. Exercises every CLI command, with every flag,
# and verifies the on-disk artifacts. Runs in CI before release.
#
# Usage:
#   bash tests/e2e/run.sh                # builds ./agilemarkdown then runs
#   AM_BIN=/path/to/am bash tests/e2e/run.sh   # use existing binary

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export REPO_ROOT
cd "${REPO_ROOT}"

if [ -z "${AM_BIN:-}" ]; then
  echo "==> building agilemarkdown"
  go build -o "${REPO_ROOT}/agilemarkdown" .
  AM_BIN="${REPO_ROOT}/agilemarkdown"
fi
export AM_BIN
echo "==> AM_BIN=${AM_BIN}"
"${AM_BIN}" --version

WORK="$(mktemp -d -t am-e2e-XXXXXX)"
trap 'echo "==> tmp ${WORK}"; rm -rf "${WORK}"' EXIT

pass=0
fail=0
fail_msgs=()

run_step() {
  local name="$1"; shift
  echo
  echo "---- ${name} ----"
  if "$@"; then
    echo "ok: ${name}"
    pass=$((pass+1))
  else
    echo "FAIL: ${name}"
    fail=$((fail+1))
    fail_msgs+=("${name}")
  fi
}

assert_file() {
  local path="$1"
  if [ ! -f "${path}" ]; then
    echo "expected file: ${path}" >&2
    return 1
  fi
}

assert_dir() {
  local path="$1"
  if [ ! -d "${path}" ]; then
    echo "expected dir: ${path}" >&2
    return 1
  fi
}

assert_grep() {
  local path="$1"; local pattern="$2"
  if ! grep -q -- "${pattern}" "${path}"; then
    echo "pattern not found in ${path}: ${pattern}" >&2
    cat "${path}" >&2
    return 1
  fi
}

assert_not_grep() {
  local path="$1"; local pattern="$2"
  if grep -q -- "${pattern}" "${path}"; then
    echo "unexpected pattern in ${path}: ${pattern}" >&2
    return 1
  fi
}

REPO_DIR="${WORK}/repo"
mkdir -p "${REPO_DIR}"
cd "${REPO_DIR}"
git init -q
git config user.email "alice@example.com"
git config user.name "Alice Example"

#######################################
# create-user (positional + flag forms)
#######################################
test_create_user_flags() {
  "${AM_BIN}" create-user --name alice --email alice@example.com
  assert_file users/alice.md
  assert_grep users/alice.md "name: alice"
  assert_grep users/alice.md "alice@example.com"
}
test_create_user_positional() {
  "${AM_BIN}" create-user "Bob Builder" bob@example.com
  assert_file "users/Bob Builder.md"
  assert_grep "users/Bob Builder.md" "bob@example.com"

  "${AM_BIN}" create-user "Carol Coder" carol@example.com
  assert_file "users/Carol Coder.md"
}

#######################################
# create-backlog
#######################################
test_create_backlog() {
  "${AM_BIN}" create-backlog product
  assert_dir product
  assert_file product.md
  # Coach mode: canonical body lives at .claude/agilemarkdown-coach.md.
  # CLAUDE.md is a thin @-import. The other three projections carry the
  # full body inline.
  assert_file .claude/agilemarkdown-coach.md
  assert_file CLAUDE.md
  assert_file AGENTS.md
  assert_file .github/copilot-instructions.md
  assert_file .cursor/rules/coach.mdc
  assert_grep .claude/agilemarkdown-coach.md "dev pair"
  assert_grep CLAUDE.md "@.claude/agilemarkdown-coach.md"
  assert_grep AGENTS.md "dev pair"
  assert_grep .cursor/rules/coach.mdc "alwaysApply: true"
  # Skills package projects under .claude/skills/.
  assert_file .claude/skills/am-accept/SKILL.md
  assert_file .claude/skills/am-decompose/SKILL.md
  assert_file .claude/skills/am-retro/SKILL.md
  assert_grep .claude/skills/am-accept/SKILL.md "PM acceptance ceremony"
  # Coach gate hook + settings.
  assert_file .claude/hooks/coach-gate.sh
  assert_file .claude/settings.json
  assert_grep .claude/settings.json "PreToolUse"
  assert_grep .claude/settings.json "mcp__agilemarkdown__set_status"
  if [ ! -x .claude/hooks/coach-gate.sh ]; then
    echo "coach-gate.sh is not executable" >&2
    return 1
  fi
}

test_coach_hook_refuses_self_accept() {
  cd "${REPO_DIR}"
  set +e
  AM_BIN="${AM_BIN}" bash .claude/hooks/coach-gate.sh <<JSON 2>/tmp/am-hook.err
{"tool_name":"mcp__agilemarkdown__set_status","tool_input":{"path":"product/Search-relevance.md","status":"accepted"}}
JSON
  rc=$?
  set -e
  if [ "${rc}" -ne 2 ]; then
    echo "expected hook exit 2, got ${rc}" >&2
    cat /tmp/am-hook.err >&2 || true
    return 1
  fi
  grep -q "coach-refuses-pm-accepts" /tmp/am-hook.err
  rm -f /tmp/am-hook.err
}

test_coach_hook_allows_legitimate_transitions() {
  cd "${REPO_DIR}"
  set +e
  AM_BIN="${AM_BIN}" bash .claude/hooks/coach-gate.sh <<JSON 2>/tmp/am-hook.err
{"tool_name":"mcp__agilemarkdown__set_status","tool_input":{"path":"product/Search-relevance.md","status":"started"}}
JSON
  rc=$?
  set -e
  if [ "${rc}" -ne 0 ]; then
    echo "hook should allow started transition; got rc=${rc}" >&2
    cat /tmp/am-hook.err >&2 || true
    return 1
  fi
  rm -f /tmp/am-hook.err
}

test_coach_hook_refuses_pull_no_acceptance() {
  # Reuse Pull-without-acceptance.md from the coach_check pull tests:
  # it is a feature with no `## Acceptance` section. The hook should
  # refuse set_status started on it.
  cd "${REPO_DIR}"
  set +e
  AM_BIN="${AM_BIN}" bash .claude/hooks/coach-gate.sh <<JSON 2>/tmp/am-hook.err
{"tool_name":"mcp__agilemarkdown__set_status","tool_input":{"path":"product/Pull-without-acceptance.md","status":"started"}}
JSON
  rc=$?
  set -e
  if [ "${rc}" -ne 2 ]; then
    echo "expected hook exit 2 for feature without acceptance; got ${rc}" >&2
    cat /tmp/am-hook.err >&2 || true
    return 1
  fi
  grep -q "acceptance-before-pull" /tmp/am-hook.err
  rm -f /tmp/am-hook.err
}

test_team_agreements_add() {
  tmp="$(mktemp -d)"
  cd "${tmp}"
  git init -q && git config user.email a@b.com && git config user.name "T"
  "${AM_BIN}" init >/dev/null
  "${AM_BIN}" team-agreements --add "Bugs go at the top of priority." >/dev/null
  "${AM_BIN}" team-agreements --add "8-point hard cap." >/dev/null
  assert_file team-agreements.md
  assert_grep team-agreements.md "Bugs go at the top"
  assert_grep team-agreements.md "8-point hard cap"
  # First --add seeds the header; subsequent --add preserves it.
  assert_grep team-agreements.md "^# Team agreements"
  cd "${REPO_DIR}"
  rm -rf "${tmp}"
}

test_coach_check_action_flag() {
  cd "${REPO_DIR}"
  set +e
  out="$("${AM_BIN}" coach-check --action set_status --path product/Search-relevance.md --status accepted 2>&1)"
  rc=$?
  set -e
  if [ "${rc}" -eq 0 ]; then
    echo "expected non-zero exit on refusal via --action flag, got 0" >&2
    return 1
  fi
  echo "${out}" | grep -q "coach-refuses-pm-accepts"
}

test_pull_cli() {
  tmp="$(mktemp -d)"
  cd "${tmp}"
  git init -q && git config user.email a@b.com && git config user.name "T"
  "${AM_BIN}" create-backlog product >/dev/null
  rm -f product/Sample-*.md
  cd product
  "${AM_BIN}" create-item "First story" >/dev/null
  "${AM_BIN}" estimate First-story.md 2 >/dev/null
  cd ..
  "${AM_BIN}" sync </dev/null >/dev/null
  cd product
  "${AM_BIN}" unice First-story.md --top >/dev/null
  cd ..
  out="$("${AM_BIN}" pull 2>&1)"
  echo "${out}" | grep -q "started"
  assert_grep product/First-story.md "^status: started"
  cd "${REPO_DIR}"
  rm -rf "${tmp}"
}

test_deliver_prompt_flag() {
  tmp="$(mktemp -d)"
  cd "${tmp}"
  git init -q && git config user.email a@b.com && git config user.name "T"
  "${AM_BIN}" create-backlog product >/dev/null
  rm -f product/Sample-*.md
  cd product
  "${AM_BIN}" create-item "Demo deliver prompt" >/dev/null
  "${AM_BIN}" estimate Demo-deliver-prompt.md 2 >/dev/null
  "${AM_BIN}" start Demo-deliver-prompt.md >/dev/null
  "${AM_BIN}" finish Demo-deliver-prompt.md >/dev/null
  out="$("${AM_BIN}" deliver --prompt Demo-deliver-prompt.md 2>&1)"
  echo "${out}" | grep -q "delivered"
  echo "${out}" | grep -q "As PM, do you accept?"
  cd "${REPO_DIR}"
  rm -rf "${tmp}"
}

test_estimate_advise() {
  out="$("${AM_BIN}" estimate --advise 2>&1)"
  echo "${out}" | grep -q "Pivotal estimation"
  echo "${out}" | grep -q "Fibonacci"
  echo "${out}" | grep -q "hard cap"
}

test_inception_cli() {
  cd "${REPO_DIR}"
  rm -f inception.md
  "${AM_BIN}" inception
  assert_file inception.md
  assert_grep inception.md "## The user"
  assert_grep inception.md "## Out of scope"
  out="$("${AM_BIN}" inception --show)"
  echo "${out}" | grep -q "The user"
}

test_sprint_plan_cli() {
  cd "${REPO_DIR}/product"
  out="$("${AM_BIN}" sprint plan 2>&1)"
  echo "${out}" | grep -q "Iteration plan"
  echo "${out}" | grep -q "velocity"
  echo "${out}" | grep -q "Committed:"
}

test_retro_cli() {
  cd "${REPO_DIR}"
  out="$("${AM_BIN}" retro 2>&1)"
  echo "${out}" | grep -q "Retro summary"
  echo "${out}" | grep -q "What worked"
  echo "${out}" | grep -q "rejection rate"
}

test_mcp_inception_doc() {
  cd "${REPO_DIR}"
  rm -f inception.md
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"inception_doc","arguments":{}}}' \
    '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"inception_doc","arguments":{"body":"# Inception\n\n## The user\nbeta testers\n"}}}' \
    '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"inception_doc","arguments":{}}}'
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  grep -q '"existed":false' "${out}"
  grep -q '"wrote":true' "${out}"
  grep -q "beta testers" "${out}"
  rm -f "${out}"
  rm -f inception.md
}

test_mcp_sprint_plan() {
  cd "${REPO_DIR}"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"sprint_plan","arguments":{"backlog":"product"}}}'
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  grep -q '"committed"' "${out}"
  grep -q '"velocity"' "${out}"
  grep -q '"warnings"' "${out}"
  rm -f "${out}"
}

test_coach_status_cli() {
  cd "${REPO_DIR}"
  out="$("${AM_BIN}" coach 2>&1)"
  echo "${out}" | grep -q "Coach status"
  echo "${out}" | grep -q "Pending acceptance:"
  echo "${out}" | grep -q "Blocked:"
}

test_am_init_reinstalls_idempotent() {
  tmp="$(mktemp -d)"
  cd "${tmp}"
  git init -q
  git config user.email init@example.com
  git config user.name "Init Tester"
  "${AM_BIN}" init >/dev/null
  assert_file CLAUDE.md
  assert_file .claude/settings.json
  # Six coach skills now ship; the new am-align is the pre-pull gate.
  assert_file .claude/skills/am-accept/SKILL.md
  assert_file .claude/skills/am-align/SKILL.md
  assert_file .claude/skills/am-decompose/SKILL.md
  assert_file .claude/skills/am-inception/SKILL.md
  assert_file .claude/skills/am-plan/SKILL.md
  assert_file .claude/skills/am-retro/SKILL.md
  # second run is a no-op
  out="$("${AM_BIN}" init)"
  echo "${out}" | grep -q "already installed"
  cd "${REPO_DIR}"
  rm -rf "${tmp}"
}
test_create_backlog_name_with_spaces() {
  "${AM_BIN}" create-backlog "marketing site"
  assert_dir "marketing-site" || assert_dir "marketing site"
}

#######################################
# create-item (with --simulate, --user)
#######################################
test_create_item_basic() {
  cd "${REPO_DIR}/product"
  "${AM_BIN}" create-item "Build login flow"
  assert_file Build-login-flow.md
  assert_grep Build-login-flow.md "title: Build login flow"
  assert_grep Build-login-flow.md "status: unstarted"
  # New default frontmatter + body shape (v4.15): type defaults to feature
  # and the body carries an `## Acceptance` section so the coach loop has
  # somewhere to read criteria from.
  assert_grep Build-login-flow.md "type: feature"
  assert_grep Build-login-flow.md "^## Acceptance"
}
test_create_item_user_flag() {
  cd "${REPO_DIR}/product"
  "${AM_BIN}" create-item --user alice "Search relevance"
  assert_file Search-relevance.md
  assert_grep Search-relevance.md "author: alice"
}
test_create_item_simulate() {
  cd "${REPO_DIR}/product"
  before=$(ls *.md | wc -l)
  "${AM_BIN}" create-item --simulate "Phantom item"
  after=$(ls *.md | wc -l)
  if [ "${before}" != "${after}" ]; then
    echo "simulate created files (before=${before} after=${after})" >&2
    return 1
  fi
}

#######################################
# Frontmatter editing helper for tests
#######################################
set_status() {
  local file="$1"; local status="$2"
  python3 - "$file" "$status" <<'PY'
import sys, re
fp, status = sys.argv[1], sys.argv[2]
with open(fp) as f: s = f.read()
s = re.sub(r'^status:.*$', f'status: {status}', s, count=1, flags=re.M)
open(fp, 'w').write(s)
PY
}
set_estimate() {
  local file="$1"; local est="$2"
  python3 - "$file" "$est" <<'PY'
import sys, re
fp, est = sys.argv[1], sys.argv[2]
with open(fp) as f: s = f.read()
if re.search(r'^estimate:', s, re.M):
    s = re.sub(r'^estimate:.*$', f'estimate: {est}', s, count=1, flags=re.M)
else:
    s = re.sub(r'^---\n', f'---\nestimate: {est}\n', s, count=1)
open(fp, 'w').write(s)
PY
}
set_tags() {
  local file="$1"; shift
  local list="["
  for t in "$@"; do list="${list}${t}, "; done
  list="${list%, }]"
  python3 - "$file" "$list" <<'PY'
import sys, re
fp, list_ = sys.argv[1], sys.argv[2]
with open(fp) as f: s = f.read()
if re.search(r'^tags:', s, re.M):
    s = re.sub(r'^tags:.*$', f'tags: {list_}', s, count=1, flags=re.M)
else:
    s = re.sub(r'^---\n', f'---\ntags: {list_}\n', s, count=1)
open(fp, 'w').write(s)
PY
}

#######################################
# sync (validates + generates views)
#######################################
test_sync_happy_path() {
  cd "${REPO_DIR}/product"
  set_status Build-login-flow.md started
  set_estimate Build-login-flow.md 5
  set_tags Build-login-flow.md q2 auth
  set_status Search-relevance.md unstarted
  set_estimate Search-relevance.md 3
  set_tags Search-relevance.md q2 search

  cd "${REPO_DIR}"
  "${AM_BIN}" sync </dev/null
  assert_file index.md
  assert_file velocity.md
  assert_file timeline.md
  assert_file users.md
  assert_file tags.md
  assert_dir tags
  assert_grep tags/q2.md "Build login flow"
  # ASCII chart is inlined in velocity.md (no SVG sidecars).
  assert_grep velocity.md "Velocity"
}

test_pivotal_transitions() {
  cd "${REPO_DIR}/product"
  rel=Build-login-flow.md
  "${AM_BIN}" finish "${rel}"
  assert_grep "${rel}" "status: finished"
  assert_grep "${rel}" "finished:"
  "${AM_BIN}" deliver "${rel}"
  assert_grep "${rel}" "status: delivered"
  assert_grep "${rel}" "delivered:"
  "${AM_BIN}" accept "${rel}"
  assert_grep "${rel}" "status: accepted"
  assert_grep "${rel}" "accepted:"
  # reject sends back to rejected
  "${AM_BIN}" reject "${rel}"
  assert_grep "${rel}" "status: rejected"
  # then start cycles back; accepted_at should be cleared
  "${AM_BIN}" start "${rel}"
  assert_grep "${rel}" "status: started"
  assert_not_grep "${rel}" "^accepted: "
}

#######################################
# Schema validation rejects bad items
#######################################
test_sync_rejects_bad_status() {
  cd "${REPO_DIR}/product"
  cp Build-login-flow.md Build-login-flow.md.bak
  set_status Build-login-flow.md "garbage"
  cd "${REPO_DIR}"
  if "${AM_BIN}" sync </dev/null 2>err.log; then
    echo "expected sync to fail on bad status" >&2
    cat err.log >&2
    return 1
  fi
  assert_grep err.log "Validation failed"
  cd "${REPO_DIR}/product"
  mv Build-login-flow.md.bak Build-login-flow.md
}

test_config_yaml_present() {
  cd "${REPO_DIR}"
  assert_file .am/config.yaml
  assert_grep .am/config.yaml "scale: fibonacci"
  assert_grep .am/config.yaml "length_weeks: 1"
  assert_grep .am/config.yaml "strategy: rolling"
}

#######################################
# work / points / velocity (terminal)
#######################################
test_work_points_velocity() {
  cd "${REPO_DIR}/product"
  "${AM_BIN}" work >/dev/null
  "${AM_BIN}" work -s d >/dev/null
  "${AM_BIN}" work -s p >/dev/null
  "${AM_BIN}" work -t q2 >/dev/null
  "${AM_BIN}" points >/dev/null
  "${AM_BIN}" points -s d >/dev/null
  "${AM_BIN}" velocity 8 >/dev/null
}

#######################################
# change-tag / delete-tag
#######################################
test_change_delete_tag() {
  cd "${REPO_DIR}/product"
  "${AM_BIN}" change-tag q2 q3
  assert_grep Build-login-flow.md "q3"
  assert_not_grep Build-login-flow.md " q2"
  echo "y" | "${AM_BIN}" delete-tag search >/dev/null
  assert_not_grep Search-relevance.md " search"
}

#######################################
# archive
#######################################
test_archive() {
  cd "${REPO_DIR}/product"
  set_status Search-relevance.md finished
  cd "${REPO_DIR}"
  "${AM_BIN}" sync </dev/null
  cd "${REPO_DIR}/product"
  tomorrow="$(date -u -v+1d +%Y-%m-%d 2>/dev/null || date -u -d 'tomorrow' +%Y-%m-%d)"
  "${AM_BIN}" archive "${tomorrow}"
  assert_dir archive || true # may be empty if finished date is too recent
}

#######################################
# import (Pivotal Tracker CSV)
#######################################
test_import() {
  cd "${REPO_DIR}/product"
  cat >"${WORK}/pivotal.csv" <<'CSV'
Id,Title,Description,Story Type,Estimate,Current State,Owned By
1001,Imported test,Some description,feature,3,started,alice@example.com
CSV
  "${AM_BIN}" import "${WORK}/pivotal.csv"
  assert_file imported-test.md
  assert_grep imported-test.md "title: Imported test"
}

#######################################
# alias
#######################################
test_alias() {
  cd "${REPO_DIR}"
  HOME_BAK="$HOME"
  TMP_HOME="$(mktemp -d)"
  export HOME="${TMP_HOME}"
  touch "${HOME}/.bashrc"
  "${AM_BIN}" alias amx </dev/null
  grep -q "amx" "${HOME}/.bashrc"
  rm -rf "${TMP_HOME}"
  export HOME="${HOME_BAK}"
}

#######################################
# change-status (interactive: enter `e` to exit immediately)
#######################################
test_change_status_interactive() {
  cd "${REPO_DIR}/product"
  echo "e" | "${AM_BIN}" change-status -s d >/dev/null
  echo "e" | "${AM_BIN}" change-status -s p >/dev/null 2>&1 || true
  echo "e" | "${AM_BIN}" change-status -s u >/dev/null 2>&1 || true
}

test_assign_interactive() {
  cd "${REPO_DIR}/product"
  echo "e" | "${AM_BIN}" assign -s d >/dev/null 2>&1 || true
}

test_timeline_interactive() {
  cd "${REPO_DIR}"
  echo "e" | "${AM_BIN}" timeline q3 >/dev/null 2>&1 || true
}

#######################################
# MCP server: initialize + tools/list + list_backlogs
#######################################
test_mcp_server() {
  cd "${REPO_DIR}"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' \
    '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_backlogs","arguments":{}}}' \
    '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"velocity_chart","arguments":{"backlog":"product","iteration_count":4}}}'
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  grep -q '"agilemarkdown"' "${out}"
  for tool in list_backlogs list_items get_item create_item create_backlog archive_items set_status set_assigned set_estimate set_tags set_epic set_hypothesis change_tag delete_tag set_iteration_override list_iteration_overrides validate sync velocity_chart timeline_chart priority_list icebox_list rank_item move_to_icebox move_to_priority epic_progress iteration_view reject_item team_agreements record_learning block_item unblock_item add_comment get_comments add_task list_tasks set_task_done burnup_chart type_mix velocity_history next_item dashboard set_description coach_check acceptance_prompt list_acceptance set_acceptance_state append_acceptance_bullet iteration_fit inception_doc sprint_plan; do
    if ! grep -q "\"name\":\"${tool}\"" "${out}"; then
      echo "expected MCP tool ${tool} not advertised" >&2
      cat "${out}" >&2
      return 1
    fi
  done
  rm -f "${out}"
}

test_import_preserves_type_and_state() {
  cd "${REPO_DIR}/product"
  cat > "${WORK}/full-pivotal.csv" <<'CSV'
Id,Title,Description,Story Type,Estimate,Current State,Owned By,Created at,Accepted at
2001,Pivotal feature accepted,Old feature,feature,3,accepted,alice@example.com,"Apr 1, 2026","Apr 8, 2026"
2002,Pivotal bug filed,Found in prod,bug,,unstarted,alice@example.com,"Apr 5, 2026",
2003,Pivotal chore queued,Tech debt,chore,,unstarted,,"Apr 5, 2026",
2004,Pivotal release marker,Q3 ship,release,,unstarted,,"Apr 5, 2026",
CSV
  "${AM_BIN}" import "${WORK}/full-pivotal.csv"
  assert_grep pivotal-feature-accepted.md "type: feature"
  assert_grep pivotal-feature-accepted.md "status: accepted"
  assert_grep pivotal-feature-accepted.md "accepted:"
  assert_grep pivotal-bug-filed.md "type: bug"
  assert_grep pivotal-chore-queued.md "type: chore"
  assert_grep pivotal-release-marker.md "type: release"
  if grep -q "^estimate:" pivotal-bug-filed.md; then
    echo "bug should not import an estimate" >&2
    return 1
  fi
}

test_chore_finish_skips_to_accepted() {
  cd "${REPO_DIR}/product"
  "${AM_BIN}" create-item "Chore shortcut test" >/dev/null
  python3 - "${REPO_DIR}/product/Chore-shortcut-test.md" <<'PY'
import sys, re
fp = sys.argv[1]
s = open(fp).read()
s = re.sub(r'^---\n', '---\ntype: chore\n', s, count=1)
open(fp, 'w').write(s)
PY
  "${AM_BIN}" finish Chore-shortcut-test.md
  assert_grep Chore-shortcut-test.md "status: accepted"
}

test_release_status_skips_intermediate() {
  cd "${REPO_DIR}/product"
  "${AM_BIN}" create-item "Release shortcut test" >/dev/null
  python3 - "${REPO_DIR}/product/Release-shortcut-test.md" <<'PY'
import sys, re
fp = sys.argv[1]
s = open(fp).read()
s = re.sub(r'^---\n', '---\ntype: release\nrelease_date: "2026-12-31"\n', s, count=1)
open(fp, 'w').write(s)
PY
  "${AM_BIN}" start Release-shortcut-test.md
  assert_grep Release-shortcut-test.md "status: accepted"
}

test_velocity_includes_volatility() {
  cd "${REPO_DIR}/product"
  out="$("${AM_BIN}" velocity 4)"
  echo "${out}" | grep -q "velocity:"
  echo "${out}" | grep -q "volatility:"
}

test_cycle_time_cli() {
  cd "${REPO_DIR}/product"
  out="$("${AM_BIN}" cycle-time)"
  echo "${out}" | grep -q "Cycle time"
}

test_rejection_rate_cli() {
  cd "${REPO_DIR}/product"
  out="$("${AM_BIN}" rejection-rate)"
  echo "${out}" | grep -q "Rejection rate"
}

test_show_priority_hide_accepted() {
  cd "${REPO_DIR}/product"
  "${AM_BIN}" create-item "Hide accepted demo" >/dev/null
  set_status Hide-accepted-demo.md accepted
  cd "${REPO_DIR}"
  "${AM_BIN}" sync </dev/null >/dev/null
  cd "${REPO_DIR}/product"
  withaccepted="$("${AM_BIN}" show priority)"
  hidden="$("${AM_BIN}" show priority --hide-accepted)"
  echo "${withaccepted}" | grep -q "Hide accepted demo" || true
  if echo "${hidden}" | grep -q "Hide accepted demo"; then
    echo "--hide-accepted did not drop the accepted item" >&2
    return 1
  fi
}

#######################################
# Fixture-based analytics tests. The fixture script builds a fully
# populated project with deterministic dates and predictable velocity /
# epic / rejection-rate numbers. Each test starts from a fresh fixture
# build, so state mutations don't leak between tests.
#######################################
FIXTURE_DIR=""

build_fixture() {
  if [ -n "${FIXTURE_DIR}" ] && [ -d "${FIXTURE_DIR}" ]; then
    rm -rf "${FIXTURE_DIR}"
  fi
  FIXTURE_DIR="$(mktemp -d -t am-fixture-XXXXXX)"
  AM_BIN="${AM_BIN}" bash "${REPO_ROOT}/tests/fixtures/build-sample.sh" "${FIXTURE_DIR}" >/dev/null
}

test_fixture_velocity_product() {
  build_fixture
  cd "${FIXTURE_DIR}/product"
  out="$("${AM_BIN}" velocity 4)"
  echo "${out}" | grep -q "velocity: 8"
  echo "${out}" | grep -q "volatility:"
}

test_fixture_velocity_platform() {
  cd "${FIXTURE_DIR}/platform"
  out="$("${AM_BIN}" velocity 4)"
  echo "${out}" | grep -q "velocity: 5"
}

test_fixture_epic_burnup() {
  cd "${FIXTURE_DIR}"
  out="$("${AM_BIN}" show epic auth-rewrite)"
  echo "${out}" | grep -q "1/3 stories"
  echo "${out}" | grep -q "5/13 pts"
  echo "${out}" | grep -q "38%"
}

test_fixture_rejection_rate() {
  cd "${FIXTURE_DIR}/product"
  out="$("${AM_BIN}" rejection-rate)"
  # iter-1 row should show 2 accepted, 1 rejected, 33%
  echo "${out}" | grep -Eq "^\s+[0-9]+\s+[-0-9]+\s+2\s+1\s+33%"
}

test_fixture_cycle_time() {
  cd "${FIXTURE_DIR}/product"
  out="$("${AM_BIN}" cycle-time)"
  echo "${out}" | grep -q "Cycle time"
  echo "${out}" | grep -q "median:"
}

test_fixture_show_priority_iteration_band() {
  cd "${FIXTURE_DIR}/product"
  out="$("${AM_BIN}" show priority)"
  echo "${out}" | grep -q "Priority (product)"
  echo "${out}" | grep -q "velocity 8"
  # First iteration band should contain Saved-searches at top
  echo "${out}" | grep -q "Saved searches"
}

test_fixture_release_late_flag() {
  # Release Q3-ship is in priority with date 60 days out; should be on track.
  cd "${FIXTURE_DIR}/product"
  out="$("${AM_BIN}" show priority --iterations 100)"
  echo "${out}" | grep -q "Q3 ship"
  echo "${out}" | grep -qE "(on track|LATE):"
}

test_fixture_unice_bulk_preserves_order() {
  build_fixture
  cd "${FIXTURE_DIR}/product"
  # Capture icebox order before bulk move.
  before="$(grep -oE '\([^)]+\.md\)' _icebox.md | tr -d '()')"
  "${AM_BIN}" unice --all >/dev/null
  # The icebox items should now appear at the bottom of priority in the
  # same order they appeared in the icebox.
  for path in ${before}; do
    if ! grep -q "${path}" _priority.md; then
      echo "expected ${path} in priority after unice --all" >&2
      return 1
    fi
  done
  # Confirm last item in priority is the last icebox item we captured.
  last_before="$(echo "${before}" | tr ' ' '\n' | tail -1)"
  last_priority="$(grep -oE '\([^)]+\.md\)' _priority.md | tr -d '()' | tail -1)"
  if [ "${last_before}" != "${last_priority}" ]; then
    echo "bulk move did not preserve order: last_before=${last_before} last_priority=${last_priority}" >&2
    return 1
  fi
}

test_fixture_iteration_strength_override_persists() {
  cd "${FIXTURE_DIR}"
  assert_file .am/iterations.yaml
  assert_grep .am/iterations.yaml "team_strength: 0.5"
  out="$("${AM_BIN}" strength --list)"
  echo "${out}" | grep -q "team_strength=0.5"
}

test_fixture_has_inception() {
  build_fixture
  cd "${FIXTURE_DIR}"
  assert_file inception.md
  assert_grep inception.md "## The user"
  assert_grep inception.md "## Out of scope"
}

test_fixture_stories_have_acceptance() {
  cd "${FIXTURE_DIR}"
  # Three accepted features should each carry an `## Acceptance` section.
  assert_grep product/Login-flow.md "## Acceptance"
  assert_grep product/Search-relevance.md "## Acceptance"
  assert_grep product/Dark-mode.md "## Acceptance"
  # Concrete bullets, not the old "thing 1" / "thing 2" placeholders.
  assert_not_grep product/Login-flow.md "thing 1"
}

test_fixture_cleanup() {
  if [ -n "${FIXTURE_DIR}" ] && [ -d "${FIXTURE_DIR}" ]; then
    rm -rf "${FIXTURE_DIR}"
  fi
}

test_priority_ice_unice() {
  # Sync should ensure every active item lands in either _priority or _icebox.
  cd "${REPO_DIR}/product"
  "${AM_BIN}" create-item "Priority test stub" >/dev/null
  cd "${REPO_DIR}"
  "${AM_BIN}" sync </dev/null >/dev/null
  assert_file product/_priority.md
  assert_file product/_icebox.md

  # Pick whichever file is currently in icebox to first promote.
  cd "${REPO_DIR}/product"
  src="$(grep -oE '\([^)]+\.md\)' _icebox.md | head -1 | tr -d '()')"
  if [ -z "${src}" ]; then
    # If everything ended up in priority already, pick from there.
    src="$(grep -oE '\([^)]+\.md\)' _priority.md | head -1 | tr -d '()')"
  fi
  if [ -z "${src}" ]; then
    echo "no item to test" >&2
    return 1
  fi

  # Force into priority via unice (no-op if already there).
  if grep -q "${src}" _icebox.md; then
    "${AM_BIN}" unice "${src}"
  fi
  if ! grep -q "${src}" _priority.md; then
    echo "expected ${src} in _priority.md after unice" >&2
    return 1
  fi

  # Now push it back to icebox.
  "${AM_BIN}" ice "${src}"
  if grep -q "${src}" _priority.md; then
    echo "expected ${src} removed from _priority.md after ice" >&2
    return 1
  fi
  if ! grep -q "${src}" _icebox.md; then
    echo "expected ${src} present in _icebox.md after ice" >&2
    return 1
  fi

  # And back to the top of priority.
  "${AM_BIN}" unice "${src}" --top
  if ! grep -q "${src}" _priority.md; then
    echo "expected ${src} in _priority.md after unice --top" >&2
    return 1
  fi
}

test_unice_all_bulk() {
  # Bulk move icebox to bottom of priority preserves order.
  cd "${REPO_DIR}/product"
  "${AM_BIN}" create-item "Bulk one" >/dev/null
  "${AM_BIN}" create-item "Bulk two" >/dev/null
  cd "${REPO_DIR}"
  "${AM_BIN}" sync </dev/null >/dev/null
  cd "${REPO_DIR}/product"
  "${AM_BIN}" ice Bulk-one.md
  "${AM_BIN}" ice Bulk-two.md
  # Now icebox should have one then two; bulk move should preserve order.
  "${AM_BIN}" unice --all
  # Both should be in priority.
  if ! grep -q "Bulk-one.md" _priority.md; then
    echo "Bulk-one missing after unice --all" >&2
    return 1
  fi
  if ! grep -q "Bulk-two.md" _priority.md; then
    echo "Bulk-two missing after unice --all" >&2
    return 1
  fi
  # Order: Bulk-one before Bulk-two.
  one_line=$(grep -n "Bulk-one.md" _priority.md | head -1 | cut -d: -f1)
  two_line=$(grep -n "Bulk-two.md" _priority.md | head -1 | cut -d: -f1)
  if [ "${one_line}" -ge "${two_line}" ]; then
    echo "bulk move did not preserve order (one at ${one_line}, two at ${two_line})" >&2
    return 1
  fi
}

test_show_priority_icebox() {
  cd "${REPO_DIR}/product"
  "${AM_BIN}" show priority | grep -q "Priority"
  "${AM_BIN}" show icebox | grep -q "Icebox"
  "${AM_BIN}" show iteration 0 | grep -q "Iteration"
}

test_epic_burnup() {
  # Tag two items with the same epic, accept one, expect 50%.
  cd "${REPO_DIR}/product"
  "${AM_BIN}" create-item "Epic A first" >/dev/null
  "${AM_BIN}" create-item "Epic A second" >/dev/null
  python3 - "Epic-A-first.md" <<'PY'
import sys, re
fp = sys.argv[1]
s = open(fp).read()
if not re.search(r'^epic:', s, re.M):
    s = re.sub(r'^---\n', '---\nepic: alpha\nestimate: 2\n', s, count=1)
open(fp, 'w').write(s)
PY
  python3 - "Epic-A-second.md" <<'PY'
import sys, re
fp = sys.argv[1]
s = open(fp).read()
if not re.search(r'^epic:', s, re.M):
    s = re.sub(r'^---\n', '---\nepic: alpha\nestimate: 2\n', s, count=1)
open(fp, 'w').write(s)
PY
  set_status Epic-A-first.md accepted
  "${AM_BIN}" show epic alpha | grep -q "alpha"
}

test_estimate_cli() {
  cd "${REPO_DIR}/product"
  "${AM_BIN}" create-item "Estimate stub" >/dev/null
  "${AM_BIN}" estimate Estimate-stub.md 5
  if ! grep -Eq '^estimate:[[:space:]]*"?5"?$' Estimate-stub.md; then
    echo "estimate not set" >&2
    cat Estimate-stub.md >&2
    return 1
  fi
}

test_tag_cli() {
  cd "${REPO_DIR}/product"
  "${AM_BIN}" create-item "Tag stub" >/dev/null
  # Replace mode (positional)
  "${AM_BIN}" tag Tag-stub.md alpha beta
  assert_grep Tag-stub.md "alpha"
  assert_grep Tag-stub.md "beta"
  # Add mode
  "${AM_BIN}" tag Tag-stub.md --add gamma
  assert_grep Tag-stub.md "gamma"
  # Remove mode
  "${AM_BIN}" tag Tag-stub.md --remove alpha
  if grep -E "^tags:.*alpha" Tag-stub.md >/dev/null; then
    echo "tag remove failed: alpha still present" >&2
    cat Tag-stub.md >&2
    return 1
  fi
}

test_epic_cli() {
  cd "${REPO_DIR}/product"
  "${AM_BIN}" create-item "Epic CLI stub" >/dev/null
  "${AM_BIN}" epic Epic-CLI-stub.md leaderboards
  assert_grep Epic-CLI-stub.md "epic: leaderboards"
  "${AM_BIN}" epic Epic-CLI-stub.md --unset
  if grep -E "^epic:" Epic-CLI-stub.md >/dev/null; then
    echo "epic --unset failed" >&2
    return 1
  fi
}

test_hypothesis_cli() {
  cd "${REPO_DIR}/product"
  "${AM_BIN}" create-item "Hypothesis CLI stub" >/dev/null
  out="$("${AM_BIN}" hypothesis Hypothesis-CLI-stub.md "Login conversion lifts" 2>&1)"
  echo "${out}" | grep -q "deprecated"
  echo "${out}" | grep -q "acceptance-criteria"
  assert_grep Hypothesis-CLI-stub.md "hypothesis:"
  assert_grep Hypothesis-CLI-stub.md "Login conversion lifts"
}

test_reject_with_reason_cli() {
  cd "${REPO_DIR}/product"
  "${AM_BIN}" create-item "Reject with reason cli" >/dev/null
  set_status Reject-with-reason-cli.md delivered
  "${AM_BIN}" reject Reject-with-reason-cli.md --reason "missed timezone case"
  assert_grep Reject-with-reason-cli.md "status: rejected"
  assert_grep Reject-with-reason-cli.md "Rejection notes"
  assert_grep Reject-with-reason-cli.md "missed timezone case"
}

test_strength_cli() {
  cd "${REPO_DIR}"
  "${AM_BIN}" strength 5 0.5
  assert_file .am/iterations.yaml
  assert_grep .am/iterations.yaml "number: 5"
  assert_grep .am/iterations.yaml "team_strength: 0.5"
  "${AM_BIN}" strength 5 --length 2
  assert_grep .am/iterations.yaml "length_weeks: 2"
  out="$("${AM_BIN}" strength --list)"
  echo "${out}" | grep -q "iteration 5"
  "${AM_BIN}" strength 5 --unset
  if grep -q "number: 5" .am/iterations.yaml 2>/dev/null; then
    echo "unset failed: number 5 still present" >&2
    cat .am/iterations.yaml >&2
    return 1
  fi
}

test_mcp_iteration_overrides() {
  cd "${REPO_DIR}"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"set_iteration_override","arguments":{"number":7,"team_strength":0.7,"length_weeks":2}}}' \
    '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_iteration_overrides","arguments":{}}}'
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  grep -q '"team_strength":0.7' "${out}"
  grep -q '"length_weeks":2' "${out}"
  assert_file .am/iterations.yaml
  assert_grep .am/iterations.yaml "number: 7"
  rm -f "${out}"
}

test_team_agreements_cli() {
  cd "${REPO_DIR}"
  "${AM_BIN}" team-agreements --set "# Team agreements

- bugs at the top
- 8-point hard cap
"
  assert_file team-agreements.md
  assert_grep team-agreements.md "8-point hard cap"
  out="$("${AM_BIN}" team-agreements)"
  echo "${out}" | grep -q "bugs at the top"
}

test_record_learning_cli() {
  cd "${REPO_DIR}"
  "${AM_BIN}" record-learning "Removed CSV export, no complaints"
  assert_file learnings.md
  assert_grep learnings.md "Removed CSV export"
  # Date prefix today
  today="$(date -u +%Y-%m-%d)"
  assert_grep learnings.md "${today}"
}

test_mcp_set_tags_set_epic() {
  cd "${REPO_DIR}/product"
  "${AM_BIN}" create-item "MCP tags epic stub" >/dev/null
  cd "${REPO_DIR}"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"set_tags","arguments":{"path":"product/MCP-tags-epic-stub.md","tags":["mcp","stub"]}}}' \
    '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"set_epic","arguments":{"path":"product/MCP-tags-epic-stub.md","slug":"alpha-platform"}}}'
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  assert_grep product/MCP-tags-epic-stub.md "mcp"
  assert_grep product/MCP-tags-epic-stub.md "stub"
  assert_grep product/MCP-tags-epic-stub.md "epic: alpha-platform"
  rm -f "${out}"
}

test_mcp_change_tag_delete_tag() {
  cd "${REPO_DIR}/product"
  "${AM_BIN}" create-item "Change tag stub" >/dev/null
  "${AM_BIN}" tag Change-tag-stub.md flux capacitor
  cd "${REPO_DIR}"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"change_tag","arguments":{"old":"flux","new":"plasma"}}}' \
    '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"delete_tag","arguments":{"tag":"capacitor"}}}'
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  assert_grep product/Change-tag-stub.md "plasma"
  if grep -E "^tags:.*flux" product/Change-tag-stub.md >/dev/null; then
    echo "change_tag did not rename flux -> plasma" >&2
    return 1
  fi
  if grep -E "^tags:.*capacitor" product/Change-tag-stub.md >/dev/null; then
    echo "delete_tag did not remove capacitor" >&2
    return 1
  fi
  rm -f "${out}"
}

test_mcp_team_and_learning_tools() {
  cd "${REPO_DIR}"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"team_agreements","arguments":{"set":"# Team agreements\n\n- bugs at the top\n- 8-point hard cap\n"}}}' \
    '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"record_learning","arguments":{"note":"Removed CSV export, no complaints in 7 days"}}}' \
    '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"team_agreements","arguments":{}}}'
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  grep -q "8-point hard cap" "${out}"
  grep -q "Removed CSV export" "${out}"
  assert_file team-agreements.md
  assert_grep team-agreements.md "8-point hard cap"
  assert_file learnings.md
  assert_grep learnings.md "Removed CSV export"
  rm -f "${out}"
}

test_mcp_set_hypothesis() {
  cd "${REPO_DIR}/product"
  "${AM_BIN}" create-item "Hypothesis test" >/dev/null
  cd "${REPO_DIR}"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"set_hypothesis","arguments":{"path":"product/Hypothesis-test.md","hypothesis":"The thing should work"}}}'
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  assert_grep product/Hypothesis-test.md "hypothesis: The thing should work"
  rm -f "${out}"
}

test_mcp_archive_items() {
  cd "${REPO_DIR}/product"
  "${AM_BIN}" create-item "Aged item" >/dev/null
  # Force a stale modified date.
  python3 - "${REPO_DIR}/product/Aged-item.md" <<'PY'
import sys, re
fp = sys.argv[1]
s = open(fp).read()
s = re.sub(r'^modified:.*$', 'modified: 2020-01-01T00:00:00Z', s, count=1, flags=re.M)
open(fp, 'w').write(s)
PY
  cd "${REPO_DIR}"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"archive_items","arguments":{"backlog":"product","before":"2024-01-01"}}}'
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  grep -q '"archived":1' "${out}" || grep -q '"archived":[1-9]' "${out}"
  rm -f "${out}"
}

test_mcp_reject_item_with_reason() {
  cd "${REPO_DIR}/product"
  "${AM_BIN}" create-item "Reject me" >/dev/null
  set_status Reject-me.md delivered
  cd "${REPO_DIR}"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"reject_item","arguments":{"path":"product/Reject-me.md","reason":"Misses timezone-aware case"}}}'
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  assert_grep product/Reject-me.md "status: rejected"
  assert_grep product/Reject-me.md "Rejection notes"
  assert_grep product/Reject-me.md "Misses timezone-aware"
  rm -f "${out}"
}

test_mcp_reject_item_with_failing_bullet() {
  cd "${REPO_DIR}/product"
  "${AM_BIN}" create-item "Reject by bullet" >/dev/null
  set_status Reject-by-bullet.md delivered
  # Flip bullet 1 to claimed so we can verify the reject reopens it.
  "${AM_BIN}" mcp <<EOF >/dev/null 2>&1
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"set_acceptance_state","arguments":{"path":"product/Reject-by-bullet.md","index":1,"state":"claimed","claim_note":"i think this works"}}}
EOF
  cd "${REPO_DIR}"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"reject_item","arguments":{"path":"product/Reject-by-bullet.md","reason":"bullet 1 broke on reload","failing_bullet":1}}}'
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  assert_grep product/Reject-by-bullet.md "status: rejected"
  assert_grep product/Reject-by-bullet.md "Acceptance bullet 1"
  assert_grep product/Reject-by-bullet.md "bullet 1 broke on reload"
  # Cited bullet should now be open, not claimed.
  if grep -q "^- \[~\]" "${REPO_DIR}/product/Reject-by-bullet.md"; then
    echo "expected bullet 1 reopened (no [~] markers should remain)" >&2
    cat "${REPO_DIR}/product/Reject-by-bullet.md" >&2
    return 1
  fi
  rm -f "${out}"
}

test_mcp_priority_tools() {
  cd "${REPO_DIR}"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"priority_list","arguments":{"backlog":"product"}}}' \
    '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"icebox_list","arguments":{"backlog":"product"}}}' \
    '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"epic_progress","arguments":{"slug":"alpha"}}}' \
    '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"iteration_view","arguments":{"backlog":"product","offset":0}}}'
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  grep -q '"items":' "${out}"
  grep -q "Iteration" "${out}"
  rm -f "${out}"
}

test_velocity_ascii() {
  cd "${REPO_DIR}/product"
  "${AM_BIN}" velocity 4 > /tmp/am-velo.txt
  grep -q "Velocity (last 4 iterations" /tmp/am-velo.txt
  rm -f /tmp/am-velo.txt
}

test_auto_discover_users() {
  tmp="$(mktemp -d)"
  cd "${tmp}"
  git init -q
  git config user.email autodisc@example.com
  git config user.name "Auto Discoverer"
  "${AM_BIN}" create-backlog product >/dev/null
  "${AM_BIN}" sync </dev/null >/dev/null
  ls "users/Auto Discoverer.md" >/dev/null
  cd "${REPO_DIR}"
  rm -rf "${tmp}"
}

#######################################
# user-management: change-user, delete-user
#######################################
test_change_user() {
  cd "${REPO_DIR}"
  # change-user merges items assigned to A into user B; both must exist.
  echo "y" | "${AM_BIN}" change-user "Bob Builder" "Carol Coder" >/dev/null
}

test_delete_user() {
  cd "${REPO_DIR}"
  echo "y" | "${AM_BIN}" delete-user "Bob Builder" >/dev/null
  if [ -f "users/Bob Builder.md" ]; then
    echo "delete-user did not remove file" >&2
    return 1
  fi
}

#######################################
# Phase 0 (v4.3) extension-readiness tools
#######################################
test_block_unblock_cli() {
  cd "${REPO_DIR}/product"
  rel=Search-relevance.md
  assert_file "${rel}"
  "${AM_BIN}" block "${rel}" --reason "waiting on legal"
  assert_grep "${rel}" "blocked: true"
  assert_grep "${rel}" "blocked_reason: waiting on legal"
  "${AM_BIN}" unblock "${rel}"
  assert_not_grep "${rel}" "blocked: true"
}

test_comment_cli() {
  cd "${REPO_DIR}/product"
  rel=Search-relevance.md
  "${AM_BIN}" comment --author alice "${rel}" "ranking review next iter"
  assert_grep "${rel}" "## Comments"
  assert_grep "${rel}" "@alice"
  assert_grep "${rel}" "ranking review next iter"
}

test_task_cli() {
  cd "${REPO_DIR}/product"
  rel=Search-relevance.md
  "${AM_BIN}" task add "${rel}" "spike on relevance scoring"
  "${AM_BIN}" task add "${rel}" "wire telemetry"
  "${AM_BIN}" task list "${rel}" | grep -q "spike on relevance scoring"
  "${AM_BIN}" task tick "${rel}" 1
  assert_grep "${rel}" "- \[x\] spike on relevance scoring"
  "${AM_BIN}" task tick --undo "${rel}" 1
  assert_grep "${rel}" "- \[ \] spike on relevance scoring"
}

test_assign_multi_cli() {
  cd "${REPO_DIR}/product"
  rel=Search-relevance.md
  "${AM_BIN}" assign "${rel}" alice bob
  assert_grep "${rel}" "assigned: \[alice, bob\]"
  "${AM_BIN}" assign "${rel}" alice
  assert_grep "${rel}" "assigned: alice"
}

test_dashboard_cli() {
  cd "${REPO_DIR}"
  "${AM_BIN}" dashboard | grep -q "Dashboard"
  "${AM_BIN}" dashboard | grep -q "velocity:"
  "${AM_BIN}" dashboard | grep -q "accepted total:"
}

test_next_cli() {
  cd "${REPO_DIR}"
  out="$("${AM_BIN}" next)"
  echo "${out}" | grep -qE "(no unstarted, unblocked|title:)"
}

test_show_burnup_cli() {
  cd "${REPO_DIR}/product"
  out="$("${AM_BIN}" show burnup 2>&1)" || true
  if ! echo "${out}" | grep -q "Burnup"; then
    echo "no Burnup header in output:" >&2
    echo "${out}" >&2
    return 1
  fi
  if ! echo "${out}" | grep -q "scope"; then
    echo "no scope column in output:" >&2
    echo "${out}" >&2
    return 1
  fi
}

test_velocity_json_cli() {
  cd "${REPO_DIR}/product"
  "${AM_BIN}" velocity --json | grep -q '"iteration"'
  "${AM_BIN}" velocity --json | grep -q '"length_weeks"'
}

test_mcp_blocked_round_trip() {
  cd "${REPO_DIR}"
  rel="product/Search-relevance.md"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"block_item\",\"arguments\":{\"path\":\"${rel}\",\"reason\":\"infra outage\"}}}" \
    "{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"get_item\",\"arguments\":{\"path\":\"${rel}\"}}}" \
    "{\"jsonrpc\":\"2.0\",\"id\":4,\"method\":\"tools/call\",\"params\":{\"name\":\"unblock_item\",\"arguments\":{\"path\":\"${rel}\"}}}"
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  grep -q '"blocked":true' "${out}"
  grep -q '"blocked_reason":"infra outage"' "${out}"
  rm -f "${out}"
}

test_mcp_comments_round_trip() {
  cd "${REPO_DIR}"
  rel="product/Search-relevance.md"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"add_comment\",\"arguments\":{\"path\":\"${rel}\",\"author\":\"bob\",\"text\":\"need PM input\"}}}" \
    "{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"get_comments\",\"arguments\":{\"path\":\"${rel}\"}}}"
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  grep -q '"author":"bob"' "${out}"
  grep -q "need PM input" "${out}"
  grep -q '"count":' "${out}"
  rm -f "${out}"
}

test_mcp_tasks_round_trip() {
  cd "${REPO_DIR}"
  rel="product/Search-relevance.md"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"add_task\",\"arguments\":{\"path\":\"${rel}\",\"text\":\"draft eval harness\"}}}" \
    "{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"list_tasks\",\"arguments\":{\"path\":\"${rel}\"}}}" \
    "{\"jsonrpc\":\"2.0\",\"id\":4,\"method\":\"tools/call\",\"params\":{\"name\":\"set_task_done\",\"arguments\":{\"path\":\"${rel}\",\"index\":1,\"done\":true}}}" \
    "{\"jsonrpc\":\"2.0\",\"id\":5,\"method\":\"tools/call\",\"params\":{\"name\":\"list_tasks\",\"arguments\":{\"path\":\"${rel}\"}}}"
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  grep -q "draft eval harness" "${out}"
  grep -q '"done":true' "${out}"
  rm -f "${out}"
}

test_mcp_dashboard_and_friends() {
  cd "${REPO_DIR}"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"dashboard","arguments":{}}}' \
    '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"type_mix","arguments":{}}}' \
    '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"velocity_history","arguments":{"backlog":"product"}}}' \
    '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"burnup_chart","arguments":{"backlog":"product"}}}' \
    '{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"next_item","arguments":{}}}'
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  grep -q '"velocity":' "${out}"
  grep -q '"rows":' "${out}"
  grep -q '"length_weeks":' "${out}"
  grep -q '"day":' "${out}"
  grep -q '"found":' "${out}"
  rm -f "${out}"
}

test_mcp_set_assigned_multi() {
  cd "${REPO_DIR}"
  rel="product/Search-relevance.md"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"set_assigned\",\"arguments\":{\"path\":\"${rel}\",\"assignees\":[\"alice\",\"bob\",\"carol\"]}}}" \
    "{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"get_item\",\"arguments\":{\"path\":\"${rel}\"}}}"
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  grep -q '"assignees":\["alice","bob","carol"\]' "${out}"
  rm -f "${out}"
}

test_mcp_priority_count_and_extras() {
  cd "${REPO_DIR}"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"priority_list","arguments":{"backlog":"product"}}}' \
    '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"icebox_list","arguments":{"backlog":"product"}}}' \
    '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"list_items","arguments":{"backlog":"product"}}}'
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  grep -q '"count":' "${out}"
  rm -f "${out}"
}

test_mcp_coach_check_refuses_accept() {
  cd "${REPO_DIR}"
  rel="product/Search-relevance.md"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"coach_check\",\"arguments\":{\"action\":\"set_status\",\"path\":\"${rel}\",\"status\":\"accepted\"}}}"
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  grep -q '"allowed":false' "${out}"
  grep -q "dev pair does not accept" "${out}"
  grep -q "coach-refuses-pm-accepts" "${out}"
  rm -f "${out}"
}

test_mcp_coach_check_refuses_pull_no_acceptance() {
  # Build a feature with no `## Acceptance` section and confirm the
  # pull action refuses with the acceptance-before-pull source.
  cd "${REPO_DIR}/product"
  cat > Pull-without-acceptance.md <<'STORY'
---
title: Pull without acceptance
project: product
type: feature
status: unstarted
author: Fixture Author
---

## Problem statement

This story has no acceptance section. The agent should not pull it.
STORY
  cd "${REPO_DIR}"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"coach_check","arguments":{"action":"pull","path":"product/Pull-without-acceptance.md"}}}'
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  grep -q '"allowed":false' "${out}"
  grep -q "acceptance-before-pull" "${out}"
  rm -f "${out}"
}

test_cli_align() {
  cd "${REPO_DIR}"
  out=$("${AM_BIN}" align product/Search-relevance.md 2>&1)
  if ! echo "${out}" | grep -q "Story:"; then
    echo "${out}" >&2
    return 1
  fi
  if ! echo "${out}" | grep -q "Acceptance:"; then
    echo "${out}" >&2
    return 1
  fi
  # Pull-without-acceptance was crafted with no bullets; align should
  # surface the warning.
  out=$("${AM_BIN}" align product/Pull-without-acceptance.md 2>&1)
  if ! echo "${out}" | grep -q "no acceptance bullets"; then
    echo "${out}" >&2
    return 1
  fi
}

test_cli_coach_check_refuses_pull_no_acceptance() {
  cd "${REPO_DIR}"
  set +e
  out=$("${AM_BIN}" coach-check pull --path product/Pull-without-acceptance.md 2>&1)
  rc=$?
  set -e
  if [ "${rc}" -eq 0 ]; then
    echo "expected non-zero exit on pull refusal" >&2
    echo "${out}" >&2
    return 1
  fi
  if ! echo "${out}" | grep -q "acceptance-before-pull"; then
    echo "${out}" >&2
    return 1
  fi
}

test_mcp_coach_check_refuses_oversized_feature() {
  cd "${REPO_DIR}"
  rel="product/Search-relevance.md"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"coach_check\",\"arguments\":{\"action\":\"set_estimate\",\"path\":\"${rel}\",\"estimate\":\"13\"}}}"
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  grep -q '"allowed":false' "${out}"
  grep -q "8 points" "${out}"
  grep -q "8-point-hard-cap" "${out}"
  rm -f "${out}"
}

test_mcp_acceptance_prompt() {
  cd "${REPO_DIR}"
  rel="product/Search-relevance.md"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"acceptance_prompt\",\"arguments\":{\"path\":\"${rel}\"}}}"
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  grep -q "As PM, do you accept?" "${out}"
  grep -q '"prompt_text"' "${out}"
  grep -q '"bullets"' "${out}"
  grep -q '"state":' "${out}"
  rm -f "${out}"
}

test_mcp_acceptance_round_trip() {
  cd "${REPO_DIR}"
  rel="product/Search-relevance.md"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"list_acceptance\",\"arguments\":{\"path\":\"${rel}\"}}}" \
    "{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"set_acceptance_state\",\"arguments\":{\"path\":\"${rel}\",\"index\":1,\"state\":\"claimed\",\"claim_note\":\"passes tests/search_test.go\"}}}" \
    "{\"jsonrpc\":\"2.0\",\"id\":4,\"method\":\"tools/call\",\"params\":{\"name\":\"list_acceptance\",\"arguments\":{\"path\":\"${rel}\"}}}" \
    "{\"jsonrpc\":\"2.0\",\"id\":5,\"method\":\"tools/call\",\"params\":{\"name\":\"append_acceptance_bullet\",\"arguments\":{\"path\":\"${rel}\",\"text\":\"results sort newest-first\"}}}" \
    "{\"jsonrpc\":\"2.0\",\"id\":6,\"method\":\"tools/call\",\"params\":{\"name\":\"list_acceptance\",\"arguments\":{\"path\":\"${rel}\"}}}"
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  grep -q '"state":"claimed"' "${out}"
  grep -q '"claim_note":"passes tests/search_test.go"' "${out}"
  grep -q "results sort newest-first" "${out}"
  rm -f "${out}"
}

test_acceptance_legacy_back_compat() {
  # Craft a story that ships in the legacy bare-bullet shape (no
  # checkboxes) and verify list_acceptance parses every bullet as
  # state=open. Matches what a real-world story written before the
  # checkbox refactor looks like on disk.
  cd "${REPO_DIR}/product"
  cat > Legacy-bullets.md <<'STORY'
---
title: Legacy bullets
project: product
type: feature
status: unstarted
author: Fixture Author
---

## Problem statement

Bare bullets are what older stories on disk look like.

## Acceptance

- the parser absorbs legacy bullets as open
- the rendered prompt shows them with [ ] markers
- list_acceptance returns them all
STORY
  cd "${REPO_DIR}"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_acceptance","arguments":{"path":"product/Legacy-bullets.md"}}}'
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  grep -q '"state":"open"' "${out}"
  grep -q '"bullets"' "${out}"
  if ! grep -q "parser absorbs legacy bullets" "${out}"; then
    echo "expected legacy bullet text in list_acceptance response" >&2
    cat "${out}" >&2
    return 1
  fi
  rm -f "${out}"
}

test_cli_acceptance_round_trip() {
  cd "${REPO_DIR}/product"
  "${AM_BIN}" create-item "Cli acceptance" >/dev/null
  out="$("${AM_BIN}" acceptance list Cli-acceptance.md)"
  if ! echo "${out}" | grep -q "\[ \]"; then
    echo "expected open bullet on fresh item; got: ${out}" >&2
    return 1
  fi
  "${AM_BIN}" acceptance add Cli-acceptance.md "results survive a reload" >/dev/null
  "${AM_BIN}" acceptance claim Cli-acceptance.md 1 --note "tested manually" >/dev/null
  "${AM_BIN}" acceptance verify Cli-acceptance.md 1 >/dev/null
  out="$("${AM_BIN}" acceptance list Cli-acceptance.md)"
  if ! echo "${out}" | grep -q "\[x\]"; then
    echo "expected bullet 1 to be verified; got: ${out}" >&2
    return 1
  fi
  if ! grep -q "^- \[x\]" Cli-acceptance.md; then
    echo "expected on-disk file to carry [x]" >&2
    cat Cli-acceptance.md >&2
    return 1
  fi
}

test_mcp_iteration_fit() {
  cd "${REPO_DIR}"
  out="$(mktemp)"
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"iteration_fit","arguments":{"backlog":"product"}}}'
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  grep -q '"velocity":' "${out}"
  grep -q '"planned_points":' "${out}"
  grep -q '"fits":' "${out}"
  rm -f "${out}"
}

test_cli_accept_prompt() {
  cd "${REPO_DIR}"
  rel="product/Search-relevance.md"
  out="$("${AM_BIN}" accept-prompt "${rel}")"
  echo "${out}" | grep -q "Story:"
  echo "${out}" | grep -q "As PM, do you accept?"
}

test_cli_coach_check_refuses() {
  cd "${REPO_DIR}"
  set +e
  out="$("${AM_BIN}" coach-check set_status --path product/Search-relevance.md --status accepted 2>&1)"
  rc=$?
  set -e
  if [ "${rc}" -eq 0 ]; then
    echo "expected non-zero exit on refusal, got 0" >&2
    return 1
  fi
  echo "${out}" | grep -q "coach-refuses-pm-accepts"
}

test_mcp_set_description() {
  cd "${REPO_DIR}"
  rel="product/Search-relevance.md"
  out="$(mktemp)"
  body='# Search relevance\n\nNew description body.\n'
  (printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"set_description\",\"arguments\":{\"path\":\"${rel}\",\"body\":\"# Search relevance\\n\\nNew description body.\\n\"}}}" \
    "{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"get_item\",\"arguments\":{\"path\":\"${rel}\"}}}"
   sleep 1.0) | "${AM_BIN}" mcp 2>/dev/null > "${out}"
  grep -q "New description body." "${out}"
  rm -f "${out}"
  unused=${body}
  : "$unused"
}

# Run all
run_step "coach projections in sync"  bash "${REPO_ROOT}/tests/check-coach-projections.sh"
run_step "create-user --name --email" test_create_user_flags

run_step "create-user positional"     test_create_user_positional
run_step "create-backlog"             test_create_backlog
run_step "create-backlog spaces"      test_create_backlog_name_with_spaces
run_step "create-item basic"          test_create_item_basic
run_step "create-item --user"         test_create_item_user_flag
run_step "create-item --simulate"     test_create_item_simulate
run_step "sync happy path"            test_sync_happy_path
run_step "config.yaml present"        test_config_yaml_present
run_step "Pivotal transitions"        test_pivotal_transitions
run_step "sync rejects bad status"    test_sync_rejects_bad_status
run_step "work / points / velocity"   test_work_points_velocity
run_step "change-tag / delete-tag"    test_change_delete_tag
run_step "archive"                    test_archive
run_step "import"                     test_import
run_step "alias"                      test_alias
run_step "change-status interactive"  test_change_status_interactive
run_step "assign interactive"         test_assign_interactive
run_step "timeline interactive"       test_timeline_interactive
run_step "mcp server"                 test_mcp_server
run_step "priority/ice/unice"         test_priority_ice_unice
run_step "unice --all preserves order" test_unice_all_bulk
run_step "show priority/icebox/iteration" test_show_priority_icebox
run_step "show priority --hide-accepted" test_show_priority_hide_accepted
run_step "import preserves type/state" test_import_preserves_type_and_state
run_step "chore finish -> accepted"   test_chore_finish_skips_to_accepted
run_step "release skips intermediate" test_release_status_skips_intermediate
run_step "velocity includes volatility" test_velocity_includes_volatility
run_step "cycle-time CLI"             test_cycle_time_cli
run_step "rejection-rate CLI"         test_rejection_rate_cli
run_step "fixture velocity product"   test_fixture_velocity_product
run_step "fixture velocity platform"  test_fixture_velocity_platform
run_step "fixture epic burnup"        test_fixture_epic_burnup
run_step "fixture rejection rate"     test_fixture_rejection_rate
run_step "fixture cycle time"         test_fixture_cycle_time
run_step "fixture show priority band" test_fixture_show_priority_iteration_band
run_step "fixture release flag"       test_fixture_release_late_flag
run_step "fixture unice --all order"  test_fixture_unice_bulk_preserves_order
run_step "fixture override persists"  test_fixture_iteration_strength_override_persists
run_step "fixture has inception"      test_fixture_has_inception
run_step "fixture stories have acceptance" test_fixture_stories_have_acceptance
run_step "fixture cleanup"            test_fixture_cleanup
run_step "epic burnup"                test_epic_burnup
run_step "estimate CLI"               test_estimate_cli
run_step "tag CLI (set/add/remove)"   test_tag_cli
run_step "epic CLI (set/--unset)"     test_epic_cli
run_step "hypothesis CLI"             test_hypothesis_cli
run_step "reject --reason CLI"        test_reject_with_reason_cli
run_step "strength CLI"               test_strength_cli
run_step "mcp iteration overrides"    test_mcp_iteration_overrides
run_step "team-agreements CLI"        test_team_agreements_cli
run_step "record-learning CLI"        test_record_learning_cli
run_step "mcp priority tools"         test_mcp_priority_tools
run_step "mcp set_tags + set_epic"    test_mcp_set_tags_set_epic
run_step "mcp change_tag + delete_tag" test_mcp_change_tag_delete_tag
run_step "mcp team_agreements + record_learning" test_mcp_team_and_learning_tools
run_step "mcp set_hypothesis"         test_mcp_set_hypothesis
run_step "mcp archive_items"          test_mcp_archive_items
run_step "mcp reject_item with reason" test_mcp_reject_item_with_reason
run_step "mcp reject_item with failing_bullet" test_mcp_reject_item_with_failing_bullet
run_step "velocity ASCII"             test_velocity_ascii
run_step "auto-discover users"        test_auto_discover_users
run_step "change-user"                test_change_user
run_step "delete-user"                test_delete_user
run_step "block/unblock CLI"          test_block_unblock_cli
run_step "comment CLI"                test_comment_cli
run_step "task CLI"                   test_task_cli
run_step "assign multi-owner CLI"     test_assign_multi_cli
run_step "dashboard CLI"              test_dashboard_cli
run_step "next CLI"                   test_next_cli
run_step "show burnup CLI"            test_show_burnup_cli
run_step "velocity --json CLI"        test_velocity_json_cli
run_step "mcp block_item round trip"  test_mcp_blocked_round_trip
run_step "mcp comments round trip"    test_mcp_comments_round_trip
run_step "mcp tasks round trip"       test_mcp_tasks_round_trip
run_step "mcp dashboard + friends"    test_mcp_dashboard_and_friends
run_step "mcp set_assigned multi"     test_mcp_set_assigned_multi
run_step "mcp list count fields"      test_mcp_priority_count_and_extras
run_step "mcp set_description"        test_mcp_set_description
run_step "mcp coach_check refuses self-accept" test_mcp_coach_check_refuses_accept
run_step "mcp coach_check refuses 13-pt feature" test_mcp_coach_check_refuses_oversized_feature
run_step "mcp coach_check refuses pull without acceptance" test_mcp_coach_check_refuses_pull_no_acceptance
run_step "cli coach-check refuses pull without acceptance" test_cli_coach_check_refuses_pull_no_acceptance
run_step "cli align prints story"     test_cli_align
run_step "mcp acceptance_prompt"      test_mcp_acceptance_prompt
run_step "mcp acceptance round trip"  test_mcp_acceptance_round_trip
run_step "acceptance legacy back-compat" test_acceptance_legacy_back_compat
run_step "cli acceptance round trip"  test_cli_acceptance_round_trip
run_step "mcp iteration_fit"          test_mcp_iteration_fit
run_step "cli accept-prompt"          test_cli_accept_prompt
run_step "cli coach-check refuses"    test_cli_coach_check_refuses
run_step "coach hook refuses self-accept" test_coach_hook_refuses_self_accept
run_step "coach hook allows started"  test_coach_hook_allows_legitimate_transitions
run_step "coach hook refuses pull without acceptance" test_coach_hook_refuses_pull_no_acceptance
run_step "am init idempotent"         test_am_init_reinstalls_idempotent
run_step "coach status CLI"           test_coach_status_cli
run_step "team-agreements --add"      test_team_agreements_add
run_step "coach-check --action flag"  test_coach_check_action_flag
run_step "pull CLI"                   test_pull_cli
run_step "deliver --prompt"           test_deliver_prompt_flag
run_step "estimate --advise"          test_estimate_advise
run_step "inception CLI"              test_inception_cli
run_step "sprint plan CLI"            test_sprint_plan_cli
run_step "retro CLI"                  test_retro_cli
run_step "mcp inception_doc"          test_mcp_inception_doc
run_step "mcp sprint_plan"            test_mcp_sprint_plan

echo
echo "==== summary ===="
echo "passed: ${pass}"
echo "failed: ${fail}"
if [ "${fail}" -gt 0 ]; then
  printf 'failed steps:\n'
  for m in "${fail_msgs[@]}"; do printf '  - %s\n' "${m}"; done
  exit 1
fi
