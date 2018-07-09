---
title: Creating a backlog
keywords: create backlog
permalink: create.html
---

# Creating a backlog

To create a new backlog use the `am create-backlog` in a git initialized directory:

```
mkdir mail-stuff-inc
cd mail-stuff-inc
git init
am create-backlog compress things
```

This creates a new directory named `compress-things` in the top directory `mail-stuff-inc.` You can create as many backlogs in the top directory as you want. Each will be located in its own subdirectory.

# Creating stories

To create a story in the `compress-things` backlog, switch directories and use the `am create-item` command:

```
cd compress-things
am create-item compress objects under 1GB
```

This will create a markdown file with some placeholders. You can edit the file using your favorite text editor. We like [Atom](https://atom.io/) with a markdown preview tab open.

# Syncing with Github

foo bar buzz

{% include links.html %}
