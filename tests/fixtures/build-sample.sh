#!/usr/bin/env bash
# Builds a deterministic, populated agilemarkdown project at $1.
#
# The fixture is a small whimsical world in the spirit of Arnold
# Lobel's Frog and Toad: 54 stories, 5 epics, 8 owners (Frog, Toad,
# Mouse, Owl, Snail, Sparrow, Bear, Lizard). None of it is meant to
# look like a real product backlog. It is meant to fill every code
# path the CLI and the VS Code extension touch: every status, every
# epic state, comments, tasks, release markers, an iteration override.
#
# Project shape (single-backlog `stories/` convention):
#   stories/                       54 stories
#   .am/iterations.yaml            iter-2 strength override 0.5
#   .am/inception.md               in voice
#   .am/team-agreements.md         in voice
#   .am/learnings.md               in voice
#   .am/users/                     8 owners created up front
#
# Distribution:
#   18 accepted, spread across iter -6 through iter -1 (3 per week)
#   18 in priority (current iter + planned + 2 release markers,
#       every feature/chore pointed so release dates are computable)
#   18 in icebox (9 pointed, 9 unpointed)
#
# Epics:
#   the-button-quest    4 stories, all accepted  (FINISHED)
#   cookies-in-the-jar  5 stories, all accepted  (FINISHED)
#   the-garden-of-patience  13 stories, mixed    (IN PROGRESS)
#   letters-by-snail    5 stories, mixed         (IN PROGRESS)
#   the-moon-watch      3 stories, all unstarted (NOT STARTED)

set -euo pipefail

if [ -z "${1:-}" ]; then
  echo "usage: build-sample.sh <target-dir>" >&2
  exit 1
fi
TARGET="$1"
AM_BIN="${AM_BIN:-am}"

rm -rf "${TARGET}"
mkdir -p "${TARGET}"
cd "${TARGET}"
git init -q
git config user.email fixture@example.com
git config user.name "Fixture Author"

# Iteration anchors. Iter 0 is the current week; iter -N starts N
# weeks before. All accepted dates fall on a Thursday so the rolling
# velocity window contains them cleanly.
ITER0_START=$(python3 - <<'PY'
from datetime import date, timedelta
today = date.today()
mon = today - timedelta(days=today.weekday())
print(mon.isoformat())
PY
)
date_for() {
  python3 -c "from datetime import date,timedelta; d=date.fromisoformat('${ITER0_START}')-timedelta(days=${1})+timedelta(days=3); print(d.isoformat())"
}
DATE_ITER_M6=$(date_for 42)
DATE_ITER_M5=$(date_for 35)
DATE_ITER_M4=$(date_for 28)
DATE_ITER_M3=$(date_for 21)
DATE_ITER_M2=$(date_for 14)
DATE_ITER_M1=$(date_for 7)
# Current iteration acceptance date. Stories with this timestamp
# render as "accepted this sprint", green chip, top of Backlog,
# above the in-flight rows, and only roll over to Done at next
# Monday's iteration tick.
DATE_ITER_0=$(python3 -c "from datetime import date; print(date.today().isoformat())")
DATE_FROST=$(python3 -c "from datetime import date,timedelta; d=date.fromisoformat('${ITER0_START}')+timedelta(days=30); print(d.isoformat())")
DATE_LONGSLEEP=$(python3 -c "from datetime import date,timedelta; d=date.fromisoformat('${ITER0_START}')+timedelta(days=90); print(d.isoformat())")

ITER_NUM_M2=$(python3 -c "from datetime import date; epoch=date(2000,1,3); d=date.fromisoformat('${ITER0_START}'); weeks=(d-epoch).days//7 - 2; print(weeks+1)")

# canon() mirrors utils.GetValidFileName + the sync-time downcase.
# Computing the canonical name from the title up front means the
# files we write match the paths sync expects, and subsequent rank /
# comment / task commands can address each story by title.
canon() {
  python3 - "$1" <<'PY'
import re, sys
name = sys.argv[1].strip()
name = re.sub(r"[ \\/&_=+:]", "-", name)
name = re.sub(r"[^A-Za-z0-9-]", "", name)
name = re.sub(r"-+", "-", name)
print(name.lower())
PY
}

