---
name: am-accept
description: >
  Render the PM acceptance ceremony for a delivered agilemarkdown story.
  Use when the user says "accept this", "accept story X", "/am-accept",
  or asks to flip a delivered item to accepted. Also use proactively
  when an item the agent delivered transitions to status `delivered` and
  the human has not yet responded.
---

# am-accept

The agent is the dev pair. The human is the PM. The agent does NOT flip
a story to `accepted` directly. Acceptance is a moment that belongs to
the human. This skill runs the ceremony that makes the moment real.

## When this fires

- Human asks to accept a story.
- Human asks "is X done?" about a delivered story.
- Agent just transitioned a story to `delivered` and the user is still
  in the loop.
- User invokes `/am-accept`.

## What to do

1. Identify the story path. Use `list_items` or `priority_list` if
   needed to disambiguate. Insist on exactly one path.

2. Call the MCP tool `acceptance_prompt(path=<path>)`. It returns:
   - title, type, current status
   - estimate
   - `bullets`: each acceptance bullet with index, state, text, and an
     optional claim note
   - `verify`: the same bullet text as a plain list (for back-compat)
   - prompt_text: the rendered ceremony with `[ ]` / `[~]` / `[x]`
     markers per bullet

3. Render `prompt_text` to the human verbatim. Do not paraphrase. The
   ritual depends on the human seeing the same shape every time.

4. Walk the bullets one at a time. For each bullet:

   - If the bullet is `[x]` verified already, skip it.
   - If the bullet is `[~]` claimed by the dev pair, ask the human a
     yes/no question: "bullet N: <text> — does this pass?"
   - If the bullet is `[ ]` open, the dev pair never claimed it. Point
     it out: "bullet N is still open. Should I mark it verified, leave
     it open, or reject?"

   For a "yes" on a bullet, call
   `set_acceptance_state(path, index=N, state="verified")`.

   For a "no" on a bullet, stop the walk and call
   `reject_item(path, reason="...", failing_bullet=N)`. The bullet
   reopens to `[ ]` and the rejection note cites it.

5. Once every bullet is verified, call `set_status(path, "accepted")`.
   The server allows the transition once the ceremony has rendered.

6. After the transition, briefly confirm: "Accepted. Hypothesis was X.
   Velocity ledger reflects N points." Or after rejection: "Sent back
   with reason: X, bullet N reopened."

## What NOT to do

- Do not call `set_status` to `accepted` before rendering the prompt
  and getting the human's yes.
- Do not summarize the diff for the human in your own words; let
  them look at the diff themselves. The prompt's "What to verify"
  bullets are the contract.
- Do not propose a new estimate or new acceptance criteria as part of
  acceptance. Acceptance is yes-or-no on the work as delivered.

## In solo mode

The same human is both the dev's pair and the PM. Render the prompt
anyway. The pause is the point. The agent does not enforce a mode
toggle; it trusts the human to answer the question with care.
