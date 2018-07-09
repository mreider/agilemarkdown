---
title: How it works
keywords: overview, how it works
permalink: how-it-works.html
---

## How it works

You can use Agile Markdown for any agile, scrum, or extreme programming (xp) project. Like most tools, Agile Markdown is opinionated about how you manage things.

## Writing stories

Let's pretend you're a product manager for Mail Your Stuff Inc., an app for attaching physical objects to email. Your CEO is all upset. "Only six people have signed up this week!" she says. "We need more users!" You talk to your product team and ask for some engineers to focus on the problem.

After some user interviews, data collection and such - you arrive at a theory about Mail Your Stuff's popularity problems. Small physical objects require at 8 petabytes of email storage. Larger items, like furniture or factory equipment, demand as much as 100 exabytes. Most users don't have enough email storage to use the service.

As an experiment, you try to shrink some chairs and tables using Winzip, but the files are still massive. It looks like you need to come up with some new kind of compression - a much better version of Winzip. If you can't shrink this stuff down Mail Your Stuff won't make it as a company - your stock will be worthless.

Using Agile Markdown, you add a new backlog to product team's set. You call the backlog **Compress things**. Then you write some stories, in Markdown of course, about the problem. As you create these stories, Agile Markdown guides you to use the following structure:

- **A short title** that indicates what the story is about. Something like "Compress objects under 1GB"

- **A problem statement** that explains the problem in a sentence or two - with some data if possible. "44% U.S. email is on Gmail, which is limited to 30GB. Sending a table, or a desk, must be done using less than 1 GB or nobody will use our service."

- **A potential solution** that suggests how you might solve the problem. "Use zip, tar, and gzip over and over again in a loop, using the new lossy regenerative Chromatic library (™)."

- **Comments** that explain why you are suggesting the solution, and any other pertinent information.

- **Attachments** that reference the data you used to come to your conclusions.

## Planning with your team

You schedule a planning meeting with your team. Before the meeting - you need to prioritize your stories in terms of importance. The compression problem is the most important one to fix since your users don't have enough storage. After that, you want to update the malware scan so it can read your new compressed format and pick up any threats. Using Agile Markdown, you edit the **Compress things** project page and put these stories in a stack ranked order.

Now that your stories are prioritized, you walk into the planning meeting and begin the discussion.

The goals for the planning meeting are:

1. To tell the team what you have learned about each problem and see if they agree with your thinking.
2. To debate your potential solution and identify the information you need to make a decision.
3. To scope the solution, if you agree that it makes sense, in terms of [story points](https://www.mountaingoatsoftware.com/blog/what-are-story-points).
4. To debate the priority of the stories, and see how you can break them down into [smaller stories](https://www.mountaingoatsoftware.com/blog/five-simple-but-powerful-ways-to-split-user-stories), if possible.
5. To get psyched about the work ahead and make sure everyone agrees with the plan.

You accomplish these goals by showing your team the stories you have written, and the prioritized list in the **Compress things** product page. As the discussion unfolds, you (optionally) add values to some pre-populated keys in the generated stories:

- **Points** your team estimated for the story (we like [fibonacci](http://www.velocitycounts.com/2013/05/why-do-high-performing-scrum-teams-tend-to-use-story-point-estimation/) pointing).
- **Status** of the story, based on whether it's **unplanned**, **planned**, **doing**, or **finished**
- **Assigned** to a particular engineer who might take on the work

Of course - all of this is over-ambitious for one meeting. More likely you will have a few.

After you are done planning - you talk with your engineering lead to get some idea for when these stories might start and end. It's not an exact science. You use Agile Markdown to build a timeline / [Gantt](https://en.wikipedia.org/wiki/Gantt_chart) chart for your CEO, so she can get some warm fuzzies about when the site's popularity might increase.

You use Agile Markdown to sync your stories with the github repository that contains all of these Markdown files. Your engineers love working this way, because they edit files all day, and build tools to update the stories automatically without much effort. You also use Agile Markdown to set up a secure website with an exact copy of the Github repo, which syncs on its own, so your CEO, and other stakeholders, can look at the stories, your progress, and the charts you generated.

## The sprint

Your engineering team starts working on the first story, "Compress objects under 1GB." One of your engineers uses Agile Markdown to change the story's status from **planned** to **doing**. But there's a problem. The Chromatic library (™) uses a commercial license and it costs $10 million. She makes a comment in the story, and uses Agile Markdown to send you a message with the comment, and a link to the story on both Github, and the secure website, to tell you the problem.  While she waits for your response, she changes the status back to **planned** and starts working on the malware story instead.

Using Agile Markdown, you respond that there is an alternative to the Chromatic library (™) called Multihued ®. You rewrite the story with a new potential solution. You ask the team to repoint the stories, and you change your Gantt chart so your CEO is informed. Agile Markdown syncs all of your changes and sends your response to the engineer, who is ready to re-start the story.

By the end of the sprint your team has delivered a few stories. You have a retrospective meeting and review the **finished** section of the **Compress things** project page. You talk about what happened during the sprint, and how you can do better next time.

## Using velocity

As you work from sprint to sprint, you look at how many points your team accomplished, using Agile Markdown's velocity chart. You wonder if your  [velocity](https://www.pivotaltracker.com/blog/velocity-is-a-measure-of-pace-not-productivity) is predictable. Typically, a team has higher velocity as they get closer to releasing a new feature, because surprises slow you down at the beginning. Looking at Agile Markdown's velocity charts helps you in a few ways:

1. You estimate how many points your team can accomplish next sprint based on past performance
2. You recognize the kinds of stories that your team underestimated, and improve their estimation next time
3. You see the impact of bugs / defects and decide to spend more time on cleaning the existing features - rather than building new ones

## Gathering ideas

It seems like everyone has an idea on how to improve Mail Your Stuff's application. A marketing manager wants to coordinate a campaign for emailing domesticated animals. Someone on the sales team needs a way to provide real-time price quotes. A customer service rep wants to attach 100 MB of raspberry chocolate cake to each support ticket, as a little gesture to ease your frustrated clients.

It's hard to say which of these ideas is a good one, what the eventual stories will look like, or which backlog they might end up in. Each of these folks uses Agile Markdown to create these ideas, outside of your backlogs, and sends your director a note asking if you can discuss them. You have an intake meeting to look at each idea, and you decide on a priority for each one, editing the **rank** setting in Agile Markdown so the stories are listed according to the team's goals.

Your team decides that attaching 100 MB of raspberry chocolate cake is the highest priority. Using Agile Markdown, they tag the idea "attach-cake" and write stories in the both the **compress stuff** and **food related** backlogs to deliver cake to your customers. Agile Markdown shows each story, across the two backlogs, and the status for each. The customer service rep checks the idea page all the time to see how things are coming. When everything is marked as **finised** the customer service rep sends cake mail to an mortgage broker in Cedar Rapids.


{% include links.html %}
