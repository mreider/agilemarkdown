#!/usr/bin/env bash
# Builds a deterministic, populated agilemarkdown project at $1.
# All accepted timestamps are computed relative to today so velocity /
# cycle-time / rejection-rate calculations land in known iteration
# windows regardless of when the test runs.
#
# Project shape:
#   product/    feature-heavy backlog with full state coverage + an epic
#   platform/   smaller backlog with a chore + a delivered feature
#   .am/iterations.yaml    one strength=0.5 override on the iter-2 window
#   team-agreements.md, learnings.md
#
# Acceptance dates land in iter -3 / -2 / -1 (relative to today's
# iteration). With config defaults (1-week iterations) and an iter-2
# strength override of 0.5, the canonical velocity formula computes:
#   product:  numerator = 5 + (3 / 0.5) + 13 = 24
#             denominator = 3 weeks
#             per_week    = 8
#             displayed   = floor(8 * 1) = 8
#   platform: numerator = (8 / 0.5) = 16, denominator = 1 week,
#             per_week  = 16, displayed = 16

set -euo pipefail

if [ -z "${1:-}" ]; then
  echo "usage: build-sample.sh <target-dir>" >&2
  exit 1
fi
TARGET="$1"
AM_BIN="${AM_BIN:-am}"

# Wipe + recreate.
rm -rf "${TARGET}"
mkdir -p "${TARGET}"
cd "${TARGET}"
git init -q
git config user.email fixture@example.com
git config user.name "Fixture Author"

# Compute relative dates. Iter 0 is the current week; iter -N starts N
# weeks before the current iteration's Monday.
ITER0_START=$(python3 - <<'PY'
from datetime import date, timedelta
today = date.today()
mon = today - timedelta(days=today.weekday())  # Monday of current week
print(mon.isoformat())
PY
)
DATE_ITER_M3=$(python3 -c "from datetime import date,timedelta; d=date.fromisoformat('${ITER0_START}')-timedelta(days=21)+timedelta(days=3); print(d.isoformat())")
DATE_ITER_M2=$(python3 -c "from datetime import date,timedelta; d=date.fromisoformat('${ITER0_START}')-timedelta(days=14)+timedelta(days=3); print(d.isoformat())")
DATE_ITER_M1=$(python3 -c "from datetime import date,timedelta; d=date.fromisoformat('${ITER0_START}')-timedelta(days= 7)+timedelta(days=3); print(d.isoformat())")
DATE_RELEASE=$(python3 -c "from datetime import date,timedelta; d=date.fromisoformat('${ITER0_START}')+timedelta(days=60); print(d.isoformat())")

# Iteration numbers (canonical 1-based weeks-since-2000-01-03).
ITER_NUM_M3=$(python3 -c "from datetime import date; epoch=date(2000,1,3); d=date.fromisoformat('${ITER0_START}'); weeks=(d-epoch).days//7 - 3; print(weeks+1)")
ITER_NUM_M2=$(python3 -c "from datetime import date; epoch=date(2000,1,3); d=date.fromisoformat('${ITER0_START}'); weeks=(d-epoch).days//7 - 2; print(weeks+1)")

# Boot the project structure.
"${AM_BIN}" create-backlog product >/dev/null
"${AM_BIN}" create-backlog platform >/dev/null

# Drop the seeded sample stories so we start clean.
rm -f product/Sample-feature-set-up-login.md product/Sample-bug-typo-on-landing.md product/Sample-chore-rotate-api-key.md
rm -f platform/Sample-feature-set-up-login.md platform/Sample-bug-typo-on-landing.md platform/Sample-chore-rotate-api-key.md

