---
name: am-inception
description: >
  Walk the human through running an inception for a new agilemarkdown
  project, capturing the six sections (user, goal, reason, success,
  constraints, out of scope) into inception.md. Use when the user says
  "let's run an inception", "kick off the project", "/am-inception",
  or starts a new agilemarkdown repo without an inception.md.
---

# am-inception

An inception is the conversation that frames a project. The output is
a one-page document at `inception.md` covering six sections. This skill
runs the conversation; the human answers; the doc lands on disk.

## When this fires

- Human says "let's run an inception" or starts a new project.
- Repo has no `inception.md` and the human is about to plan stories.
- User invokes `/am-inception`.

## What to do

1. Check whether `inception.md` already exists. Call
   `inception_doc()` (no body argument). If existed=true, ask the
   human whether to read the current doc or rewrite it. Default:
   read.

2. If the doc does not exist or the human wants to rewrite, walk
   the six questions one at a time. Ask each, wait, write the
   answer down. Do not move on until the human answers.

   - **The user.** "Who, specifically, are we building this for?
     Not a demographic. A real person with real circumstances."

   - **The goal.** "What changes for that user when we ship? Make
     it observable."

   - **The reason.** "Why this, why now? What changes if we do not
     ship this?"

   - **Success.** "How will we know it worked? The smallest signal
     you would believe."

   - **Constraints.** "What can't move? Budget, deadline, regulator,
     dependency."

   - **Out of scope.** "What are we explicitly not doing? Naming
     this saves arguments later."

3. Assemble the answers into the inception template shape (six
   `##` headings with the human's text under each) and write the
   doc with `inception_doc(body=<assembled body>)`.

4. Confirm: print the doc back to the human and ask whether to
   commit it (the human commits via git; you do not run git from
   the agent).

## What NOT to do

- Do not invent answers. If the human says "I don't know," that is
  the answer; record it and move on. The inception is more useful
  with honest gaps than with confident fabrications.
- Do not write stories yet. The inception produces a one-pager.
  Stories come later, once the team is ready to plan the first
  iteration.
- Do not skip the "Out of scope" section. It is the most useful
  one in retrospect. If the human says "nothing", push: "really,
  nothing? what are we deliberately not doing?"
