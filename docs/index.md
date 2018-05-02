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
0-overview.md		smart-automerge.md
```

Once the file is created you can start editing the story using your favorite text editor. We like [Atom](https://atom.io/)

![Edit the story in a text editor](https://monosnap.com/image/qZq7zbKCBbUMYJXOXjg2OodzP4gZMc.png)

After you edit the file, you can sync to the wiki to make your changes are available.
