<p align="center">
  <img src="docs/apple-touch-icon.png" alt="agilemarkdown" width="180">
</p>

<h1 align="center">agilemarkdown</h1>

<p align="center"><em>A backlog manager that stores in markdown so any LLM reads it natively, and coaches you through the Pivotal-style XP workflow.</em></p>

<p align="center">
  <a href="https://github.com/mreider/agilemarkdown/actions/workflows/ci.yml"><img src="https://github.com/mreider/agilemarkdown/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
</p>

A tool for managing backlogs efficiently. Three pieces fit together.

**The backlog lives in git as markdown.** Each story is a plain file with YAML frontmatter on top and a body underneath. Priority is a ranked list; the icebox is a capture pile, not ranked. Velocity is computed from accepted points across a rolling window. The repo is the database, and any text editor works on it.

**The LLM sees what the human sees.** `am mcp` is a stdio MCP server with 55 tools, and any MCP-aware client connects (Claude Desktop, Claude Code, Cursor, Codex CLI). LLMs read markdown natively, so the agent and the human are looking at the same files at the same time.

**The Pivotal way ships as a coach.** The agent is the dev pair and the human is the product manager. The hard rules block real violations: features are capped at 8 points, bugs and chores are not estimated, the dev pair never accepts its own work, an iteration cannot silently overcommit beyond rolling velocity, and releases stay as date markers. Working agreements layer on top as nudges, and acceptance is the moment the human owns.

## Lineage

agilemarkdown is the way Pivotal built software, encoded in plain files for the AI era.