# p() returns the stories/<canonical-name>.md path for a title.
p() { echo "stories/$(canon "$1").md"; }

# Boot the project. Single backlog at stories/.
"${AM_BIN}" create-backlog stories >/dev/null
rm -f stories/Sample-feature-set-up-login.md stories/Sample-bug-typo-on-landing.md stories/Sample-chore-rotate-api-key.md

# Seed the cast. These eight users are referenced by the assigned:
# field on every story. Create them before sync so the user pages
# resolve cleanly on the first pass.
"${AM_BIN}" create-user "Frog"    "frog@meadow.test"    >/dev/null
"${AM_BIN}" create-user "Toad"    "toad@meadow.test"    >/dev/null
"${AM_BIN}" create-user "Mouse"   "mouse@meadow.test"   >/dev/null
"${AM_BIN}" create-user "Owl"     "owl@meadow.test"     >/dev/null
"${AM_BIN}" create-user "Snail"   "snail@meadow.test"   >/dev/null
"${AM_BIN}" create-user "Sparrow" "sparrow@meadow.test" >/dev/null
"${AM_BIN}" create-user "Bear"    "bear@meadow.test"    >/dev/null
"${AM_BIN}" create-user "Lizard"  "lizard@meadow.test"  >/dev/null

# ----------------------------------------------------------------
# story_body: a short, plain-spoken problem statement keyed by title.
# Every entry is two or three sentences in the same gentle register.
# ----------------------------------------------------------------
story_body() {
  local title="$1"
  case "${title}" in
    # ---- the-button-quest (accepted) ----
    "Find a small white button")
      echo "Toad has lost a small white button from his coat."
      echo "The coat is brown. The button is somewhere, but no one"
      echo "knows where it is."
      ;;
    "Look in the bushes near the pond")
      echo "Frog said the bushes were a sensible place to look."
      echo "Toad was not convinced. Even so, they looked."
      ;;
    "Count all the buttons in a small wooden box")
      echo "There is a small wooden box of spare buttons in the attic."
      echo "The missing button might already be in there. We will count"
      echo "them to be sure."
      ;;
    "Wear the brown coat the whole afternoon")
      echo "If we wear the coat, we may notice the empty buttonhole and"
      echo "remember when it tugged at a thread. Wearing the coat all"
      echo "afternoon should help."
      ;;

    # ---- cookies-in-the-jar (accepted) ----
    "Open the cookie jar very slowly")
      echo "The cookie jar is heavy. If we open it too quickly the lid"
      echo "will clatter and Toad will rush in. Very slowly is the way."
      ;;
    "Bake a tall tray of warm cookies")
      echo "Toad has the recipe. Frog has the patience. Together they"
      echo "will bake a tall tray of warm cookies before noon."
      ;;
    "Hide the cookies under the bed for safekeeping")
      echo "If the cookies are in plain sight, they will be eaten in a"
      echo "single afternoon. Under the bed, they may last longer."
      ;;
    "Eat half the cookies anyway")
      echo "Toad found the cookies under the bed. Frog joined him."
      echo "Half of the tray is gone."
      ;;
    "Toss the last crumb to a hopeful bird")
      echo "A small bird has been watching the kitchen window all morning."
      echo "The last cookie crumb belongs to that bird now."
      ;;

    # ---- the-garden-of-patience (mixed) ----
    "Place a single seed in soft brown dirt")
      echo "It is a very small seed. The pot is very small too."
      echo "Frog says small things sometimes turn into larger things."
      ;;
    "Water the seed at exactly dawn")
      echo "The book says dawn is the best hour for watering."
      echo "Toad sets an alarm clock so he does not miss it."
      ;;
    "Speak to the seed in a soft, hopeful voice")
      echo "Toad will say polite, encouraging things to the seed."
      echo "Frog will keep the volume gentle so the seed is not startled."
      ;;
    "Hum a small tune over the seed at night")
      echo "Toad has chosen a quiet tune. He will hum it just above the"
      echo "pot for ten minutes before bed."
      ;;
    "Watch the seedling for the smallest sign of movement")
      echo "Something has appeared. It is green. Toad cannot tell if it"
      echo "is leaning toward the window or simply standing still."
      ;;
    "Build a small fence so the seedling feels safer")
      echo "A fence around the pot will keep the cat out and will make"
      echo "the seedling feel a little less exposed."
      ;;
    "Set a tiny chair beside the seedling")
      echo "Toad will sit beside the seedling for a few quiet minutes"
      echo "each evening. The chair is just big enough."
      ;;
    "Yell at the seedling to grow faster")
      echo "Toad lost his patience and yelled at the pot."
      echo "Frog has explained, gently, that this is not a method."
      ;;
    "Apologize to the seedling for yelling")
      echo "Toad would like to make amends. A small apology, said out"
      echo "loud, will do."
      ;;
    "Read a long, slow book to the seedling")
      echo "Owl has volunteered to read aloud. The book is long. The"
      echo "seedling will hear it through to the end."
      ;;
    "Replant the seedling somewhere a little warmer")
      echo "The kitchen windowsill is too cold in the afternoons."
      echo "We will move the pot to the south-facing room."
      ;;
    "Knit a tiny sweater for the seedling")
      echo "The seedling looks chilly. A tiny sweater, knit from soft"
      echo "yellow yarn, is the answer."
      ;;

    # ---- letters-by-snail (mixed) ----
    "Address the very first letter to Snail")
      echo "Snail has agreed to carry the post. The first letter must"
      echo "be addressed very carefully so it does not get lost."
      ;;
    "Wait by the mailbox for any kind of reply")
      echo "Frog stands by the mailbox. He has been standing there"
      echo "since breakfast. He is sure a letter is coming today."
      ;;
    "Pace exactly seven times up and down the path")
      echo "Toad has read that pacing seven times brings on a letter."
      echo "He paces seven times, then sits down again."
      ;;
    "Send a thank-you note by Snail")
      echo "A thank-you note is the polite response to a kind dinner."
      echo "Snail will deliver it when Snail arrives."
      ;;
    "Sort the unread letters into three quiet piles")
      echo "There are a great many unopened letters on the kitchen"
      echo "table. Mouse will sort them: friendly, official, and other."
      ;;

    # ---- the-moon-watch (all unstarted) ----
    "Plan the moons very first birthday party")
      echo "Owl has decided the moon has never had a proper birthday."
      echo "This will require a cake, some quiet music, and a careful"
      echo "list of invitations."
      ;;
    "Stand on a chair to watch the moon properly")
      echo "The kitchen window is too low. A chair is required so the"
      echo "moon can be observed at full height."
      ;;
    "Brew a small pot of tear-water tea before midnight")
      echo "Owl makes tear-water tea on quiet evenings. It tastes a"
      echo "little of sadness and a little of the moon."
      ;;

    # ---- no-epic accepted ----
    "Make a list of small things to do today")
      echo "Toad cannot remember what he meant to do. A list will help."
      echo "A list of small things."
      ;;
    "Sweep the front porch before tea")
      echo "Sparrow likes the porch to be tidy before guests arrive."
      echo "Tea is at four. The broom is in the cupboard."
      ;;
    "Knock politely on every door on Maple Lane")
      echo "Mouse is delivering invitations. Each door receives exactly"
      echo "three polite knocks, no more."
      ;;
    "Wave once at the moon through the window")
      echo "Owl wished the moon a quiet good evening. The moon did not"
      echo "wave back, but Owl felt better all the same."
      ;;

    # ---- no-epic priority unstarted ----
    "Decline the dinner party invitation politely")
      echo "Mouse cannot face a dinner party this week. A polite refusal"
      echo "must be written without hurting anyone's feelings."
      ;;

    # ---- release markers ----
    "First-frost release")
      echo "We will gather the last warm things before the first frost"
      echo "settles over the meadow."
      ;;
    "The Long-Sleep release")
      echo "Bear will sleep soon. Everything we mean to ship before"
      echo "Bear lies down must ship before this marker."
      ;;

    # ---- icebox pointed ----
    "Catch a falling leaf using only one hand")
      echo "It is said that catching a leaf in autumn grants a small"
      echo "private wish. Frog would like to try."
      ;;
    "Practice the very silent walk for a whole week")
      echo "Mouse intends to walk so quietly that even Mouse cannot"
      echo "hear Mouse walking. This will take practice."
      ;;
    "Make a tall hat from yesterdays newspaper")
      echo "A newspaper hat is a sensible thing to own. Toad has"
      echo "yesterday's paper and a serious intention."
      ;;
    "Whisper a small secret to a passing turtle")
      echo "Snail would like to share a small secret. A turtle is a"
      echo "trustworthy listener and seldom hurries off."
      ;;
    "Befriend the cranky stove in the back corner")
      echo "The stove rattles in protest each morning. Toad believes"
      echo "kindness, applied daily, will calm it."
      ;;
    "Memorize the names of every cloud shape")
      echo "Sparrow has a small book of cloud shapes. Memorizing them"
      echo "will take a season."
      ;;
    "Build a small umbrella for one worried ant")
      echo "An ant has been carrying a crumb in the rain. A very small"
      echo "umbrella would be a kind gift."
      ;;
    "Polish every doorknob on Maple Lane")
      echo "Toad has a tin of brass polish and a soft cloth. By spring"
      echo "every doorknob on the lane will shine."
      ;;
    "Carry exactly four pebbles in a pocket for luck")
      echo "Frog has read that four is the lucky number for pebbles."
      echo "Three is too few. Five is too many."
      ;;

    # ---- icebox unpointed ----
    "Apologize very gently to a very tall tree")
      echo "Toad bumped into the tree last Tuesday. He has felt sorry"
      echo "about it ever since."
      ;;
    "Hide quietly when a bumblebee comes near")
      echo "Toad finds bumblebees alarming. Hiding behind a teacup"
      echo "until the bee has passed is the agreed-upon plan."
      ;;
    "Wait for a cloud shaped like a soup ladle")
      echo "Sparrow has waited for this particular cloud all summer."
      echo "It must look exactly like a soup ladle, or it does not count."
      ;;
    "Yell down the well until it answers back")
      echo "Lizard would like to know if the well is friendly. Yelling"
      echo "is one way to find out."
      ;;
    "Count the exact number of steps to the well")
      echo "Mouse believes the count is between thirty and forty."
      echo "Mouse would like a more precise number."
      ;;
    "Pretend not to notice the neighborhood cat")
      echo "The cat must not feel watched. Mouse will look firmly at"
      echo "the wallpaper until the cat has gone."
      ;;
    "Eat exactly three crackers and then stop")
      echo "Bear has been told that three crackers is the proper number."
      echo "Bear is unsure whether Bear will be able to stop at three."
      ;;
    "Sigh in the kitchen at the wallpaper")
      echo "Owl has been waiting for the right moment to sigh at the"
      echo "wallpaper. The kitchen, in the late afternoon, will do."
      ;;
    "Stir a small pot of nothing in particular")
      echo "Toad finds the act of stirring calming. There need not be"
      echo "anything in the pot."
      ;;

    *)
      echo "A small task in the meadow: ${title}."
      ;;
  esac
}

