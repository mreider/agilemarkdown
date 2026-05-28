# 1. Inception

Before the first story, agilemarkdown asks you to write down what
the work is actually for; that document is the inception.

## 1.1 Initialize a project

agilemarkdown projects are git repos. They have a `.am/`
directory for derived state and one folder of stories at the
root. `am create-backlog` makes both.

<table>
<tr><th>Board</th><th>CLI</th><th>MCP</th></tr>
<tr>
<td>

The board for this moment is your file system. After
`am create-backlog`, you have a <code>stories/</code> directory
next to a <code>.am/</code> directory in the project root, both
filled with plain markdown ready for git.

</td>
<td>

```
$ cd projects/meadow-letters
$ git init
Initialized empty Git repository
$ am create-backlog
created backlog: stories
created .am/config.yaml
```

</td>
<td>

`create_backlog` does the same: creates the directory, seeds the
config, returns the path. Equivalent to the CLI verb. Useful when
an agent is bootstrapping a fresh project on a fresh repo.

<details><summary>Full envelope</summary>

```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "create_backlog",
    "arguments": { "name": "stories" }
  }
}
```

</details>

</td>
</tr></table>

## 1.2 Write the inception

`am inception` seeds `.am/inception.md` with prompts for the six
sections agilemarkdown asks every project to articulate:

- **The user**, a sentence about who the work is for
- **The goal**, one sentence the team agrees with
- **The reason**, why now, why this team
- **Success**, what has to be true for the work to have mattered
- **Constraints**, hard limits on time, scope, dependencies
- **Out of scope**, what you're consciously not doing

You edit the file in any editor (VS Code, vim, the lab's
Terminal tab, same file). When the document feels honest, you
commit it. Stories come after.

<table>
<tr><th>Board</th><th>CLI</th><th>MCP</th></tr>
<tr>
<td>

The inception lives at <code>.am/inception.md</code>. Open it in
any markdown editor, VS Code's preview, vim, the lab's Terminal
tab. Same file, same content; no agilemarkdown-specific UI is
needed because markdown is the UI.

</td>
<td>

```
$ am inception
seeded .am/inception.md (edit it; ask the team)

$ am inception --show
# Inception

## The user
(describe the people you are building for)

## The goal
(one sentence the team agrees with)

## The reason
(why now, why us)

## Success
(what specifically must be true for this to have worked)

## Constraints
- (hard limits on time, scope, dependencies)

## Out of scope
- (what you're consciously not doing)
```

</td>
<td>

`inception_doc` reads or writes the file. No args returns the
current content (or the seeded template). Passing `text:`
overwrites the file with what the agent thinks the project's
inception should say.

<details><summary>Full envelope</summary>

```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "inception_doc",
    "arguments": {}
  }
}
```

</details>

</td>
</tr></table>

## 1.3 Record team agreements

Team agreements are anything the team has decided matters: "bugs at
the top of priority," "no merges Friday afternoon," "questions in
#project-meadow within a day." The coach warns when an action would
break one but never refuses; the team can override with a reason.

<table>
<tr><th>Board</th><th>CLI</th><th>MCP</th></tr>
<tr>
<td></td>
<td>

```
$ am team-agreements --add "Bugs go to the top of priority"
recorded: 1 agreement
$ am team-agreements --add "8-point hard cap. Above 3 is a red flag."
recorded: 2 agreements
$ am team-agreements --add "PM rejects with a reason"
recorded: 3 agreements
```

</td>
<td>

`team_agreements` with `add:` appends. With no args it returns
the current list. Useful for an agent to surface in a prompt
like *"the team has decided: …"* whenever the agent is about to
do something that touches a soft rule.

<details><summary>Full envelope</summary>

```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "team_agreements",
    "arguments": {
      "add": "Bugs go to the top of priority"
    }
  }
}
```

</details>

</td>
</tr></table>

### What just happened

- A new git repo became an agilemarkdown project (one
  `am create-backlog` + a `.am/` directory)
- The inception lives in a markdown file in `.am/inception.md`,
  open in any editor, commit with the rest of the repo
- Team agreements live in `.am/team-agreements.md`, one bullet
  per agreement, dated
- All three artifacts are plain files in git. No SaaS, no
  database, no central server.

Next: [Create the first story →](../02-first-story/)
