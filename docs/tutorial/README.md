# agilemarkdown tutorial

Every chapter shows the same thing three ways: the board (GUI), the
`am` command-line, and the MCP tool an AI agent would call. By the
end you'll know which surface to reach for, and you'll have seen the
full mapping between them.

## The Pivotal week

The workflow this tutorial demonstrates comes from Pivotal Tracker,
and before that from Extreme Programming. The shape is small: one
ranked backlog, weekly sprints, three kinds of work, a few rituals
to hold it together. The chapters demonstrate the mechanics; this
page sketches the week so the mechanics make sense.

**One ranked backlog.** Different tools mean different things by
"backlog." Here it is a single list of stories, ranked top to bottom.
The top of the list is the current sprint, the work we expect to
finish this week. A sprint is a week, not two, and it is not a
contract for a fixed set of stories. Unplanned work arrives every
week and the team finishes what it finishes. As people free up, they
pull the next story off the top.

**Velocity is a forecast.** Each story carries an estimate in points,
a rough sizing the team agrees on. agilemarkdown uses those points
to place markers down the backlog showing which stories are likely
to land this week, the next week, or later. The math accounts for
team strength: who is sick, who is on vacation. Only estimated
stories get markers, and the deeper into the backlog you look, the
looser the projection. That blurriness further out is just what
software actually feels like.

> "Velocity is the average normalized points completed across the
> last N iterations, where N is the project's Velocity Strategy
> setting (default 3, range 1-4). Velocity is computed once per
> iteration boundary and shown as an integer (it is floored on
> display and never rounded up)."
>
> Pivotal Tracker help center, [understanding velocity](https://web.archive.org/web/20170503214450/https://www.pivotaltracker.com/help/articles/understanding_velocity/)

**Three kinds of work.** Features are stories: a unit of customer
value, carrying a point estimate and an acceptance check. The dev
pair pulls the story off the top and works it; the product manager
accepts it once delivered. The dev pair never accepts its own work,
and the coach refuses any attempt. Bugs go to the top of the list
and carry no estimate, so a buggy stretch shows up automatically as
a drop in velocity. Chores are work the team owes itself, CI setup,
doc fixes after a customer complains, research the team needs, and
also carry no estimate since they do not ship to a customer. Release
markers are a fourth type but are not work; they are dated stakes in
the priority list (Chapter 7).

**The icebox.** A backlog needs a place for ideas that are not ready
to prioritize. The icebox is that place: a capture pile that is not
ranked and is not being worked on. Engineers drop in features they
think would be cool, the PM parks ideas before deciding, customer
feedback lands there as it comes in. At planning time, items get
pulled out, talked through, estimated if they are nearly ready, and
ranked into the backlog at whatever position the PM decides.

**The weekly rhythm.**

- **Monday plan.** The team walks the top of the backlog together
  so the week is clear, estimates anything new with a quick
  fingers-up game called pointing poker, and slides out any story
  the PM cannot answer clearly. The aim is to leave ready to work,
  not stuck on questions.
- **Daily standup.** A short go-round in which each person offers
  either an *interesting* (something they learned, or something
  another team is doing) or a *help* (I am stuck on X, who can
  pair). Helps often resolve in the meeting itself, the PM can pass
  without speaking, and a standup with nothing to report is a
  perfectly good thing.
- **Story time.** On most Friday afternoons the PM and the
  engineering lead meet to prep for Monday, pulling icebox items
  out and estimating them so they can be ranked into priority.
- **Friday retro.** At the end of the week the team takes a breath
  and puts three columns up on a board (*good*, *meh*, *not so
  good*); people write topics and discuss them civilly, without
  blame, and the action items that come out of the conversation
  are the first thing the next retro checks on.

The 8-point cap holds across all of this. Any story bigger than 8
points is an epic and gets split before it can move. agilemarkdown's
coach refuses to estimate a feature above the cap; Chapter 6 covers
epics directly.

## Following along

The tutorial uses two project states. Chapters 1–2 start from an
empty repo so you can see how an agilemarkdown project comes into
being. Chapters 3 onward use a seeded fixture in the spirit of
Arnold Lobel's *Frog and Toad*: Toad mails letters, Frog tends a
seedling, Owl plans the moon's birthday. The whimsy is the
example, not the tool. (Lobel, *Frog and Toad Are Friends*,
Harper & Row, 1970.)

If you want to follow along interactively, run the laboratory:

```
am serve --root /tmp/am-demo --fixture scripts/build-demo.sh
open http://localhost:8080
```

That gives you the same web UI, CLI, and MCP console the tutorial
captures from. Every action you take maps to a CLI verb in the
**CommandLog** tab and a JSON-RPC frame in the **MCP Console** tab.

## Chapters

1. [Inception](01-inception/), write the problem statement before the first story.
2. [Your first story](02-first-story/), create, estimate, rank, start.
3. [The seeded backlog](03-meadow-intro/), switching to the loaded fixture.
4. [The current sprint](04-current-sprint/), start, finish, deliver, accept.
5. [When things go wrong](05-when-things-go-wrong/), reject, unstart, block.
6. [Epics and ETA](06-epics/), roll-ups, projected ship dates.
7. [Release markers](07-release-markers/), date stakes in the priority list.
8. [The Done panel](08-done/), historical sprints as facts.
9. [Search and filter](09-search/), by text, owner, tag, epic, status.
10. [Comments and tasks](10-comments-tasks/), conversation around a story.
11. [Working with an AI agent](11-mcp-and-agents/), the MCP wire in detail.

## How to read each chapter

Each moment is a single operation shown three ways. The table reads
left-to-right: GUI, then the CLI verb you'd type to do the same
thing, then the JSON-RPC frame an MCP client would send.

The MCP column is collapsed by default. Click "Full envelope" to
see the literal request and response, same bytes an AI agent
exchanges with the project.