# add_story TITLE TYPE STATUS ESTIMATE ACCEPTED EPIC OWNER EXTRA [TAGS]
add_story() {
  local title="$1"; local type="$2"; local status="$3"; local estimate="$4"
  local accepted="$5"; local epic="$6"; local owner="$7"; local extra="$8"; local tags="${9:-}"
  local path="stories/$(canon "${title}").md"

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
    echo "project: stories"
    echo "type: ${type}"
    echo "status: ${status}"
    echo "author: Fixture Author"
    echo "created: ${DATE_ITER_M6}T09:00:00Z"
    echo "modified: ${DATE_ITER_M1}T09:00:00Z"
    if [ -n "${owner}" ]; then echo "assigned: ${owner}"; fi
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
    story_body "${title}"
    echo
    echo "## Comments"
  } > "${path}"
}

# ================================================================
# DONE, 18 accepted stories, three per week, iter -6 through iter -1.
# ================================================================

# iter -6
add_story "Find a small white button"                       feature accepted "3" "${DATE_ITER_M6}" "the-button-quest"        "Toad"    "" "buttons, coat"
add_story "Make a list of small things to do today"         feature accepted "1" "${DATE_ITER_M6}" ""                         "Toad"    "" "lists"
add_story "Look in the bushes near the pond"                feature accepted "2" "${DATE_ITER_M6}" "the-button-quest"        "Frog"    "" "buttons, pond"

