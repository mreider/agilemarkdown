---
title: Getting started
keywords: getting started, agile markdown
permalink: getting-started.html
---

# Getting started

You can download the latest CLI from the releases page [here](https://github.com/mreider/agilemarkdown/releases). After downloading, give the binary privileges and run the alias command.

```
chmod 755 agilemarkdown
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


{% include links.html %}
