package main

import "html/template"

// Shared head: the agilemarkdown.com warm-paper palette, plus a
// small tutorial-specific block for the three-column moment tables.
// Kept inline so a generated chapter HTML file is self-contained, 
// matches the convention of the hand-crafted pages.
const sharedHead = `<head>
<meta charset="UTF-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>{{.Title}} · agilemarkdown tutorial</title>
<meta name="description" content="agilemarkdown tutorial · chapter {{.Num}}: {{.Title}}. Same operation shown three ways, the board, the CLI verb, the MCP JSON-RPC frame." />
<link rel="preconnect" href="https://fonts.googleapis.com" />
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
<link href="https://fonts.googleapis.com/css2?family=Source+Serif+4:opsz,wght@8..60,300;8..60,400;8..60,500;8..60,600;8..60,700&family=JetBrains+Mono:wght@400;500;600&display=swap" rel="stylesheet" />
<style>
  :root {
    --paper:   oklch(0.974 0.009 82);
    --paper-2: oklch(0.945 0.012 82);
    --paper-3: oklch(0.915 0.014 82);
    --ink:     oklch(0.205 0.015 60);
    --ink-2:   oklch(0.36  0.018 60);
    --ink-3:   oklch(0.52  0.018 65);
    --rule:    oklch(0.84  0.014 78);
    --rule-2:  oklch(0.78  0.016 78);
    --accent:  oklch(0.52  0.135 38);
    --accent-soft: oklch(0.85 0.06 38);
    --term-bg: oklch(0.18  0.012 60);
    --term-fg: oklch(0.93  0.012 82);
    --term-dim:oklch(0.62  0.012 82);
    --term-accent: oklch(0.78 0.13 38);
  }
  * { box-sizing: border-box; }
  html { scroll-behavior: smooth; }
  body {
    margin: 0; background: var(--paper); color: var(--ink);
    font-family: "Source Serif 4", Georgia, serif;
    font-size: 18px; line-height: 1.65;
    -webkit-font-smoothing: antialiased;
  }
  a { color: var(--accent); }
  code { font-family: "JetBrains Mono", ui-monospace, monospace; font-size: 0.9em; padding: 1px 4px; background: var(--paper-2); border-radius: 3px; }
  main { max-width: 1100px; margin: 0 auto; padding: 60px 28px 120px; }
  header.masthead { padding: 60px 0 40px; border-bottom: 1px solid var(--rule); margin-bottom: 50px; }
  header.masthead a.back { font-family: "JetBrains Mono", monospace; font-size: 12px; color: var(--ink-3); text-decoration: none; }
  header.masthead a.back:hover { color: var(--accent); }
  header.masthead .eyebrow {
    font-family: "JetBrains Mono", monospace;
    font-size: 11px; letter-spacing: 0.15em; text-transform: uppercase;
    color: var(--ink-3); margin: 18px 0 14px;
  }
  header.masthead h1 {
    font-size: clamp(34px, 6.5vw, 54px);
    font-weight: 600; line-height: 1.08;
    letter-spacing: -0.025em; margin: 0 0 14px;
  }
  header.masthead h1 em { font-style: italic; color: var(--accent); font-weight: 500; }
  header.masthead .deck { font-size: clamp(17px, 2.6vw, 21px); color: var(--ink-2); line-height: 1.45; max-width: 720px; }
  @media (max-width: 520px) {
    main { padding: 36px 18px 80px; }
    header.masthead { padding: 36px 0 26px; margin-bottom: 30px; }
  }
  h1 { font-size: 36px; font-weight: 600; letter-spacing: -0.022em; margin: 38px 0 18px; }
  h2 {
    font-size: 26px; font-weight: 600; line-height: 1.18;
    letter-spacing: -0.018em; margin: 50px 0 16px;
    border-top: 1px solid var(--rule-2); padding-top: 30px;
  }
  h3 {
    font-size: 19px; font-weight: 600; margin: 28px 0 10px;
    letter-spacing: -0.012em;
  }
  p { margin: 0 0 16px; }
  ul, ol { margin: 0 0 18px; padding-left: 1.25em; }
  li { margin-bottom: 6px; }
  blockquote { margin: 18px 0; padding: 4px 18px; border-left: 3px solid var(--accent); color: var(--ink-2); font-style: italic; }

  /* Three-channel moment table: GUI | CLI | MCP. Stack on narrow
     viewports. */
  table {
    width: 100%; border-collapse: collapse; margin: 22px 0;
    font-size: 14.5px;
  }
  th {
    text-align: left; padding: 8px 12px;
    font-family: "JetBrains Mono", monospace;
    font-size: 11px; letter-spacing: 0.12em; text-transform: uppercase;
    color: var(--ink-3); font-weight: 500;
    border-bottom: 1px solid var(--rule);
    background: var(--paper-2);
  }
  td {
    padding: 12px 12px;
    vertical-align: top;
    border-bottom: 1px solid var(--rule);
    width: 33.33%;
  }
  td img { max-width: 100%; height: auto; border-radius: 6px; border: 1px solid var(--rule); }
  td pre {
    margin: 0;
    background: var(--term-bg); color: var(--term-fg);
    padding: 10px 12px; border-radius: 5px;
    font-family: "JetBrains Mono", monospace; font-size: 12px;
    line-height: 1.55;
    overflow-x: auto;
    white-space: pre;
  }
  td pre code { background: transparent; padding: 0; color: inherit; font-size: inherit; }
  td details { margin-top: 8px; }
  td details summary {
    font-family: "JetBrains Mono", monospace; font-size: 11px;
    color: var(--accent); cursor: pointer; user-select: none;
    padding: 4px 0;
  }
  td details summary:hover { text-decoration: underline; }
  td details > * { margin-top: 8px; }
  @media (max-width: 760px) {
    table, thead, tbody, tr { display: block; }
    th { display: none; }
    td { display: block; width: 100%; border-bottom: none; padding: 8px 0; }
    td + td { border-top: 1px solid var(--rule); padding-top: 12px; }
    td::before {
      content: attr(data-channel);
      display: block;
      font-family: "JetBrains Mono", monospace;
      font-size: 10.5px; letter-spacing: 0.14em; text-transform: uppercase;
      color: var(--ink-3); margin-bottom: 6px;
    }
  }

  /* "What just happened" lists pop a touch so they read as
     summaries rather than continuation prose. */
  h3 + ul {
    background: var(--paper-2);
    padding: 12px 18px 12px 36px;
    border-radius: 6px;
    border-left: 3px solid var(--accent-soft);
  }

  /* Chapter nav (prev / next) under each page. */
  nav.pager {
    display: flex; justify-content: space-between; gap: 16px;
    margin: 70px 0 0; padding-top: 22px; border-top: 1px solid var(--rule);
    font-family: "JetBrains Mono", monospace; font-size: 13px;
  }
  nav.pager a { color: var(--ink-2); text-decoration: none; }
  nav.pager a:hover { color: var(--accent); }
  nav.pager .label {
    font-size: 10.5px; letter-spacing: 0.14em; text-transform: uppercase;
    color: var(--ink-3); display: block; margin-bottom: 4px;
  }
  nav.pager .next { text-align: right; }

  footer { margin-top: 80px; padding-top: 24px; border-top: 1px solid var(--rule); font-size: 14px; color: var(--ink-3); text-align: center; }

  /* Chapter index cards. */
  .chapter-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
    gap: 16px;
    margin: 30px 0 60px;
  }
  .chapter-card {
    display: block;
    padding: 18px 20px;
    background: var(--paper-2);
    border: 1px solid var(--rule);
    border-radius: 8px;
    text-decoration: none;
    color: var(--ink);
    transition: border-color 120ms, transform 120ms;
  }
  .chapter-card:hover {
    border-color: var(--accent);
    transform: translateY(-1px);
  }
  .chapter-card .num {
    font-family: "JetBrains Mono", monospace;
    font-size: 12px; color: var(--accent);
    letter-spacing: 0.08em;
  }
  .chapter-card .title {
    font-size: 20px; font-weight: 600;
    letter-spacing: -0.012em;
    margin: 4px 0 0;
  }
</style>
<link rel="stylesheet" href="surface-widget.css" />
</head>`