# iter -5
add_story "Count all the buttons in a small wooden box"     feature accepted "5" "${DATE_ITER_M5}" "the-button-quest"        "Frog"    "" "buttons, attic"
add_story "Sweep the front porch before tea"                chore   accepted ""  "${DATE_ITER_M5}" ""                         "Sparrow" "" "porch, tea"
add_story "Open the cookie jar very slowly"                 feature accepted "1" "${DATE_ITER_M5}" "cookies-in-the-jar"      "Toad"    "" "cookies"

# iter -4
add_story "Wear the brown coat the whole afternoon"         feature accepted "1" "${DATE_ITER_M4}" "the-button-quest"        "Toad"    "" "buttons, coat"
add_story "Bake a tall tray of warm cookies"                feature accepted "5" "${DATE_ITER_M4}" "cookies-in-the-jar"      "Toad"    "" "cookies, baking"
add_story "Knock politely on every door on Maple Lane"      chore   accepted ""  "${DATE_ITER_M4}" ""                         "Mouse"   "" "invitations"

# iter -3
add_story "Hide the cookies under the bed for safekeeping"  feature accepted "3" "${DATE_ITER_M3}" "cookies-in-the-jar"      "Toad"    "" "cookies"
add_story "Eat half the cookies anyway"                     feature accepted "2" "${DATE_ITER_M3}" "cookies-in-the-jar"      "Frog"    "" "cookies"
add_story "Place a single seed in soft brown dirt"          feature accepted "2" "${DATE_ITER_M3}" "the-garden-of-patience"  "Frog"    "" "garden, seedling"

