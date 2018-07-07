---
title: How it works
keywords: overview, how it works
permalink: how-it-works.html
---

## What is Agile Markdown?

1. A command line interface (CLI) for managing backlogs and stories in markdown using git
2. A framework for managing backlogs and stories, locally, using any text editor
3. A simple way to collaborate using a Google oAuth protected website
4. A way to gather ideas (also in markdown) from different people and connect them to stories
5. A simple set of generated charts showing a backlog's velocity
6. A timeline of forecasted delivery - a.k.a. gantt charts

## Who can use it?

Anyone in your organization can use Agile Markdown.

- **Engineers and technical product managers** can use Agile Markdown's command line interface (CLI) to manage backlogs, sync backlogs with other git users, and collaborate with one another. They can also set up an Agile Markdown website so that other users can participate.

- **Non-technical product managers** can use an Agile Markdown website to manage backlogs and collaborate with other users.

- **Subject matter experts and other stakeholders** can use an Agile Markdown website to suggest ideas, check on progress, and respond to comments made by other users.

## What's the workflow?

You can use Agile Markdown for any agile, scrum, or xp project. Like most tools, this one is opinionated about how work should flow. I wrote the following narrative to describe the process I had in mind when I built the tool. It supports this process quite nicely.

### Writing stories

Ok. So let's pretend you're a product manager for Puppy Pants Inc., and your CEO is all upset. "People aren't buying puppy pants!" she says. "Figure this out! Fix it!"

Off you go to dig into the problem. After some user interviews, data collection and such - you come to a few conclusions. First, the sign-up page is in Spanish, and most of your users are French. Second, your Google ranking is 1,650 for search terms like, "pants for puppies." Lastly, your online credit card verification only supports Discover Card.

Using Agile Markdown, you create a new backlog and call it **Sell more pants**. Then write some stories, in Markdown of course, about the problems you discovered.  When you created these stories, Agile Markdown created a little template for you, based on these important sections:

- **A short title** that indicates what the story is about. Something like "Support for French."

- **A problem statement** that explains the problem in a sentence or two - with some data if possible. "72% of home page visitors never login to the site. This might be because 86% of our users speak French, but our website is in Spanish."

- **A potential solution** that suggests how you might solve the problem. "Build support for different languages - and start with French."

- **Comments** that explain why you are suggesting the solution, and any other pertinent information.

- **Attachments** that reference the data you used to come to your conclusions.

### Planning wit your team

You schedule a planning meeting with your team. Before the meeting - you need to prioritize your stories in terms of importance. The language problem is the most important one to fix since your users can't even read the site. The search rank problem is the second priority since nobody visits a site they can't find. The credit card problem is the least important. Using Agile Markdown, you edit the **Sell more pants** project page and put these stories in a stack ranked order.

Now that your stories are prioritized, you walk into the planning meeting and begin the discussion.

The goals for the planning meeting are:

1. To tell the team what you have learned about each problem and see if they agree with your thinking.
2. To debate your potential solution and identify the information you need to make a decision.
3. To scope the solution, if you agree that it makes sense, in terms of [story points](https://www.mountaingoatsoftware.com/blog/what-are-story-points).
4. To debate the priority of the stories, and see how you can break them down into [smaller stories](https://www.mountaingoatsoftware.com/blog/five-simple-but-powerful-ways-to-split-user-stories), if possible.
5. To get psyched about the work ahead and make sure everyone agrees with the plan.

You accomplish these goals by showing your team the stories you have written, and the prioritized list in the **Sell more pants** product page. As the discussion unfolds, you (optionally) add values to some pre-populated keys in the generated stories:

- **Points** your team estimated for the story (we like [fibonacci](http://www.velocitycounts.com/2013/05/why-do-high-performing-scrum-teams-tend-to-use-story-point-estimation/) pointing).
- **Status** of the story, based on whether it's **unplanned**, **planned**, **doing**, or **finished**
- **Assigned** to a particular engineer who might take on the work

Of course - all of this is over-ambitious for one meeting. More likely you will have a few.

After you are done planning - you talk with your engineering lead to get some idea for when these stories might start and end. It's not an exact science. Then you use Agile Markdown to build a timeline / [Gantt](https://en.wikipedia.org/wiki/Gantt_chart) chart for your CEO, so she can get some warm fuzzies about when stuff might be fixed.

You use Agile Markdown to sync your stories with the github repository that contains all of these Markdown files. You also use Agile Markdown to set up a secure website with an exact copy of the Github repo, which syncs on its own, so your CEO, and other stakeholders, can look at the stories, your progress, and the charts you generated.

### The sprint

Your engineering team starts working on the first story, "Support for French." The engineer who picks up the story uses Agile Markdown to change the story's status from **planned** to **doing**. But there's a problem. She realizes that she doesn't know French well enough to do the actual translation. The engineering manager wonders if the goal of the story should be simplified - maybe she should build some tools for a French speaker to translate each page, rather than doing the translation herself. She makes a comment in the story, and uses Agile Markdown to send you a message with the comment, and a link to the story on both Github, and the secure website, to ask for clarification.  While she waits for your response, she changes the status back to **planned** and starts working on the search result stories instead.

Using Agile Markdown, you respond that the translation story should be rewritten. You rewrite the story with a new title, problem statement, and a potential solution of internationalizing the website. Then you write a new story to localize the website in French with help from a native speaker. You ask the team to repoint the stories, and you change your Gantt chart so your CEO is informed. Agile Markdown syncs all of your changes and sends your response to the engineer, who is ready to re-start the story.

By the end of the sprint your team has delivered a few stories. You have a retrospective meeting and review the **finished** section of the **Sell more pants** project page. You talk about what happened during the sprint, and how you can do better next time.

### Using velocity

As you work from sprint to sprint, you look at how many points your team accomplished, using Agile Markdown's velocity chart. You wonder if your  [velocity](https://www.pivotaltracker.com/blog/velocity-is-a-measure-of-pace-not-productivity) is predictable. Typically, a team has higher velocity as they get closer to releasing a new feature, because surprises slow you down at the beginning ('nobody knows French!'). Looking at Agile Markdown's velocity charts helps you in a few ways:

1. You estimate how many points your team can accomplish next sprint based on past performance
2. You recognize the kinds of stories that your team underestimated, and improve their estimation next time
3. You see the impact of bugs / defects and decide to spend more time on cleaning the existing features - rather than building new ones

### Gathering ideas

It seems like everyone has an idea on how to improve the Puppy Pants application. A marketing manager wants to coordinate a campaign for schnauzers and dachshunds. Someone on the sales team wants to apply discounts for their biggest corporate partners. A customer service rep prefers chatting with customers rather than using email. How do you decide what is most important and communicate what will be done, and when?

Lie




{% include links.html %}
