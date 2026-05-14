---
name: am-plan
description: >
  Run the iteration planning meeting (IPM) for an agilemarkdown
  backlog. Render the top of priority up to rolling velocity, flag
  stories that need attention, walk the human through the commit.
  Use when the user says "let's plan the iteration", "run IPM",
  "/am-plan", or at the start of a new iteration.
---

# am-plan

The IPM is the short conversation at the start of each iteration where
the team commits to the work. The commit is implicit: it is the rank
order of `_priority.md` at the moment the iteration starts.

## When this fires

- Human says "let's plan", "run IPM", "what's the iteration".
- New iteration is starting.
- User invokes `/am-plan`.

## What to do

1. Identify the backlog. If multiple, ask which. Most projects
   have one.

2. Call `sprint_plan(backlog=<name>)`. The tool returns the rank
   order, divided into committed (top of priority up to rolling
   velocity) and below-line (the rest), plus warnings on stories
   that need attention before the iteration starts.

3. Render the result to the human in a clear shape:

   ```
   Iteration plan: <backlog>   velocity <N> / iteration

   Committed:
     1. ★  <title>            <type>  <pts>p   acceptance: <count> bullets
     2. ●  <title>            <type>  <pts>p   acceptance: <count>
     ...

   Below the line:
     <next 3-5 stories>

   Committed: <total> pts. Velocity: <N>. <Fits | Over by N pts>.
   ```

4. Surface warnings explicitly. The common ones:
   - **Missing acceptance criteria**: a feature in the committed
     set has no `## Acceptance` section. Offer to draft criteria
     into the body via `set_description`.
   - **Oversized feature**: a feature has an estimate above 8.
     Offer to call the `am-decompose` skill to split it.
   - **Unestimated feature**: a feature has no estimate. Ask the
     human for a Fibonacci value (1, 2, 3, 5, 8). Pivotal pattern:
     point by uncertainty, not by hours.
   - **Overcommit**: committed points exceed rolling velocity.
     Offer to trim from the bottom of the committed set.

5. Wait for the human's response. Loop on edits: the human edits
   `_priority.md` (re-rank) or asks you to fix a story; you re-call
   `sprint_plan`; you re-render. The loop ends when the human is
   satisfied with the plan as rendered.

6. The commit moment is implicit. The human says "let's go," or
   the human starts pulling the first story. Do not write a
   separate commit artifact.

## What NOT to do

- Do not invent estimates. If a story has no estimate, ask the
  human for one. Do not set one and move on.
- Do not propose splitting a story without offering to call
  `am-decompose`. The decompose skill is the right tool; use it.
- Do not pull a story (set_status started) during planning. The
  planning ceremony is a review; the iteration starts when the
  human pulls the first story afterwards.
