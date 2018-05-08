## About the project

Agilemarkdown is framework for managing agile projects using markdown files.

Here's why we built it:

- We like the flexibility of files over the opinionated structure of a web interface
- We'd rather use open source software than hand money over to Pivotal, Atlassian, Rally, or Aha
- We want to cut down on the number of tools we use and integrate with existing ones
- Our engineers spend all their time on the command line, in text editors, and in git
- We thought it would be fun to build it

## Getting started

Before you get started visit the [readme page](https://github.com/mreider/agilemarkdown) and install the agilemarkdown CLI.

The best way to get started is to use Github's Wiki feature as a place to manage agilemarkdown backlogs. If your team uses a number of Github repositories, or already uses the wiki of the main repository, you should create a new repo just for backlog management. If your team uses one Github repository, and there is no existing wiki content, use that.

Since you can't clone an empty Github wiki, you must create a home page. Begin by accessing your repo's wiki and creating the first page. It's just a placeholder for now - with a simple welcome message.

![Github Wiki Welcome Message](https://monosnap.com/image/VdA9yvJv9iWbYWqkccdpcU4XVt1kP6.png)
![First page](https://monosnap.com/image/6csVFgCZrTUWwWXAuGaSivpnmAuBAy.png)

After creating the first page, you can clone the wiki to your local machine. Github Wiki repositories use the repository address with the word 'wiki' in the file extension:

```
git clone git@github.com:mreider/agileproject.wiki.git
 Cloning into 'agileproject'...
 remote: Counting objects: 4, done.
 remote: Compressing objects: 100% (4/4), done.
 remote: Total 4 (delta 0), reused 0 (delta 0), pack-reused 0
 Receiving objects: 100% (4/4), done.
```

With the Github Wiki page on your local machine you can create your first backlog using agilemarkdown

```
cd agileproject
am create-backlog paint-the-house
```

Cool. Now you have a backlog named paint-the-house. Now sync your project with Github and look at your wiki page online. The overview page should appear with a link to your project and a graph of your velocity, which is zero of course.

```
am sync
```

![Project View](https://monosnap.com/image/F8raU3cNHhf5WpYVFptEPxfCfaTMjn.png)

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

After you edit the file, you could sync to the wiki using `am sync` to make your changes are available. If you do this, you might notice that the home page on the wiki looks the same as it did before. That's because nobody has started working on these stories - and nothing is planned.

From the home page you can click on the project page to see the story listed.

![Project page](https://monosnap.com/image/EC6liJITrt6Fwg6fmG3OaWDWQ89ZIc.png)

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

Use the `am sync` command after you make the change to keep the Github wiki up to date. The order will be preserved unless you move a story to a different section or delete a story from the list.

Moving a story to a different section (hangar to gate) will not work. You must edit the story or use `am change-status` instead.

### Measuring velocity

The main page of the Wiki shows how many points your team has landed over the course of the last few weeks. You can also look at this by using the command `am progress`

![Alt text](https://monosnap.com/image/sqrDGVQVmwFRWQVFyuOYEKtjlmoy6p.png)

## Importing stories from Pivotal Tracker

We switched to agilemarkdown from Pivotal Tracker. We also built an import command for Pivotal Tracker backlogs. Begin by exporting your tracker backlog using the export feature.

![Pivotal Tracker Export](https://monosnap.com/image/mW5fJGIPxEkI2niaMDAaVcT4DsUnpJ.png)

Now import the csv file using the `am import` command.

```
am import

```
