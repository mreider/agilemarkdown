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

![Alt text](https://monosnap.com/image/hxSCiIhhs67Af8ym5TgWb3JllBjvXq.png)

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

Cool. Now you have a backlog named our-project. Now sync your project with Github and look at your wiki page online.

```
am sync
```
