---
name: am-retro
description: >
  Run a Pivotal-style end-of-iteration retro on an agilemarkdown repo.
  Use when the user says "retro", "let's do a retro", "wrap the
  iteration", "/am-retro", or at the natural cadence boundary of an
  iteration in a project the user is actively working on.
---

# am-retro

A retro is a moment to ask three questions: what worked, what didn't,
what changes do we make. Pivotal teams ran them at the iteration
boundary. This skill walks the human through one, using the data
already in the repo.

## When this fires

- Iteration boundary (the last accepted story for the iteration just
  landed, or the human says "wrap up").
- Human asks for a retro.
- User invokes `/am-retro`.

## What to do

1. Pull the iteration's data via MCP tools:
   - `dashboard()` for velocity, volatility, cycle time median,
     latest rejection rate.
   - `velocity_history(backlog=<name>)` for recent iteration deltas.
   - `type_mix()` for feature/bug/chore split.
   - Read `learnings.md` if present (via the file system; it is not a
     story file, so no MCP wrapper).
   - Read `team-agreements.md` if present.

2. Render a one-screen summary in this shape:

   ```
   Iteration X retro

   Numbers:
     velocity:       <n> pts (was <prev>)
     volatility:     <n>%
     cycle time:     <n>d (median)
     rejection rate: <n>% (target band 5-15%)
     type mix:       feature <n>%, bug <n>%, chore <n>%

   What landed:
     - <story> (<type>, <pts>)
     ...

   What got rejected (if any):
     - <story> — <reason>
     ...

   Carryover:
     <items still in started/finished/delivered that did not accept>
   ```

3. Ask the human three questions, one at a time, waiting between
   each:

   - **What worked?** Give them a moment. Then offer to record their
     answer to `learnings.md` via the `record_learning` tool.

   - **What did not work?** Same shape. The signal here is the one
     that informs the next agreement or canon-tightening.

   - **What changes for next iteration?** This is where new working
     agreements come from. If the human names a behavior, a moment,
     and a default, propose adding it to `team-agreements.md` via
     the `team_agreements` tool with `set:` and the appended line.

4. If rejection rate is outside the 5-15% target band, surface that
   separately. Below 5% means the PM is not being careful enough;
   above 15% means the team is shipping work the PM does not want.
   Ask the human what they think is driving it.

5. If volatility is high (over ~40%), surface that as a forecasting
   risk. Iterations are not predictable. Either the team is taking
   on stories whose size they cannot estimate, or the team's
   composition is shifting. Ask which.

6. Close with a one-line summary: what was learned, what changed,
   what to watch next iteration.

## What NOT to do

- Do not turn the retro into a status update. The numbers are a
  starting point; the human's three answers are the point.
- Do not propose new stories from the retro. New stories come from
  user value, not retro outputs. Retro outputs go into agreements
  or learnings.
- Do not promote a working agreement to canon. Canon is the Pivotal
  set; agreements are project-local nudges.