# story_body returns a problem-statement + acceptance criteria block
# tailored to the story title. The criteria are specific so the
# acceptance ceremony has something concrete to render. Falls back to
# generic bullets when the title is not in the table.
story_body() {
  local title="$1"
  local marker="${2:-[ ]}"
  case "${title}" in
    "Login flow")
      echo "Users cannot sign into the app."
      echo
      echo "## Acceptance"
      echo
      echo "- ${marker} a registered user can sign in with email + password"
      echo "- ${marker} a wrong password produces a recoverable error"
      echo "- ${marker} the session persists across a page reload"
      ;;
    "Search relevance")
      echo "Search returns too many irrelevant recipes."
      echo
      echo "## Acceptance"
      echo
      echo "- ${marker} typing 'pasta' returns pasta dishes in the top 5"
      echo "- ${marker} typo tolerance: 'pasata' still returns pasta dishes"
      echo "- ${marker} search latency below 500ms p95"
      ;;
    "Dark mode")
      echo "Some users want a dark color scheme."
      echo
      echo "## Acceptance"
      echo
      echo "- ${marker} toggle in settings flips theme"
      echo "- ${marker} the choice persists across sessions"
      echo "- ${marker} system-prefers-dark is the default for new users"
      ;;
    "Saved searches")
      echo "Users want to revisit common search queries."
      echo
      echo "## Acceptance"
      echo
      echo "- ${marker} a 'save this search' button on the results page"
      echo "- ${marker} saved searches appear under My Account"
      echo "- ${marker} a saved search runs again with one click"
      ;;
    "Pagination")
      echo "Result pages over 50 items are slow."
      echo
      echo "## Acceptance"
      echo
      echo "- ${marker} result pages cap at 20 items"
      echo "- ${marker} next / previous navigation works"
      echo "- ${marker} the page query parameter survives a refresh"
      ;;
    "API tokens")
      echo "Third-party integrators need API access."
      echo
      echo "## Acceptance"
      echo
      echo "- ${marker} a user can mint a token from settings"
      echo "- ${marker} the token shows once and never again"
      echo "- ${marker} a revoked token rejects within 60 seconds"
      ;;
    "Login bug")
      echo "Some users are bounced back to the login screen after signing in."
      echo
      echo "## Steps to reproduce"
      echo
      echo "1. enter valid credentials on Safari iOS"
      echo "2. observe redirect loop"
      ;;
    "Search typo")
      echo "Typos in the search bar produce no results."
      ;;
    "Rotate API key")
      echo "The deployment API key is overdue for rotation."
      ;;
    "Auth rewrite phase 1")
      echo "First slice of the auth rewrite: extract the session module."
      echo
      echo "## Acceptance"
      echo
      echo "- ${marker} the session module owns all cookie reads/writes"
      echo "- ${marker} existing tests still pass against the extracted module"
      ;;
    "Auth rewrite phase 2")
      echo "Second slice of the auth rewrite: route protection through middleware."
      echo
      echo "## Acceptance"
      echo
      echo "- ${marker} protected routes go through one middleware function"
      echo "- ${marker} the middleware short-circuits anonymous users with a 401"
      ;;
    "Auth rewrite phase 3")
      echo "Third slice of the auth rewrite: sign-out flow."
      echo
      echo "## Acceptance"
      echo
      echo "- ${marker} a sign-out button clears the session cookie"
      echo "- ${marker} a signed-out user trying a protected route gets the login page"
      ;;
    "Customer CSV")
      echo "Internal team wants to export customer records."
      echo
      echo "## Acceptance"
      echo
      echo "- ${marker} an admin-only page exports a CSV"
      echo "- ${marker} columns: id, email, signup date, plan"
      echo "- ${marker} the export does not include cancelled accounts"
      ;;
    "Q3 ship")
      echo "Quarterly ship marker."
      ;;
    "Postgres 16")
      echo "Move the production database to Postgres 16."
      echo
      echo "## Acceptance"
      echo
      echo "- ${marker} staging is on PG16 and passes the integration suite"
      echo "- ${marker} production migration runs without downtime"
      echo "- ${marker} rollback procedure documented"
      ;;
    "Monitoring")
      echo "Add latency and error-rate dashboards for the API."
      echo
      echo "## Acceptance"
      echo
      echo "- ${marker} p50/p95 latency dashboard on the staging Grafana"
      echo "- 5xx error count alerted to the on-call channel"
      ;;
    "Audit deps")
      echo "Run the dependency audit; rotate anything flagged critical."
      ;;
    *)
      echo "Fixture story: ${title}"
      echo
      echo "## Acceptance"
      echo
      echo "- ${marker} the visible behavior matches the story title"
      echo "- ${marker} the change ships without regressing existing tests"
      ;;
  esac
}

