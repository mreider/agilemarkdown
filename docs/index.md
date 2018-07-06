What? You wanna manage backlogs in markdown? Are you a masochist?

Absolutely. Yes. I do. So I built Agile Markdown. Here's why:

1. It's easy. Backlogs are directories. Stories are files.
2. It's flexible. You can use any editor you want.
3. Non technical folks can learn markdown easily
4. Technical folks ♥ it cuz they can use a terminal all day
5. Conflicts in git are good - they encourage conversations

Agile Markdown consists of a handful of tools:

1. A command line interface (CLI) for managing backlogs and stories in markdown using git
2. A framework for managing backlogs and stories, locally, using any text editor
3. A simple way to share the backlog on a web server and sync everyone's copy using git
4. A way to gather ideas (also in markdown) from different people and connect them to stories
5. A simple set of generated charts showing a backlog's velocity
6. A timeline of forecasted delivery - a.k.a. gantt charts

## Using the CLI

You can download the latest CLI from the releases page [here](https://github.com/mreider/agilemarkdown/releases). You can put the binary in your path, or not, and run the following command to create the `am` alias:

```
./agilemarkdown alias am
Please, restart your terminal session to use the new alias
```

To see the list of commands use the `am help` command:

```
am help
NAME:
   agilemarkdown - A framework for managing a backlog using Git, Markdown, and YAML

USAGE:
   agilemarkdown [global options] command [command options] [arguments...]

VERSION:
   2018.07.05.102952

DESCRIPTION:
   A framework for managing a backlog using Git, Markdown, and YAML

COMMANDS:
     create-backlog  Create a new backlog
     create-item     Create a new item for the backlog
     create-idea     Create a new idea
     sync            Sync state
     work            Show user work by status
     points          Show total points by user and status
     assign          Assign a story to a user
     change-status   Change story status
     velocity        Show the velocity of a backlog over time
     alias           Add a Bash alias for the script
     import          Import an existing Pivotal Tracker story
     archive         Archive stories before a certain date
     help, h         Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
```

## Creating a new backlog

To create a new backlog use the `am create-backlog` command:

```
mkdir test
cd test
am create-backlog paint the house
```

This creates a new directory named `paint-the-house` and performs a git init in the top directory `test.` You can create as many backlogs in the test directory as you want, and each will be located in its own subdirectory.


## Creating stories in a backlog

To create a story in the `paint-the-house` backlog, switch directories and use the `am create-item` command:

```
cd paint-the-house
am create-item figure out which colors to buy
```

This will create a markdown file that you can edit using your favorite text editor. We like [Atom](https://atom.io/).

![Edit the story in a text editor](https://monosnap.com/image/dgLPinN9gCJLTruBMqTwGdwb0sllan.png)

After you edit the file, you could sync to the wiki using `am sync` to generate new project pages and push everything up to github.

`am sync`

## Managing stories in a backlog

To manage stories you must set different keys at the top of each story file. These keys are as follows:

- **Assigned**: The engineer who will work on the story. You can see what stories they are working on using the `am work` command, and in the project overview.

- **Estimate**: The number of points for this story. This number is used to generate the velocity graphs that are available in `am velocity` and on the overview page.

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

The main page of the Wiki shows how many points your team has landed over the course of the last few weeks. You can also look at this by using the command `am velocity`

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
