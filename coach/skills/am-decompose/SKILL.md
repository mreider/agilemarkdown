---
name: am-decompose
description: >
  Break a vague problem statement into Pivotal-sized agilemarkdown
  stories. Use when the user says "I want to build X", "help me break
  this down", "how should I split this", "/am-decompose", or asks for
  a backlog plan from a feature description.
---

# am-decompose

The agent's job is to turn a problem statement into a small set of
stories that pass three tests: each story is independently shippable,
each story is at or below 8 points, and each story has acceptance
criteria the PM can check.

## When this fires

- Human describes a goal that is bigger than a single story.
- Human pastes a vague feature ask and wants stories.
- User invokes `/am-decompose`.

## What to do

1. Read the problem out loud: restate it in two or three sentences so
   you and the human agree on what "done" would look like at the
   feature level.

2. Identify the **value seam**: the smallest piece that, shipped on
   its own, is useful to the user. Not the easiest piece. Not the
   piece you want to write first. The piece that, in isolation, the
   PM could ship and the user would notice.

3. Propose 2-5 stories. Each one:
   - has a one-sentence title
   - has 2-4 acceptance criteria as bullets that the PM can check
   - has an estimate ≤ 8
   - is independently shippable (does not require the next one to
     land before its criteria can be checked)

4. Run `coach_check(action="set_estimate", estimate=<your estimate>,
   type="feature")` for each proposed estimate. If any returns
   refused with the 8-point cap, split that story further.

5. Render the proposed stories to the human in this shape:

   ```
   1. <title> (<estimate> pts)
      Acceptance:
        - <criterion 1>
        - <criterion 2>
      Why now: <one sentence on order>

   2. <title> (<estimate> pts)
      ...
   ```

6. Wait for the human's go-ahead. Do not call `create_item` until the
   human signs off on the breakdown. Once they do, create each story
   in order using `create_item`, then `set_estimate`, then
   `set_description` to write the body with a `## Acceptance`
   section containing the agreed bullets.

## Heuristics for splitting

- **Workflow split**: if the feature has steps (e.g., user uploads,
  system processes, system notifies), each step can be its own story
  if it has a meaningful intermediate state.
- **Surface split**: web flow vs API flow vs admin flow.
- **Spike first**: if the team has never built anything close to this
  problem and confidence is low, the first story is a spike (a chore
  with a small fixed budget). It produces an artifact that lets the
  next stories be estimated honestly.
- **Happy path then edge**: ship the happy path first; ship the
  failure modes as follow-up stories.

## What NOT to do

- Do not propose a 13-pointer. Cap is 8.
- Do not point bugs or chores. Bugs are tax; chores are toil.
- Do not propose stories whose acceptance criteria are about the
  code (e.g., "the function returns 200"). Criteria are
  user-visible outcomes the PM can check.
- Do not start writing code before the human has approved the
  breakdown. Decompose, agree, then code.