# Helper: write a story with a real body.
#
# Signature: write_story PATH TITLE TYPE STATUS ESTIMATE ACCEPTED EPIC EXTRA [TAGS]
#
# Timestamps are derived from STATUS so the cumulative-flow chart and
# cycle-time analytics have real data to draw:
#   - accepted: started = ACCEPTED - 3d, then finished/delivered/accepted
#   - delivered: started/finished/delivered cluster around ITER_M1
#   - finished: started + finished
#   - started: started only
#   - everything else: no transition timestamps
write_story() {
  local path="$1"; local title="$2"; local type="$3"; local status="$4"; local estimate="$5"
  local accepted="$6"; local epic="$7"; local extra="$8"; local tags="${9:-}"

  # Pick the acceptance-bullet checkbox marker based on status: accepted
  # stories show [x], delivered stories show [~] (claimed but not yet
  # verified), and everything else stays [ ] open.
  local marker="[ ]"
  case "${status}" in
    accepted) marker="[x]" ;;
    delivered) marker="[~]" ;;
  esac

  # Started timestamp: three days before accepted for completed
  # stories, a sensible mid-iteration date for in-flight ones.
  local started=""
  case "${status}" in
    accepted)
      started=$(python3 -c "from datetime import date,timedelta; d=date.fromisoformat('${accepted}')-timedelta(days=3); print(d.isoformat())")
      ;;
    delivered|finished|started)
      started="${DATE_ITER_M1}"
      ;;
  esac

  {
    echo "---"
    echo "title: ${title}"
    echo "project: $(basename "$(dirname "${path}")")"
    echo "type: ${type}"
    echo "status: ${status}"
    echo "author: Fixture Author"
    echo "created: ${DATE_ITER_M3}T09:00:00Z"
    echo "modified: ${DATE_ITER_M1}T09:00:00Z"
    if [ -n "${estimate}" ]; then echo "estimate: ${estimate}"; fi
    if [ -n "${started}" ]; then echo "started: ${started}T09:00:00Z"; fi
    case "${status}" in
      finished)
        echo "finished: ${DATE_ITER_M1}T15:00:00Z"
        ;;
      delivered)
        echo "finished: ${DATE_ITER_M1}T15:00:00Z"
        echo "delivered: ${DATE_ITER_M1}T17:00:00Z"
        ;;
    esac
    if [ -n "${accepted}" ]; then
      echo "finished: ${accepted}T09:00:00Z"
      echo "delivered: ${accepted}T11:00:00Z"
      echo "accepted: ${accepted}T15:00:00Z"
    fi
    if [ -n "${tags}" ]; then echo "tags: [${tags}]"; fi
    if [ -n "${epic}" ]; then echo "epic: ${epic}"; fi
    if [ -n "${extra}" ]; then echo "${extra}"; fi
    echo "---"
    echo
    echo "## Problem statement"
    echo
    story_body "${title}" "${marker}"
    echo
    echo "## Comments"
  } > "${path}"
}

# product backlog: full state coverage + auth-rewrite epic. Filenames
# match the canonical GetValidFileName(title) result so sync does not
# rename them and downstream test commands can target them stably.
write_story product/Login-flow.md            "Login flow"            feature accepted "5" "${DATE_ITER_M3}" "" ""              "auth, ux"
write_story product/Search-relevance.md      "Search relevance"      feature accepted "3" "${DATE_ITER_M2}" "" ""              "search"
write_story product/Dark-mode.md             "Dark mode"             feature accepted "8" "${DATE_ITER_M1}" "" ""              "ux"
write_story product/Saved-searches.md        "Saved searches"        feature started  "5" ""                ""             ""  "search, ux"
write_story product/Pagination.md            "Pagination"            feature finished "3" ""                ""             ""  "search"
write_story product/API-tokens.md            "API tokens"            feature delivered "5" ""               ""             ""  "auth, api"
write_story product/Login-bug.md             "Login bug"             bug     rejected ""  ""                ""             ""  "auth"
write_story product/Search-typo.md           "Search typo"           bug     unstarted "" ""                ""             ""  "search"
write_story product/Rotate-API-key.md        "Rotate API key"        chore   unstarted "" ""                ""             ""  "ops"
write_story product/Auth-rewrite-phase-1.md  "Auth rewrite phase 1"  feature accepted "5" "${DATE_ITER_M1}" "auth-rewrite" ""  "auth"
write_story product/Auth-rewrite-phase-2.md  "Auth rewrite phase 2"  feature started  "3" ""                "auth-rewrite" ""  "auth"
write_story product/Auth-rewrite-phase-3.md  "Auth rewrite phase 3"  feature unstarted "5" ""               "auth-rewrite" ""  "auth"
write_story product/Customer-CSV.md          "Customer CSV"          feature unstarted "" ""                ""             ""  ""
write_story product/Q3-ship.md               "Q3 ship"               release unstarted "" ""               ""             "release_date: ${DATE_RELEASE}" ""

# platform backlog.
write_story platform/Postgres-16.md   "Postgres 16"        feature accepted  "8" "${DATE_ITER_M2}" "" ""  "infra, db"
write_story platform/Monitoring.md    "Monitoring"         feature started   "3" ""                "" ""  "ops"
write_story platform/Audit-deps.md    "Audit deps"         chore   accepted  ""  "${DATE_ITER_M1}" "" ""  "ops"