# iter -2 (strength 0.5)
add_story "Water the seed at exactly dawn"                  feature accepted "3" "${DATE_ITER_M2}" "the-garden-of-patience"  "Toad"    "" "garden, seedling"
add_story "Speak to the seed in a soft, hopeful voice"      feature accepted "1" "${DATE_ITER_M2}" "the-garden-of-patience"  "Toad"    "" "garden, seedling"
add_story "Toss the last crumb to a hopeful bird"           feature accepted "1" "${DATE_ITER_M2}" "cookies-in-the-jar"      "Frog"    "" "cookies, birds"

# iter -1
add_story "Hum a small tune over the seed at night"         feature accepted "2" "${DATE_ITER_M1}" "the-garden-of-patience"  "Toad"    "" "garden, seedling"
add_story "Wave once at the moon through the window"        feature accepted "1" "${DATE_ITER_M1}" ""                         "Owl"     "" "moon"
add_story "Address the very first letter to Snail"          feature accepted "3" "${DATE_ITER_M1}" "letters-by-snail"        "Frog"    "" "letters"

# ================================================================
# PRIORITY, 18 stories in current and upcoming iterations.
# ================================================================

# Current iteration: 4 in-flight stories + 3 accepted this week.
# The accepted-this-week trio shows up in Backlog with the green
# chip and no action buttons; they only graduate to Done at the
# next Monday tick.
add_story "Watch the seedling for the smallest sign of movement" feature started   "5" ""                "the-garden-of-patience" "Toad"  "" "garden, seedling"
add_story "Wait by the mailbox for any kind of reply"             feature started   "3" ""                "letters-by-snail"       "Frog"  "" "letters, patience"
add_story "Pace exactly seven times up and down the path"         feature started   "2" ""                "letters-by-snail"       "Toad"  "" "letters, patience"
add_story "Sort the unread letters into three quiet piles"        feature delivered "3" ""                "letters-by-snail"       "Mouse" "" "letters"
add_story "Build a small fence so the seedling feels safer"       feature accepted  "3" "${DATE_ITER_0}"  "the-garden-of-patience" "Frog"  "" "garden, seedling"
add_story "Set a tiny chair beside the seedling"                  feature accepted  "1" "${DATE_ITER_0}"  "the-garden-of-patience" "Toad"  "" "garden, seedling"
add_story "Send a thank-you note by Snail"                        feature accepted  "2" "${DATE_ITER_0}"  "letters-by-snail"       "Mouse" "" "letters, manners"

# Rejected (1).
add_story "Yell at the seedling to grow faster"                   feature rejected  "3" "" "the-garden-of-patience" "Toad"  "" "garden, regret"