var chapterTpl = template.Must(template.New("chapter").Parse(`<!DOCTYPE html>
<html lang="en">
` + sharedHead + `
<body>
<main>
  <header class="masthead">
    <a class="back" href="tutorial.html">← Tutorial</a>
    <div class="eyebrow">Chapter {{.Num}}</div>
    <h1>{{.Title}}</h1>
  </header>
  {{.BodyHTML}}
  <nav class="pager">
    <div class="prev">
      {{if .Prev}}<a href="tutorial-{{.Prev.Num}}-{{.Prev.Slug}}.html"><span class="label">Previous</span>← {{.Prev.Title}}</a>{{end}}
    </div>
    <div class="next">
      {{if .Next}}<a href="tutorial-{{.Next.Num}}-{{.Next.Slug}}.html"><span class="label">Next</span>{{.Next.Title}} →</a>{{end}}
    </div>
  </nav>
  <footer>agilemarkdown · <a href="index.html">home</a> · <a href="reference.html">reference</a></footer>
</main>
<script src="surface-widget.js"></script>
</body>
</html>
`))

var indexTpl = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html lang="en">
` + indexHead + `
<body>
<main>
  <header class="masthead">
    <a class="back" href="index.html">← agilemarkdown</a>
    <div class="eyebrow">Tutorial</div>
    <h1>Walkthrough <em>tutorial</em></h1>
    <p class="deck">Every chapter shows the same thing three ways: the board, the <code>am</code> command line, and the MCP JSON-RPC an AI agent would call. By the end you'll know which surface to reach for, and you'll have seen the full mapping between them.</p>
  </header>

  <h2>The Pivotal week</h2>

  <p>The workflow this tutorial demonstrates comes from Pivotal Tracker, and before that from Extreme Programming. The shape is small: one ranked backlog, weekly sprints, three kinds of work, a few rituals to hold it together. The chapters demonstrate the mechanics; this page sketches the week so the mechanics make sense.</p>

  <p><strong>One ranked backlog.</strong> Different tools mean different things by "backlog." Here it is a single list of stories, ranked top to bottom. The top of the list is the current sprint, the work we expect to finish this week. A sprint is a week, not two, and it is not a contract for a fixed set of stories. Unplanned work arrives every week and the team finishes what it finishes. As people free up, they pull the next story off the top.</p>

  <p><strong>Velocity is a forecast.</strong> Each story carries an estimate in points, a rough sizing the team agrees on. agilemarkdown uses those points to place markers down the backlog showing which stories are likely to land this week, the next week, or later. The math accounts for team strength: who is sick, who is on vacation. Only estimated stories get markers, and the deeper into the backlog you look, the looser the projection. That blurriness further out is just what software actually feels like.</p>

  <blockquote class="pullquote">
    <p>"Velocity is the average normalized points completed across the last N iterations, where N is the project's Velocity Strategy setting (default 3, range 1-4). Velocity is computed once per iteration boundary and shown as an integer (it is floored on display and never rounded up)."</p>
    <p class="source">Pivotal Tracker help center · <a href="https://web.archive.org/web/20170503214450/https://www.pivotaltracker.com/help/articles/understanding_velocity/">understanding velocity</a></p>
  </blockquote>

  <p><strong>Three kinds of work.</strong> Features are stories: a unit of customer value, carrying a point estimate and an acceptance check. The dev pair pulls the story off the top and works it; the product manager accepts it once delivered. The dev pair never accepts its own work, and the coach refuses any attempt. Bugs go to the top of the list and carry no estimate, so a buggy stretch shows up automatically as a drop in velocity. Chores are work the team owes itself, CI setup, doc fixes after a customer complains, research the team needs, and also carry no estimate since they do not ship to a customer. Release markers are a fourth type but are not work; they are dated stakes in the priority list (Chapter 7).</p>

  <p><strong>The icebox.</strong> A backlog needs a place for ideas that are not ready to prioritize. The icebox is that place: a capture pile that is not ranked and is not being worked on. Engineers drop in features they think would be cool, the PM parks ideas before deciding, customer feedback lands there as it comes in. At planning time, items get pulled out, talked through, estimated if they are nearly ready, and ranked into the backlog at whatever position the PM decides.</p>

  <p><strong>The weekly rhythm.</strong></p>

  <ul class="rhythm">
    <li><strong>Monday plan.</strong> The team walks the top of the backlog together so the week is clear, estimates anything new with a quick fingers-up game called pointing poker, and slides out any story the PM cannot answer clearly. The aim is to leave ready to work, not stuck on questions.</li>
    <li><strong>Daily standup.</strong> A short go-round in which each person offers either an <em>interesting</em> (something they learned, or something another team is doing) or a <em>help</em> (I am stuck on X, who can pair). Helps often resolve in the meeting itself, the PM can pass without speaking, and a standup with nothing to report is a perfectly good thing.</li>
    <li><strong>Story time.</strong> On most Friday afternoons the PM and the engineering lead meet to prep for Monday, pulling icebox items out and estimating them so they can be ranked into priority.</li>
    <li><strong>Friday retro.</strong> At the end of the week the team takes a breath and puts three columns up on a board (<em>good</em>, <em>meh</em>, <em>not so good</em>); people write topics and discuss them civilly, without blame, and the action items that come out of the conversation are the first thing the next retro checks on.</li>
  </ul>

  <p>The 8-point cap holds across all of this. Any story bigger than 8 points is an epic and gets split before it can move. agilemarkdown's coach refuses to estimate a feature above the cap; Chapter 6 covers epics directly.</p>

  <h2>Following along</h2>

  <p>The tutorial uses two project states. Chapters 1–2 start from an empty repo so you can see how an agilemarkdown project comes into being. Chapters 3 onward use a seeded fixture in the spirit of Arnold Lobel's <em>Frog and Toad</em>, Toad mailing letters, Frog tending a seedling, Owl planning the moon's birthday. The whimsy is the example, not the tool. (Lobel, <em>Frog and Toad Are Friends</em>, Harper &amp; Row, 1970.)</p>

  <p>If you want to follow along interactively, run the laboratory:</p>