I worked at Pivotal Software for six years. Pivotal Labs was the consulting practice; [Pivotal Tracker](https://github.com/pivotaltracker) was the opinionated SaaS the team built for itself and shipped to anyone who wanted to work the same way. The two went hand in hand: Tracker only made sense if you also followed the workflow it assumed. This project is biased toward Pivotal Tracker on purpose. The original help center, captured by the [Internet Archive on 2017-05-03](https://web.archive.org/web/20170503214450/https://www.pivotaltracker.com/help/), is quoted inline throughout the site.

## Documentation

[https://agilemarkdown.com](https://agilemarkdown.com)

- [Tutorial](https://agilemarkdown.com/tutorial.html). End-to-end walkthrough of the Pivotal lifecycle (inception, plan, iteration, retro) on the same fixture as the test suite. Three surfaces: CLI, Claude Code, VS Code extension.
- [Reference](https://agilemarkdown.com/reference.html). Repository layout, item schema, state machine, velocity formula, MCP tools, CLI verbs, coach-mode wiring, and configuration.

## Install

The intended workflow is: VS Code with Claude Code installed, your code repo open, agilemarkdown added to the repo, the MCP server wired up. After that the agent reads the backlog and writes the code in one place.

**1. VS Code.** Install [VS Code](https://code.visualstudio.com).

**2. Claude Code.** Install [Claude Code](https://claude.com/claude-code) in VS Code. Other agents that read project files (Cursor, Codex CLI, Copilot) work too; the install step below ships an instruction file for each.

**3. The `am` binary.** Single static binary, available for Linux, macOS, and Windows on amd64 and arm64.

```
# macOS arm64 example
curl -L -o /usr/local/bin/am \
  https://github.com/mreider/agilemarkdown/releases/latest/download/agilemarkdown_darwin_arm64
chmod +x /usr/local/bin/am
am --version
```

Or build from source with [Go 1.23+](https://go.dev/dl/): `go install github.com/mreider/agilemarkdown@latest`. See the [latest release](https://github.com/mreider/agilemarkdown/releases/latest) for prebuilt binaries.

**4. Add agilemarkdown to your code repo.** Open your repo in VS Code and run:

```
cd ~/code/my-app
am init
```

This drops the coach-mode files (source body at `.claude/agilemarkdown-coach.md`, a thin `CLAUDE.md` that `@`-imports it, AGENTS.md, copilot-instructions, cursor rule, the six Claude Code skills, and the PreToolUse hook with settings.json). Existing files are never overwritten. Greenfield repos can use `am create-backlog` instead, which adds a backlog folder on top.

**5. Wire the MCP server.** Claude Code reads MCP config from `.mcp.json` at the workspace root. Add:

```json
{
  "mcpServers": {
    "agilemarkdown": { "command": "am", "args": ["mcp"] }
  }
}
```

For Claude Desktop, paste the same JSON into `~/Library/Application Support/Claude/claude_desktop_config.json` (or the equivalent for your OS). Cursor: Settings &rarr; MCP. Codex CLI: `~/.codex/config.toml` per the codex docs.

**6. Optional: the Agile Markdown VS Code extension.** A board view inside VS Code (Current, Icebox, Epics, detail panel, drag-and-drop, KPI tab, ⌘K search). The extension shells out to the same `am` CLI per click, so anything the board does is reachable from your shell too. Install from the marketplace once it ships, or build from `vscode/` in this repo.

**7. Optional: bash alias.**

```
$(go env GOPATH)/bin/agilemarkdown alias am
```

## Start a backlog

A backlog is a git repo. Empty or shared:

```
mkdir my-backlog && cd my-backlog
git init
am create-backlog product
cd product
am create-item "Build login flow"
am sync
```

`am create-backlog` drops three sets of files into the repo: the backlog folder itself with sample stories, the agent-instruction projections so any AI agent inherits the coach stance, and the Claude Code skills + PreToolUse hook under `.claude/` so the hard rules can actually block tool calls. Existing files are never overwritten.

The source coach body lives at `.claude/agilemarkdown-coach.md`. `CLAUDE.md` is a thin file that pulls it in via `@.claude/agilemarkdown-coach.md`, so a project that already has its own `CLAUDE.md` instructions can keep them and add a single import line. `AGENTS.md`, `.github/copilot-instructions.md`, and `.cursor/rules/coach.mdc` carry the same body inline since those formats do not support the `@` import.

Already have a backlog repo from before v4.4? Run `am init` in it to install or refresh the same set. Idempotent.

Users auto-discover from `git config` and `git log`. For a team-shared backlog, push the repo to GitHub. Each contributor clones it, runs `am` locally, and pushes their changes. Repo permissions are the access model; branch protection plus required reviews give backlog changes an approval flow.

`am sync` regenerates derived views (`index.md`, per-tag pages, `velocity.md`, `timeline.md`, `users.md`), validates each item against the JSON Schema, then commits and pushes if a remote is configured.

## Editing

Any markdown editor works. Items are YAML frontmatter on top with a markdown body underneath, so VS Code, Obsidian, nvim, Cursor, and the rest read them out of the box.

## AI integration

`am mcp` exposes the backlog as a Model Context Protocol server with 55 tools across read, write, run, and coach categories. Wire it into Claude Desktop, Claude Code, Cursor, or any MCP-aware client and the agent can:

- Read and write items, change status, run sync, validate the schema
- Stack-rank `_priority.md` and `_icebox.md`, bulk-promote icebox to priority preserving order
- Render velocity, iteration, burnup, and epic-burnup charts as ASCII inline in the chat
- Capture rejection reasons, hypotheses, learnings, and team agreements as plain markdown
- Run the coach: preflight an action against the hard rules (`coach_check`, including `action=pull` for pre-pull alignment), render the PM ceremony for a delivered story (`acceptance_prompt`), check whether the iteration fits rolling velocity (`iteration_fit`)
- Walk acceptance bullets: `list_acceptance`, `set_acceptance_state` (open / claimed / verified), `append_acceptance_bullet`

The same surface is reachable from the shell. Every read view has a `--json` sibling (`am dashboard --json`, `am show priority --json`, `am sprint plan --json`, `am show cfd --json`), and ten data-export verbs (`am list-backlogs`, `am list-items`, `am get-item`, `am get-comments`, `am type-mix`, `am cfd`, `am whoami`, `am history`, `am search`, `am set-description`) emit the same JSON shapes the MCP server returns. Pipe through `jq` for shell scripts, CI checks, or status-bar widgets without standing up an MCP client. See the [Data API reference](https://agilemarkdown.com/reference.html#data-api).

## Coach mode (AI as dev pair, human as PM)

The coach is the part of agilemarkdown that turns an MCP-aware agent into a practitioner of the Pivotal way. It ships in four layers.

**Words.** `am create-backlog` writes the source coach body to `.claude/agilemarkdown-coach.md` and projects it across the four agent-instruction formats: `CLAUDE.md` is a thin `@` import (so an existing CLAUDE.md is not overwritten), and `AGENTS.md`, `.github/copilot-instructions.md`, `.cursor/rules/coach.mdc` carry the same body inline. The body says: the agent is the dev pair, the human is the PM, here are the hard rules, here is how to run the acceptance ceremony.

**Tools.** Three MCP tools and CLI mirrors back the ceremonies: `coach_check` returns a structured verdict with the rule, its slug, and a suggested next move; `acceptance_prompt` renders the PM ceremony from a story file; `iteration_fit` reports whether the iteration overcommits.

**Skills.** Six Claude Code skills under `.claude/skills/` auto-fire on the right phrases: `am-inception` runs the kickoff, `am-plan` runs IPM, `am-align` runs the pre-pull restatement so the agent does not confidently build the wrong thing, `am-decompose` breaks a problem into Pivotal-sized stories, `am-accept` runs the PM acceptance ceremony bullet-by-bullet, and `am-retro` runs the end-of-iteration retro.

**Hooks.** A PreToolUse hook at `.claude/hooks/coach-gate.sh`, wired through `.claude/settings.json`, gates `set_status` and `set_estimate`. When Claude Code is about to flip a story to `accepted` or estimate a feature above 8 points, the hook intercepts and exits 2; the tool call aborts with the refusal message in the conversation. Words become teeth.

Run `am coach` in any agilemarkdown repo to see the coach's read on the project right now.

## Multi-user

Git provides the multi-user model. Concurrent edits across different items merge cleanly, attribution comes from the git author of each commit, and history is read with `git log path/to/item.md`. Access control is the repository's permissions. Generated views regenerate every sync; configure `.gitattributes` with `merge=ours` on those paths if their merges become noisy.

## Inspired by

The workflow comes from [Pivotal Tracker](https://github.com/pivotaltracker). agilemarkdown follows the same six-state machine, the same acceptance gate, fixed iteration windows, velocity computed from accepted points, the same story-type rules (features pointed by default, bugs and chores not), the icebox as the single intake list, and the bulk-promote step from icebox to backlog. See ["Why do we miss Pivotal Tracker?"](https://dwf.bigpencil.net/why-do-we-miss-pivotal-tracker/) for the rationale that drove the design.
