---
name: am-align
description: >
  Run the pre-pull alignment for an agilemarkdown story. Read the story,
  restate the agent's understanding in one paragraph, surface any
  ambiguous acceptance bullet, and ask the PM for clarification before
  writing code. Use when the user says "let's pull X", "start X", "I
  want to work on X", "/am-align", or when the agent is about to start
  coding on a feature.
---

# am-align

The agent is the dev pair. Before the dev pair writes code on a story,
the dev pair restates the story in its own words and asks the PM to
confirm. With humans, this was pair programming. With an AI in the
dev-pair seat, this is the safeguard against the central failure mode
of agent coding: the agent that confidently builds the wrong feature.

## When this fires

- The user says any of: "let's pull X", "start X", "I want to work on
  X", "what's next", or invokes `/am-align`.
- The agent is about to call `set_status(path, "started")` on a feature.
  Fire this skill first.

## What to do

1. Identify the story path. Use `next_item` if no path was given. Insist
   on exactly one path.

2. Call `get_item(path)` for the frontmatter and body. Call
   `list_acceptance(path)` for the parsed bullets (or read the
   `acceptance` field on `get_item`'s response).

3. Read `inception.md` if it exists. The story should fit the project's
   stated user, goal, and constraints; if the story does not, surface
   the conflict here, not after code is written.

4. Run `coach_check(action="pull", path=<path>)`. If the verdict is
   refused, stop and surface the rule + next move: there is no
   acceptance section in the body, and the agent should not write code
   against an empty target.

5. Restate the agent's reading of the story in one paragraph. Cover:
   - Who the user is (one line).
   - What the change is (one or two lines).
   - What done looks like, in terms of the acceptance bullets.

6. List the acceptance bullets verbatim with their indices and state
   markers (`[ ]` / `[~]` / `[x]`).

7. Surface checkable ambiguities. Look for:
   - Fewer than two bullets.
   - A bullet that mentions a metric without a value ("fast", "many",
     "most").
   - A bullet that references another story not in the priority list.
   - An estimate missing, or above 8.

8. Ask the PM: "anything to clarify before I pull?"

9. If clarifications come back, use the right writer:
   - New criterion or refined criterion: `append_acceptance_bullet` or
     `set_description` for a body edit.
   - Different estimate: `set_estimate`.
   - Different priority: `rank_item`.
   - Wrong story altogether: leave it, return to step 1 with a
     different path.

10. Once the PM is satisfied, call `set_status(path, "started")`. The
    server allows this transition; the dev pair is now actively working
    on the story.

## What NOT to do

- Do not write code before step 10. Alignment is the seam.
- Do not paraphrase the acceptance bullets in step 6. Verbatim. The PM
  needs to see them as they will be checked at acceptance time.
- Do not invent new acceptance bullets in the restatement. If a bullet
  is missing, ask the PM whether to add it; if yes, append explicitly.
- Do not skip this skill on bugs or chores. They have their own
  conventions; restate the story in one line so the PM can confirm the
  agent has the right context, but do not block on bullets.

## In solo mode

The same human is both the dev's pair and the PM. The restatement is
still useful: written down, it forces the human to notice when their
own one-line story title is hiding a hard decision. The pause is the
point.
