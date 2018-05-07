## About the project

Agilemarkdown is framework for managing agile projects using markdown files, rather than tools like JIRA, Pivotal Tracker, Trello, etc. The arguments for using simple markdown files to manage a project are:

- It's easy to understand. All of your artifacts are just files.
- It's easy to read. Stakeholders won't be distracted by a heavy or distracting interface.
- It integrates well with Github, so it's convenient for your engineers to read and update.
- It can be used offline. Editing files can be done anywhere, with or without an Internet connection.
- The history of your project is preserved in Git.
- It forces conversations by forcing you to git merge a story conflict.

## Getting started

Before you get started visit the [readme page](https://github.com/mreider/agilemarkdown) and install the agilemarkdown CLI.

The best way to get started is to use Github's Wiki feature as a place to manage agilemarkdown backlogs. If your team uses a number of Github repositories, or already uses the wiki of the main repository, you should create a new repo just for backlog management. If your team uses one Github repository, and there is no existing wiki content, use that.

Since you can't clone an empty Github wiki, you must create a home page. Begin by accessing your repo's wiki and creating the first page. It's just a placeholder for now - with a simple welcome message.

![Github Wiki Welcome Message](https://monosnap.com/image/hxSCiIhhs67Af8ym5TgWb3JllBjvXq.png)

After creating the first page, you can clone the wiki to your local machine. Github Wiki repositories use the repository address with the word 'wiki' in the file extension:

```
git clone git@github.com:mreider/agilemarkdown.wiki.git agile-backlog
Cloning into 'agilemarkdown.wiki'...
remote: Counting objects: 3, done.
remote: Total 3 (delta 0), reused 0 (delta 0), pack-reused 0
Receiving objects: 100% (3/3), done.
```

With the Github Wiki page on your local machine you can create your first backlog using agilemarkdown

```
cd agile-backlog
am create-backlog our-project
```

Cool. Now you have a backlog named our-project. Now sync your project with Github and look at your wiki page online. The overview page should appear with nothing in your backlog. That overview page is automatically generated based on the items you have created and landed.

```
am sync
```

![Github wiki project view](https://monosnap.com/image/Myuk0ga2ZLYJ0FE3bAQ15g86NVEInt.png)

## Creating stories in a backlog

Use the `create-item` command to create items in your backlog. The story is created as a markdown file.

```
am create-item smart-automerge
ls
smart-automerge.md
```

Once the file is created you can start editing the story using your favorite text editor. We like [Atom](https://atom.io/).

![Edit the story in a text editor](https://monosnap.com/image/qZq7zbKCBbUMYJXOXjg2OodzP4gZMc.png)

After you edit the file, you could sync to the wiki using `am sync` to make your changes are available. If you do this, you might notice that the home page on the wiki looks the same as it did before. That's because nobody has started working on these stories. This will be explained in the next section.

## Managing stories in a backlog

To manage stories you must set different keys at the top of each story file. These keys are as follows:

- **Assigned**: The engineer who will work on the story. You can see what stories they are working on using the `am work` command, and in the project overview.

- **Estimate**: The number of points for this story. This number is used to generate the progress graphs that are available in `am progress` and on the overview page.

- **Status**: The status of a story describes whether it is unplanned, planned, started, or finished. Instead of using these terms, though, we think about stories like airplanes. If a story is in the **hangar** it is not planned. It's just sitting there, waiting for a mechanic to show up and get it ready. This is where stories begin. If a story is at the **gate** it's ready to go, but not actually flying yet. Nobody is working on it, but it's planned to depart at some point. If a story is **flying** someone is working on it. It's in flight, going from point A to B, and eventually it will be **landed**, which means it's done.

### Status overview

✈️ ✈️ ✈️ ✈️ ✈️ ✈️

| Hangar | Gate | Flying | Landed |
|---|---|---|---|
| Not planned | Planned | Being worked on  |  Finished |


### Prioritizing a story

If a story is important - you need to get it out of the hangar, into the gate and up into the air. By default, when you create a new story, it lives in the hangar. Let's begin by creating a new story to fix the overview page. Notice that spaces in the story are replaced with underscores.

```
am create-item overview page needs unique names
ls
overview_page_needs_unique_names.md	smart-automerge.md
```

Now we open our favorite text editor and outline the story a bit, so the engineer can understand it, and talk to us about it. That's what the hangar is for. A place to get things ready.

![Getting a story ready](https://monosnap.com/image/i3yxrZ7qP5hjbi4oBsLDoiqehmG02C.png)

Once we agree on the story, and it's ready to go, the engineer should give it an estimate. Then we can move it to the gate.

![Giving an estimate](https://monosnap.com/image/9RNiCCjt5s9Duaupiqmml8kzuRsvGL.png)

Next time we sync, the story will show up on the Wiki, in the gate, ready to go.

![Overview shows things at the gate](https://monosnap.com/image/Xg79Lit5hu9dSpHalz6vl7latzmokI.png)

From the engineer's perspective, the stories at the gate should be stack ranked - with the most important story at the top of the gate. That way, the engineer will know what to work on next. To change the stack rank of the work at the gate, simply edit the overview page, and the order of things in that section. Before we do that, let's move the other story into the gate using the `am change-status` command.

```
am change-status -s h
Status: hangar
--------------------------------------------
   # | User       | Title           | Points
--------------------------------------------
   1 | falconandy | smart-automerge |
--------------------------------------------

Enter a number to a story number followed by a status, or e to exit
1 g
Enter a number to a story number followed by a status, or e to exit
e
```

Now there are two stories at the gate, which we can see using `am work`

```
m work -s g
Status: gate
-------------------------------------------------------
 User       | Title                            | Points
-------------------------------------------------------
 falconandy | overview page needs unique names |      1
 falconandy | smart-automerge                  |
-------------------------------------------------------
```

To change the priority of these, go to the root directory of the wiki and edit the order of the stories in a text editor.