# Planned next, pointed unstarted (8).
add_story "Apologize to the seedling for yelling"                 feature unstarted "1" "" "the-garden-of-patience" "Toad"  "" "garden, manners"
add_story "Decline the dinner party invitation politely"          feature unstarted "2" "" ""                       "Mouse" "" "manners"
add_story "Replant the seedling somewhere a little warmer"        feature unstarted "3" "" "the-garden-of-patience" "Frog"  "" "garden, seedling"
add_story "Knit a tiny sweater for the seedling"                  feature unstarted "5" "" "the-garden-of-patience" "Toad"  "" "garden, knitting"
add_story "Read a long, slow book to the seedling"                feature unstarted "5" "" "the-garden-of-patience" "Owl"   "" "garden, books"
add_story "Brew a small pot of tear-water tea before midnight"    feature unstarted ""  "" "the-moon-watch"         "Owl"   "" "moon, tea"
add_story "Stand on a chair to watch the moon properly"           feature unstarted "3" "" "the-moon-watch"         "Owl"   "" "moon"
add_story "Plan the moons very first birthday party"              feature unstarted "8" "" "the-moon-watch"         "Owl"   "" "moon, party"

# Release markers (2). Type=release; release_date drives the marker.
add_story "First-frost release"                                   release unstarted ""  "" "" "" "release_date: ${DATE_FROST}"
add_story "The Long-Sleep release"                                release unstarted ""  "" "" "" "release_date: ${DATE_LONGSLEEP}"

# ================================================================
# ICEBOX, 18 unstarted stories. 9 pointed, 9 unpointed.
# ================================================================

# Pointed (9).
add_story "Catch a falling leaf using only one hand"              feature unstarted "1" "" "" "Frog"    "" "luck, leaves"
add_story "Practice the very silent walk for a whole week"        feature unstarted "2" "" "" "Mouse"   "" "practice"
add_story "Make a tall hat from yesterdays newspaper"             feature unstarted "1" "" "" "Toad"    "" "hats"
add_story "Whisper a small secret to a passing turtle"            feature unstarted "3" "" "" "Snail"   "" "secrets"
add_story "Befriend the cranky stove in the back corner"          feature unstarted "5" "" "" "Toad"    "" "kitchen, kindness"
add_story "Memorize the names of every cloud shape"               feature unstarted "5" "" "" "Sparrow" "" "clouds, study"
add_story "Build a small umbrella for one worried ant"            feature unstarted "2" "" "" "Mouse"   "" "ants, weather"
add_story "Polish every doorknob on Maple Lane"                   feature unstarted "3" "" "" "Toad"    "" "shine, lane"
add_story "Carry exactly four pebbles in a pocket for luck"       feature unstarted "1" "" "" "Frog"    "" "luck, pebbles"

# Unpointed (9).
add_story "Apologize very gently to a very tall tree"             feature unstarted ""  "" "" "Toad"    "" "manners, trees"
add_story "Hide quietly when a bumblebee comes near"              feature unstarted ""  "" "" "Toad"    "" "bees"
add_story "Wait for a cloud shaped like a soup ladle"             feature unstarted ""  "" "" "Sparrow" "" "clouds, soup"
add_story "Yell down the well until it answers back"              feature unstarted ""  "" "" "Lizard"  "" "echoes, wells"
add_story "Count the exact number of steps to the well"           feature unstarted ""  "" "" "Mouse"   "" "wells, counting"
add_story "Pretend not to notice the neighborhood cat"            feature unstarted ""  "" "" "Mouse"   "" "cats, wallpaper"
add_story "Eat exactly three crackers and then stop"              feature unstarted ""  "" "" "Bear"    "" "crackers, restraint"
add_story "Sigh in the kitchen at the wallpaper"                  feature unstarted ""  "" "" "Owl"     "" "sighing"
add_story "Stir a small pot of nothing in particular"             feature unstarted ""  "" "" "Toad"    "" "stirring, calm"

# ================================================================
# Project docs (.am/).
# ================================================================
mkdir -p .am

cat > .am/iterations.yaml <<EOF
overrides:
  - number: ${ITER_NUM_M2}
    team_strength: 0.5
