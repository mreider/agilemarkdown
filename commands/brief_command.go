package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/urfave/cli/v3"
)

// BriefCommand emits a single agent-onboarding context blob: the project
// frame (a condensed inception), the dashboard KPIs, the next pull, work
// in progress, and the working agreements. The point is that a fresh
// agent — or a SessionStart hook — gets the whole picture in one command:
// the "why/what" (vision) alongside the "what's next" (backlog state),
// without stitching `am inception`, `am dashboard`, and `am next`
// together by hand. All output is plain markdown so it drops straight
// into a chat or a hook's context block.
var BriefCommand = &cli.Command{
	Name:  "brief",
	Usage: "One-shot agent onboarding: project vision + dashboard + next pull + WIP + agreements",
	Action: func(ctx context.Context, c *cli.Command) error {
		root, err := findRootDirectory()
		if err != nil {
			return err
		}

		var b strings.Builder
		fmt.Fprintf(&b, "# %s — agent brief\n\n", filepath.Base(root))

		if proj := inceptionBrief(root); proj != "" {
			b.WriteString("## Project\n")
			b.WriteString(proj)
			b.WriteString("\n\n_Full vision (success, constraints, out-of-scope): `am inception --show`._\n\n")
		}

		dash, err := dashboardText(root)
		if err != nil {
			return err
		}
		b.WriteString("## Dashboard\n")
		b.WriteString(strings.TrimPrefix(dash, "Dashboard\n\n"))
		b.WriteString("\n")

		nxt, err := nextText(root)
		if err != nil {
			return err
		}
		b.WriteString("## Next pull\n")
		b.WriteString(nxt)
		b.WriteString("\n")

		b.WriteString("## In progress\n")
		b.WriteString(wipText(root))
		b.WriteString("\n")

		if agr := agreementsBrief(root); agr != "" {
			b.WriteString("## Working agreements\n")
			b.WriteString(agr)
			b.WriteString("\n_Full text: `am team-agreements`._\n")
		}

		fmt.Print(b.String())
		return nil
	},
}

// wipText lists every story currently in `started` across all backlogs,
// or a nudge when none are. This is the WIP the dev pair is mid-flight on.
func wipText(root string) string {
	dirs, err := backlog.NewBacklogsStructure(root).BacklogDirs()
	if err != nil {
		return "  (could not read backlog)\n"
	}
	var lines []string
	for _, d := range dirs {
		bck, err := backlog.LoadBacklog(d)
		if err != nil {
			continue
		}
		for _, it := range bck.AllItems() {
			if strings.EqualFold(it.Status(), backlog.StartedStatus.Name) {
				rel, _ := filepath.Rel(root, it.Path())
				lines = append(lines, fmt.Sprintf("  - %s  (%s)", it.Title(), rel))
			}
		}
	}
	if len(lines) == 0 {
		return "  (none — pull a story before editing code)\n"
	}
	return strings.Join(lines, "\n") + "\n"
}

// inceptionBrief pulls the two most orienting sections — "The user" and
// "The goal" — out of .am/inception.md, in document order. The rest of
// the vision (success, constraints, out-of-scope) stays one command away
// (`am inception --show`) rather than bloating every session start.
// Returns "" when there is no inception document yet.
func inceptionBrief(root string) string {
	data, err := os.ReadFile(filepath.Join(root, inceptionFileName))
	if err != nil {
		return ""
	}
	want := map[string]bool{"the user": true, "the goal": true}
	var out []string
	capturing := false
	for _, ln := range strings.Split(string(data), "\n") {
		switch {
		case strings.HasPrefix(ln, "## "):
			title := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(ln, "## ")))
			capturing = want[title]
			if capturing {
				if len(out) > 0 {
					out = append(out, "")
				}
				out = append(out, "### "+strings.TrimSpace(strings.TrimPrefix(ln, "## ")))
			}
		case strings.HasPrefix(ln, "# "):
			capturing = false
		default:
			if capturing {
				out = append(out, ln)
			}
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// agreementsBrief returns the bullet lines from .am/team-agreements.md
// (capped) so the brief carries the soft rules without reproducing the
// whole document. Returns "" when there is no agreements file.
func agreementsBrief(root string) string {
	data, err := os.ReadFile(filepath.Join(root, agreementsFileName))
	if err != nil {
		return ""
	}
	const max = 12
	var lines []string
	for _, ln := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") {
			lines = append(lines, "  "+t)
			if len(lines) >= max {
				lines = append(lines, "  - …")
				break
			}
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}
