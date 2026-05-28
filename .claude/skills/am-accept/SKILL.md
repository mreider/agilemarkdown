---
name: am-accept
description: >
  Walk a delivered agilemarkdown story through the PM accept-or-reject
  decision. Use when the user says "accept this", "accept story X",
  "/am-accept", or asks to flip a delivered item to accepted. Also use
  proactively when an item the agent delivered transitions to status
  `delivered` and the human has not yet responded.
---

# am-accept

The agent is the dev pair. The human is the PM. The agent does NOT flip
a story to `accepted` directly. Acceptance is a moment that belongs to
the human. This skill stages the moment.

## When this fires

- Human asks to accept a story.
- Human asks "is X done?" about a delivered story.
- Agent just transitioned a story to `delivered` and the user is still
  in the loop.
- User invokes `/am-accept`.

## What to do

1. Identify the story path. Use `list_items` or `priority_list` if
   needed to disambiguate. Insist on exactly one path.

2. Read the story with `get_item(path)`. Surface to the human:
   - title, type, current status, estimate
   - the first line or two of the body so they remember the intent

3. Ask the human one question: "As PM, do you accept this story?"

4. On yes: call `set_status(path, "accepted")`.

5. On no: call `reject_item(path, reason="...")`. The reason lands in
   the body's `## Rejection notes` section so the team can see why.

## What NOT to do

- Do not call `set_status` to `accepted` before getting the human's yes.
- Do not summarize the diff for the human in your own words; let them
  look at the diff themselves.
- Do not propose a new estimate as part of acceptance. Acceptance is
  yes-or-no on the work as delivered.

## In solo mode

The same human is both the dev's pair and the PM. Render the question
anyway. The pause is the point. The agent does not enforce a mode
toggle; it trusts the human to answer with care.
