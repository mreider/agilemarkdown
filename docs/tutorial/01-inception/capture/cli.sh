#!/usr/bin/env bash
# Chapter 1 actually mutates the scratch workspace so Chapter 2
# picks it up with a backlog in place. CLI snippets capture the
# verbs as a tutorial reader would see them.

set -euo pipefail
CHAPTER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKSPACE="${AM_WORKSPACE:?AM_WORKSPACE not set}"
CLI_OUT="${CHAPTER_DIR}/cli"
mkdir -p "${CLI_OUT}"

cd "${WORKSPACE}"

if [[ ! -d .git ]]; then
  git init -q
  git config user.email tutorial@example.com
  git config user.name "Tutorial"
fi

if [[ ! -d stories ]]; then
  am create-backlog stories >/dev/null
fi

# Seed an inception with realistic prose. The .txt snippets
# show the *template* a reader sees, but the file on disk is
# populated so the rest of the tutorial looks like a real project.
mkdir -p .am
cat > .am/inception.md <<'INCEPTION'
# Inception

## The user

A small team that wants to track letters, gardens, and small
ceremonies without buying a SaaS. They live in a meadow.

## The goal

Keep one shared list of small commitments. Honour them.

## The reason

Toad cannot remember things. Frog is patient but forgetful.
A written list helps; a written list with timestamps helps more.

## Success

By spring the seedling has grown, the letters have been answered,
and the moon has had a birthday.

## Constraints

- One git repo for everything
- No SaaS dependencies
- Decisions in writing

## Out of scope

- Hurrying
- Anything that would upset Bear before the Long-Sleep
INCEPTION

am team-agreements --add "Bugs go to the top of priority" >/dev/null 2>&1 || true
am team-agreements --add "8-point hard cap. Above 3 is a red flag." >/dev/null 2>&1 || true
am team-agreements --add "PM rejects with a reason" >/dev/null 2>&1 || true

cat > "${CLI_OUT}/01-init.txt" <<'EOF'
$ cd projects/meadow-letters
$ git init
Initialized empty Git repository
$ am create-backlog
created backlog: stories
created .am/config.yaml
EOF

cat > "${CLI_OUT}/02-inception-show.txt" <<'EOF'
$ am inception
seeded .am/inception.md (edit it; ask the team)
$ am inception --show
# Inception

## The user
(describe the people you are building for)

## The goal
(one sentence the team agrees with)

## The reason
(why now, why us)

## Success
(what specifically must be true for this to have worked)

## Constraints
- (hard limits on time, scope, dependencies)

## Out of scope
- (what you're consciously not doing)
EOF

cat > "${CLI_OUT}/03-team-agreements.txt" <<'EOF'
$ am team-agreements --add "Bugs go to the top of priority"
recorded: 1 agreement
$ am team-agreements --add "8-point hard cap. Above 3 is a red flag."
recorded: 2 agreements
$ am team-agreements --add "PM rejects with a reason"
recorded: 3 agreements
EOF

echo "  wrote ${CLI_OUT}/*.txt"