<pre><code>am serve --root /tmp/am-demo --fixture scripts/build-demo.sh
open http://localhost:8080</code></pre>

  <p>That gives you the same web UI, CLI, and MCP console the tutorial captures from. Every action you take maps to a CLI verb in the <strong>CommandLog</strong> tab and a JSON-RPC frame in the <strong>MCP Console</strong> tab.</p>

  <h2>Chapters</h2>
  <div class="chapter-grid">
    {{range .Chapters}}
    <a class="chapter-card" href="tutorial-{{.Num}}-{{.Slug}}.html">
      <div class="num">Chapter {{.Num}}</div>
      <div class="title">{{.Title}}</div>
    </a>
    {{end}}
  </div>

  <h2>How to read each chapter</h2>
  <p>Each moment is a single operation shown three ways. The table reads left-to-right: GUI screenshot, then the CLI verb you would type to do the same thing, then the JSON-RPC frame an MCP client would send.</p>
  <p>The MCP column carries a <em>Full envelope</em> toggle. Open it to see the literal JSON-RPC request and response, same bytes an AI agent exchanges with the project.</p>

  <footer>agilemarkdown · <a href="index.html">home</a> · <a href="reference.html">reference</a></footer>
</main>
</body>
</html>
`))

// indexHead is the same as sharedHead but with a static title.
const indexHead = `<head>
<meta charset="UTF-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>Tutorial · agilemarkdown</title>
<meta name="description" content="agilemarkdown tutorial. Three surfaces for the same workflow: the board (GUI), the am CLI, and MCP JSON-RPC for AI agents." />
<link rel="preconnect" href="https://fonts.googleapis.com" />
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
<link href="https://fonts.googleapis.com/css2?family=Source+Serif+4:opsz,wght@8..60,300;8..60,400;8..60,500;8..60,600;8..60,700&family=JetBrains+Mono:wght@400;500;600&display=swap" rel="stylesheet" />
<style>
  :root {
    --paper:   oklch(0.974 0.009 82);
    --paper-2: oklch(0.945 0.012 82);
    --paper-3: oklch(0.915 0.014 82);
    --ink:     oklch(0.205 0.015 60);
    --ink-2:   oklch(0.36  0.018 60);
    --ink-3:   oklch(0.52  0.018 65);
    --rule:    oklch(0.84  0.014 78);
    --rule-2:  oklch(0.78  0.016 78);
    --accent:  oklch(0.52  0.135 38);
    --accent-soft: oklch(0.85 0.06 38);
    --term-bg: oklch(0.18  0.012 60);
    --term-fg: oklch(0.93  0.012 82);
    --term-dim:oklch(0.62  0.012 82);
  }
  * { box-sizing: border-box; }
  html { scroll-behavior: smooth; }
  body {
    margin: 0; background: var(--paper); color: var(--ink);
    font-family: "Source Serif 4", Georgia, serif;
    font-size: 18px; line-height: 1.65;
    -webkit-font-smoothing: antialiased;
  }
  a { color: var(--accent); }
  code { font-family: "JetBrains Mono", ui-monospace, monospace; font-size: 0.9em; padding: 1px 4px; background: var(--paper-2); border-radius: 3px; }
  main { max-width: 760px; margin: 0 auto; padding: 60px 28px 120px; }
  header.masthead { padding: 80px 0 50px; border-bottom: 1px solid var(--rule); margin-bottom: 60px; }
  header.masthead a.back { font-family: "JetBrains Mono", monospace; font-size: 12px; color: var(--ink-3); text-decoration: none; }
  header.masthead a.back:hover { color: var(--accent); }
  header.masthead .eyebrow {
    font-family: "JetBrains Mono", monospace;
    font-size: 11px; letter-spacing: 0.15em; text-transform: uppercase;
    color: var(--ink-3); margin: 18px 0 14px;
  }
  header.masthead h1 {
    font-size: clamp(36px, 7vw, 60px);
    font-weight: 600; line-height: 1.05;
    letter-spacing: -0.025em; margin: 0 0 18px;
  }
  header.masthead h1 em { font-style: italic; color: var(--accent); font-weight: 500; }
  header.masthead .deck { font-size: clamp(17px, 2.7vw, 21px); color: var(--ink-2); line-height: 1.45; max-width: 620px; }
  h2 {
    font-size: 28px; font-weight: 600; line-height: 1.15;
    letter-spacing: -0.018em; margin: 50px 0 14px;
    border-top: 1px solid var(--rule-2); padding-top: 28px;
  }
  pre { background: var(--term-bg); color: var(--term-fg); padding: 14px 18px; border-radius: 6px; overflow-x: auto; font-family: "JetBrains Mono", monospace; font-size: 13px; }
  pre code { background: transparent; padding: 0; color: inherit; font-size: inherit; }
  p { margin: 0 0 16px; }
  strong { font-weight: 600; }
  ul.rhythm { list-style: none; padding-left: 0; margin: 4px 0 18px; }
  ul.rhythm li { margin: 0 0 10px; padding-left: 0; }
  blockquote.pullquote {
    margin: 18px 0 22px;
    padding: 2px 0 4px 20px;
    border-left: 3px solid var(--accent);
    font-style: italic;
    color: var(--ink-2);
  }
  blockquote.pullquote p { margin: 0 0 8px; line-height: 1.55; font-size: 16.5px; }
  blockquote.pullquote .source {
    font-family: "JetBrains Mono", monospace;
    font-style: normal;
    font-size: 10.5px;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--ink-3);
    margin: 0;
  }
  blockquote.pullquote .source a { color: inherit; text-decoration: underline; text-decoration-color: var(--rule-2); }
  blockquote.pullquote .source a:hover { color: var(--accent); text-decoration-color: var(--accent); }
  .chapter-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
    gap: 16px;
    margin: 30px 0 60px;
  }
  .chapter-card {
    display: block;
    padding: 18px 20px;
    background: var(--paper-2);
    border: 1px solid var(--rule);
    border-radius: 8px;
    text-decoration: none;
    color: var(--ink);
    transition: border-color 120ms, transform 120ms;
  }
  .chapter-card:hover { border-color: var(--accent); transform: translateY(-1px); }
  .chapter-card .num { font-family: "JetBrains Mono", monospace; font-size: 12px; color: var(--accent); letter-spacing: 0.08em; }
  .chapter-card .title { font-size: 20px; font-weight: 600; letter-spacing: -0.012em; margin: 4px 0 0; }
  footer { margin-top: 80px; padding-top: 24px; border-top: 1px solid var(--rule); font-size: 14px; color: var(--ink-3); text-align: center; }
</style>
</head>`
