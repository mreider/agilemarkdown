## About the project

Agilemarkdown is a project for managing agile projects using markdown files.

The project consists of:

1. A command line tool (CLI) that creates backlogs, adds stories, and syncs with remote git repos
2. A framework for hosting backlogs on a secure website
3. A way to edit files, in a browser, for web users who don't know git or use text editors
4. A way to organize external ideas, label things, send comments via email, track velocity, etc.

## Installing the CLI

To install the CLI tool you need to have [GO](https://golang.org/doc/install) installed.

Get the Go library

```
go get -u github.com/mreider/agilemarkdown
```

Compile the code

```
cd $GOPATH/src/github.com/mreider/agilemarkdown
./build.sh
```

Create an alias for the binary

```
$GOPATH/bin/agilemarkdown alias am
```

## Creating a new backlog

sdfsdf

```
git clone git@github.com:mreider/agile-project.git
Cloning into 'agile-project'...
remote: Counting objects: 4, done.
remote: Compressing objects: 100% (4/4), done.
remote: Total 4 (delta 0), reused 0 (delta 0), pack-reused 0
Receiving objects: 100% (4/4), done.
```

With the Github repo on your local machine you can create your first backlog using agilemarkdown.

```
cd agile-project
am create-backlog paint the house
```

Cool. Now you have a backlog named paint-the-house. You also have a few new files, and folders, on your machine:

- index.md is an overview page that shows all of your backlogs. So far we only have one.
- agile-project.md is a project page that shows all of the stories in your backlog. We don't have any yet.
- agile-project is a folder that will contain all of your stories. So far we have none.
- ideas is a folder where users will drop ideas that you can decide to put in your backlog


## Creating stories in a backlog

Use the `create-item` command to create items in your backlog. The story is created as a markdown file.

```
cd paint-the-house
am create-item figure out which colors to buy
ls
figure_out_which_colors_to_buy.md
```

Once the file is created you can start editing the story using your favorite text editor. We like [Atom](https://atom.io/).

![Edit the story in a text editor](https://monosnap.com/image/dgLPinN9gCJLTruBMqTwGdwb0sllan.png)

After you edit the file, you could sync to the wiki using `am sync` to generate new project pages and push everything up to github.

`am sync`

## Managing stories in a backlog

To manage stories you must set different keys at the top of each story file. These keys are as follows:

- **Assigned**: The engineer who will work on the story. You can see what stories they are working on using the `am work` command, and in the project overview.

- **Estimate**: The number of points for this story. This number is used to generate the progress graphs that are available in `am progress` and on the overview page.

- **Status**: The status of a story describes whether it is unplanned, planned, doing, or finished.

### Status overview

| Status | Explanation |
|---|---|
| Unplanned | You are figuring these stories out. These stories need some research, clarity, approval, or input from other people in order to prioritize them.|
| Planned   | These stories are well understood and important. Your team will work on these stories soon.|
| Doing     | These are the stories your engineers are working on right now.|
| Finished  | All the things your team as completed since the project started.|


### Editing keys

By default, when you create a new story, it us unplanned. If the story is important - you need do the research required to move it into the planned section.

Stories in the planned section should be pointed, and have an assignee, though this is not required. You can change those items directly in the story.

![Change estimate and assignee ](https://monosnap.com/image/axlg12y1EqPaFokU8Uw2T1IuvzSioT.png)

You could also change the status of the story to `planned` in the editor or use the `change-status` command as follows:


```
am change-status -s u
Status: unplanned
------------------------------------------------------------
  # | User       | Title                          | Points
------------------------------------------------------------
  1 | falconandy | figure out which colors to buy |      1
------------------------------------------------------------

Enter a number to a story number followed by a status, or e to exit
1 p
Enter a number to a story number followed by a status, or e to exit
e
```

The `change-status` command takes a status argument `-s` - we passed `u` (unplanned) to view a list of unplanned stories. From that list we can move as many stories as we want from unplanned to planned. This command is intended for sprint planning meetings when you want to move a handful of stories from one status to another without opening each one separately.

### Listing stories

Use the `am work` command to see a list of stories. You can also pass a status like `-s p` to see a list of stories in a certain status.

```
am work -s p
Status: planned
------------------------------------------------------
User       | Title                          | Points
------------------------------------------------------
falconandy | figure out which colors to buy |      1
------------------------------------------------------
```

### Changing priorities

Things in the planned section should be stack ranked, with the most important story at the top. This is how engineers know which story to work on next. To change the order of the stories in any status list, you must open the project page for the project and cut / paste things according to your plans.

To edit the overview page, go to the root directory of your git repo. The overview page has the same name as your backlog.

```
pwd
/Users/mreider/agileproject/agileproject.wiki/paint-the-house
cd ..
ls
Home.md			_Sidebar.md		my-backlog.md		paint-the-house		paint-the-house.md
atom paint-the-house.md
```

![Alt text](https://monosnap.com/image/Gjkt9GCEsYg3u55nks1kdhnqQmavMn.png)

### Syncing the project page

The project page is destroyed and regenerated every time you sync. The order of your stories will be preserved when it is regenerated. If you change a story's status, the story will appear at the bottom of that list next time you sync.

### Measuring velocity

The main page of the Wiki shows how many points your team has landed over the course of the last few weeks. You can also look at this by using the command `am progress`

![Alt text](https://monosnap.com/image/sqrDGVQVmwFRWQVFyuOYEKtjlmoy6p.png)

## Working as a team

### Asking for a clarification

You can make a comment in your markdown file when you need clarify something with a teammate. Agilemarkdown will read tagged usernames, starting with an @ symbol, and put a list of clarification requests at the top of your project page.

![Tagging a user](https://monosnap.com/image/pXM5u3aOH6C8TVm12L8STprOTXpaq8.png)

The next time you run `am sync` this clarification request will appear at the top of your project page.

### Resolving a clarification

Clarifying something could be done in the story itself, or by making another comment for the user who asked for the clarification. To get the clarification out the list, put a space, or a tab in front of the @username in the comment section. This will remove the clarification from the project page, but keep the comment intact in the story.

## Importing stories from Pivotal Tracker

We switched to agilemarkdown from Pivotal Tracker. We also built an import command for Pivotal Tracker backlogs. Begin by exporting your tracker backlog using the export feature.

![Pivotal Tracker Export](https://monosnap.com/image/mW5fJGIPxEkI2niaMDAaVcT4DsUnpJ.png)

Now import the csv file using the `am import` command. You can test this yourself using this [file](https://gist.github.com/mreider/e346d82c82b4d20d53858565aaa8470e).

```
am import ~/Downloads/paint_our_house_20180430_1429.csv

```

Now use the `am work` command to see all of the stories you just created.

```
am work
Status: doing
-------------------------------------------------------------
 User        | Title                                | Points
-------------------------------------------------------------
 Matt Reider | Buy supplies at OSH                  |      2
 Matt Reider | Figure out which paint colors to buy |      5
 Matt Reider | Lay out our work areas               |      2
-------------------------------------------------------------

Status: planned
----------------------------------------------------
 User | Title                              | Points
----------------------------------------------------
      | Apply two coats to siding and trim |      8
      | Prep the house                     |      8
      | Prime the siding and trim          |      5
----------------------------------------------------

Status: unplanned
--------------------------------------------------------------------
 User | Title                                              | Points
--------------------------------------------------------------------
      | Take "after" pictures of every part of the house.  |      2
      | Take "before" pictures of every part of the house. |      2
--------------------------------------------------------------------
```
