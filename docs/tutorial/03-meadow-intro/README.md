# 3. The seeded backlog

The first two chapters used an empty repo so you could see how an
agilemarkdown project comes into being. From here on we switch to
a seeded fixture so the examples have realistic depth: multiple
epics, six iterations of history, release markers, an iteration
override that scaled the team's strength down for one week.

The fixture is built by `scripts/build-demo.sh`, which seeds 54
stories across five epics in the meadow the tutorial prologue
introduced. This is the fixture every subsequent chapter operates
on; you can reproduce it locally:

```
bash scripts/build-demo.sh /tmp/am-demo
am serve --root /tmp/am-demo --port 8080
open http://localhost:8080
```

Next: [The current sprint →](../04-current-sprint/)