EOF

cat > .am/inception.md <<'INCEPTION'
# Inception

## The users

Frog, Toad, and their small circle of friends in the meadow.

## The goal

To keep a careful list of the small things that must be done, and to
feel some sense of order in a confusing season.

## The reason

Toad cannot keep things straight in his head. Frog is patient but
also forgetful. A list helps. A list helps a great deal.

## Success

When a small thing is on the list and that small thing has been done,
the list says so. Everyone agrees that this is enough.

## Constraints

- The list must not be longer than the kitchen table.
- Snail delivers the post when Snail delivers the post.
- The seedling will grow at the seedling's chosen pace.

## Out of scope

- Hurrying.
- Anything that would upset Bear before the Long-Sleep.
INCEPTION

cat > .am/team-agreements.md <<'AGREEMENTS'
# Team agreements

- When a story makes Toad nervous, Frog will sit with him for a while.
- No story is too small to put on the list.
- If the seedling has not grown, this is not anyone's fault.
- Yelling is not a method.
- Snail delivers the post when Snail delivers the post.
- Bear should not be disturbed in the afternoons.
AGREEMENTS

cat > .am/learnings.md <<LEARNINGS
# Learnings

- On the morning of ${DATE_ITER_M3}: yelling at the seedling did not help. We have agreed to try kindness instead.
- On the afternoon of ${DATE_ITER_M2}: the cookies were better when shared. We will remember this.
- On the evening of ${DATE_ITER_M1}: Owl is not as alone as Owl seems. We will write to Owl more often.
LEARNINGS

# ================================================================
# Sync. EnforceOrderInvariant places started/finished/delivered/
# accepted/rejected in priority and unstarted in icebox.
# ================================================================
"${AM_BIN}" sync </dev/null >/dev/null

# ================================================================
# Rank the priority list so the current iteration reads top-down.
# Order: in-flight → rejected → planned unstarted → release markers.
# Accepted stories sit untouched below; the iteration-band view in
# `am show priority` slots them into their original iteration weeks.
# ================================================================
rank_after() { "${AM_BIN}" rank "$(p "$1")" --after "$(p "$2")" >/dev/null; }
"${AM_BIN}" rank "$(p "Wait by the mailbox for any kind of reply")" --top >/dev/null
rank_after "Pace exactly seven times up and down the path"         "Wait by the mailbox for any kind of reply"
rank_after "Watch the seedling for the smallest sign of movement"  "Pace exactly seven times up and down the path"
rank_after "Build a small fence so the seedling feels safer"       "Watch the seedling for the smallest sign of movement"
rank_after "Set a tiny chair beside the seedling"                  "Build a small fence so the seedling feels safer"
rank_after "Send a thank-you note by Snail"                        "Set a tiny chair beside the seedling"
rank_after "Sort the unread letters into three quiet piles"        "Send a thank-you note by Snail"
rank_after "Yell at the seedling to grow faster"                   "Sort the unread letters into three quiet piles"
rank_after "Apologize to the seedling for yelling"                 "Yell at the seedling to grow faster"
rank_after "Decline the dinner party invitation politely"          "Apologize to the seedling for yelling"
# Outdoor / garden-cleanup work above First-frost. Anything that
# should be wrapped up before the cold sets in lands here: the
# seedling moves indoors, last sweater is knit, the long slow book
# is read while there's still light on the porch.
rank_after "Replant the seedling somewhere a little warmer"        "Decline the dinner party invitation politely"
rank_after "Knit a tiny sweater for the seedling"                  "Replant the seedling somewhere a little warmer"
rank_after "Read a long, slow book to the seedling"                "Knit a tiny sweater for the seedling"
rank_after "First-frost release"                                   "Read a long, slow book to the seedling"
# Indoor / moon-watch work below First-frost but above Long-Sleep.
# Tea brewed at midnight, chair pulled to the kitchen window, the
# moon's birthday planned, all activities for the dark months
# leading up to Bear's Long-Sleep.
rank_after "Brew a small pot of tear-water tea before midnight"    "First-frost release"
rank_after "Stand on a chair to watch the moon properly"           "Brew a small pot of tear-water tea before midnight"
rank_after "Plan the moons very first birthday party"              "Stand on a chair to watch the moon properly"
rank_after "The Long-Sleep release"                                "Plan the moons very first birthday party"