# Iteration override: iter -2 ran at half strength.
mkdir -p .am
cat > .am/iterations.yaml <<EOF
overrides:
  - number: ${ITER_NUM_M2}
    team_strength: 0.5
EOF

# Inception document for the fixture project.
cat > inception.md <<'INCEPTION'
# Inception

## The user

A small product team running a backlog inside the same repo as their
code. Familiar with git, comfortable in a markdown editor. Want to
keep the iteration loop tight without buying a SaaS.

## The goal

Turn the existing repo into a working agilemarkdown project: stories
visible in priority and icebox, velocity computed from accepted
points, the agent enforcing canon and rendering the PM ceremony.

## The reason

The team has tried Pivotal Tracker, Jira, and Linear. None of them
play well with an LLM as the dev pair. Markdown in git does.

## Success

A new contributor clones the repo, runs `am sync`, and the velocity
ledger plus the priority list match what the PM had in their head.

## Constraints

- one git repo for code and backlog
- no SaaS dependencies
- works offline

## Out of scope

- a separate dashboard UI
- a real-time collaboration layer
- per-user assignments beyond what `assigned:` already supports
INCEPTION

# Team agreements: items the team picked up over the last quarter.
cat > team-agreements.md <<'AGREEMENTS'
# Team agreements

- Bugs go at the top of priority unless intentionally deprioritized.
- 8-point hard cap. Above 3 is a red flag.
- Every feature needs an `## Acceptance` section in the body before pulling.
- PM rejects with a comment so the dev pair sees the reason.
- No merges to main on Friday after lunch.
AGREEMENTS

# Learnings: outputs from recent retros.
cat > learnings.md <<'LEARNINGS'
# Learnings

- 2026-04-15: Removed the legacy CSV export. No complaints in 7 days. Permanent.
- 2026-04-22: Pairing on the Postgres migration turned a 5 into a 2. The seam was the helper, not the migration itself.
- 2026-04-29: Stories without acceptance criteria slipped two iterations. Adopted the agreement: features need `## Acceptance` before pulling.
- 2026-05-06: The 13-pointer that got refused split into a 5 and a 3. The 5 shipped this iteration; the 3 is queued.
LEARNINGS

# Run sync to populate _priority.md, _icebox.md, derived views. After
# sync every active item lands in icebox; we then promote a hand-picked
# set into priority so test functions have stable items to rank against.
"${AM_BIN}" sync </dev/null >/dev/null

# Promote a Pivotal-shaped current iteration into priority. Order
# (top-down): the in-flight features, the bug, the chore.
"${AM_BIN}" unice product/Saved-searches.md       --top >/dev/null
"${AM_BIN}" unice product/Pagination.md           >/dev/null
"${AM_BIN}" unice product/API-tokens.md           >/dev/null
"${AM_BIN}" unice product/Auth-rewrite-phase-2.md >/dev/null
"${AM_BIN}" unice product/Auth-rewrite-phase-3.md >/dev/null
"${AM_BIN}" unice product/Login-bug.md            >/dev/null
"${AM_BIN}" unice product/Rotate-API-key.md       >/dev/null
"${AM_BIN}" unice product/Q3-ship.md              >/dev/null
# Customer-CSV stays in icebox so downstream tests have a canonical
# icebox item to manipulate.

# Add a small trail of commits on Login-flow.md so the per-item
# History panel has multiple entries to render against. Stories
# without targeted edits will still show the sync + unice commits.
{ git add product/Login-flow.md >/dev/null 2>&1 || true; }
{ echo "" >> product/Login-flow.md; git -c user.email=fixture@example.com -c user.name="Fixture Author" commit --quiet -am "Login-flow: clarify password reset wording" >/dev/null 2>&1 || true; }
{ echo "" >> product/Login-flow.md; git -c user.email=fixture@example.com -c user.name="Fixture Author" commit --quiet -am "Login-flow: tighten acceptance bullets" >/dev/null 2>&1 || true; }

# Echo computed numbers so callers can sanity-check.
echo "fixture built at: ${TARGET}"
echo "iter -3 date:     ${DATE_ITER_M3}"
echo "iter -2 date:     ${DATE_ITER_M2}"
echo "iter -1 date:     ${DATE_ITER_M1}"
echo "iter -2 number:   ${ITER_NUM_M2}  (strength override 0.5)"
echo "release date:     ${DATE_RELEASE}"