# ================================================================
# Comments on selected stories, in voice.
# ================================================================
add_comment() { "${AM_BIN}" comment "$(p "$1")" "$2" --author "$3" >/dev/null; }

add_comment "Watch the seedling for the smallest sign of movement" "Nothing has happened yet. The pot looks exactly as it did yesterday." "toad"
add_comment "Watch the seedling for the smallest sign of movement" "Things are happening underground, Toad. Trust the seed."               "frog"
add_comment "Watch the seedling for the smallest sign of movement" "I have placed the chair a little closer, just in case."                "toad"
add_comment "Wait by the mailbox for any kind of reply"             "A letter must come today. I have a feeling."                           "frog"
add_comment "Wait by the mailbox for any kind of reply"             "I am walking as fast as I can."                                        "snail"
add_comment "Yell at the seedling to grow faster"                   "A seedling does not respond to volume. Please try kindness next."      "frog"
add_comment "Plan the moons very first birthday party"              "The moon does not know I am planning this. That is part of the gift." "owl"
add_comment "Sort the unread letters into three quiet piles"        "Three piles: friendly, official, and other. The Other pile is large." "mouse"
add_comment "Brew a small pot of tear-water tea before midnight"    "Sad things make the tea taste right. Quiet things help too."          "owl"
add_comment "Apologize to the seedling for yelling"                 "I will say it out loud, even if no one is listening."                 "toad"

# Reject the rejected story with a real reason so the rejection note
# section is populated.
"${AM_BIN}" reject "$(p "Yell at the seedling to grow faster")" --reason "Yelling is not a method. Please try kindness instead." >/dev/null

# ================================================================
# Tasks on selected stories.
# ================================================================
add_task() { "${AM_BIN}" task add "$(p "$1")" "$2" >/dev/null; }
tick_task() { "${AM_BIN}" task tick "$(p "$1")" "$2" >/dev/null; }

add_task "Plan the moons very first birthday party" "Draft a small list of invitations"
add_task "Plan the moons very first birthday party" "Find a sensibly sized cake"
add_task "Plan the moons very first birthday party" "Choose a quiet hour after sundown"
add_task "Plan the moons very first birthday party" "Compose one short, suitable song"
tick_task "Plan the moons very first birthday party" 1

add_task "Knit a tiny sweater for the seedling" "Find soft yellow yarn"
add_task "Knit a tiny sweater for the seedling" "Measure the seedling carefully"
add_task "Knit a tiny sweater for the seedling" "Buy three very small buttons"

add_task "Replant the seedling somewhere a little warmer" "Check the south-room soil temperature"
add_task "Replant the seedling somewhere a little warmer" "Find a pot with no chips"
add_task "Replant the seedling somewhere a little warmer" "Carry the seedling gently across the kitchen"

# ================================================================
# History on one story so the per-item History panel has a trail.
# ================================================================
BUTTON_PATH="$(p "Find a small white button")"
{ git add "${BUTTON_PATH}" >/dev/null 2>&1 || true; }
{ echo "" >> "${BUTTON_PATH}"; git -c user.email=fixture@example.com -c user.name="Fixture Author" commit --quiet -am "Find-a-small-white-button: clarified that the coat is brown" >/dev/null 2>&1 || true; }
{ echo "" >> "${BUTTON_PATH}"; git -c user.email=fixture@example.com -c user.name="Fixture Author" commit --quiet -am "Find-a-small-white-button: added a hopeful note from Frog" >/dev/null 2>&1 || true; }

echo "fixture built at: ${TARGET}"
echo "stories:          54   (18 accepted + 18 priority + 18 icebox)"
echo "epics:            the-button-quest (done), cookies-in-the-jar (done),"
echo "                  the-garden-of-patience (in flight), letters-by-snail (in flight),"
echo "                  the-moon-watch (not started)"
echo "iter -6 .. -1:    ${DATE_ITER_M6} .. ${DATE_ITER_M1}"
echo "iter -2 number:   ${ITER_NUM_M2}  (strength override 0.5)"
echo "release dates:    first-frost ${DATE_FROST}, long-sleep ${DATE_LONGSLEEP}"
