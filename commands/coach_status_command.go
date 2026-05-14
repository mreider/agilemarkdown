package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"
	"github.com/urfave/cli/v3"
)

// CoachStatusCommand prints what the coach knows about the project right
// now. Distinct from `am dashboard` (which is the KPI block) in that
// this surface is coach-specific: pending acceptance, blockers, working
// agreements active, recent learnings.
var CoachStatusCommand = &cli.Command{
	Name:  "coach",
	Usage: "Print the coach's read on the project: pending acceptance, blockers, agreements, recent learnings",
	Action: func(ctx context.Context, c *cli.Command) error {
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		structure := backlog.NewBacklogsStructure(root)
		dirs, err := structure.BacklogDirs()
		if err != nil {
			return err
		}

		var pendingAccept []*backlog.BacklogItem
		var blocked []*backlog.BacklogItem
		var unstartedTop *backlog.BacklogItem
		var unstartedBacklog string
		for _, d := range dirs {
			bck, err := backlog.LoadBacklog(d)
			if err != nil {
				return err
			}
			pri, err := backlog.LoadPriority(d)
			if err != nil {
				return err
			}
			byPath := map[string]*backlog.BacklogItem{}
			for _, it := range bck.ActiveItems() {
				byPath[filepath.Base(it.Path())] = it
			}
			for _, it := range bck.ActiveItems() {
				if strings.EqualFold(it.Status(), backlog.DeliveredStatus.Name) {
					pendingAccept = append(pendingAccept, it)
				}
				if it.Blocked() {
					blocked = append(blocked, it)
				}
			}
			if unstartedTop == nil {
				for _, e := range pri.Entries() {
					it, ok := byPath[e.Path]
					if !ok || it.Blocked() || strings.EqualFold(it.Status(), backlog.AcceptedStatus.Name) {
						continue
					}
					if !strings.EqualFold(it.Status(), backlog.UnstartedStatus.Name) {
						continue
					}
					if it.Type() == "release" {
						continue
					}
					unstartedTop = it
					unstartedBacklog = filepath.Base(d)
					break
				}
			}
		}

		fmt.Println("Coach status")
		fmt.Println()

		// 1. Pending PM acceptance.
		if len(pendingAccept) == 0 {
			fmt.Println("Pending acceptance: none.")
		} else {
			fmt.Printf("Pending acceptance: %d story%s waiting on PM.\n", len(pendingAccept), plural(len(pendingAccept)))
			for _, it := range pendingAccept {
				rel, _ := filepath.Rel(root, it.Path())
				fmt.Printf("  - %s (%s)\n", it.Title(), rel)
				fmt.Printf("    am accept-prompt %s\n", rel)
			}
		}
		fmt.Println()

		// 2. Blockers.
		if len(blocked) == 0 {
			fmt.Println("Blocked: none.")
		} else {
			fmt.Printf("Blocked: %d story%s.\n", len(blocked), plural(len(blocked)))
			for _, it := range blocked {
				rel, _ := filepath.Rel(root, it.Path())
				reason := it.BlockedReason()
				if reason == "" {
					reason = "(no reason recorded)"
				}
				fmt.Printf("  - %s\n    %s\n    %s\n", rel, it.Title(), reason)
			}
		}
		fmt.Println()

		// 3. Iteration fit (rolling velocity vs planned).
		cfg, err := config.LoadConfig(filepath.Join(root, ".am", "config.yaml"))
		if err == nil {
			var feed []*backlog.BacklogItem
			for _, d := range dirs {
				bck, _ := backlog.LoadBacklog(d)
				if bck == nil {
					continue
				}
				for _, it := range bck.AllItems() {
					if backlog.CountsForVelocity(it, cfg) {
						feed = append(feed, it)
					}
				}
			}
			overrides, _ := backlog.LoadIterationOverrides(root)
			velocity, _, boot := backlog.ComputeVelocity(time.Now(), feed, cfg, overrides)
			if boot {
				fmt.Printf("Velocity: %.0f pts (bootstrap)\n", velocity)
			} else {
				fmt.Printf("Velocity: %.0f pts (rolling, %d-iter lookback)\n", velocity, cfg.Velocity.Lookback)
			}
		}

		// 4. Next pull.
		if unstartedTop != nil {
			rel, _ := filepath.Rel(root, unstartedTop.Path())
			fmt.Printf("Next pull (%s): %s — %s\n", unstartedBacklog, unstartedTop.Title(), rel)
		} else {
			fmt.Println("Next pull: nothing unstarted in priority.")
		}
		fmt.Println()

		// 5. Working agreements.
		agreementsPath := filepath.Join(root, "team-agreements.md")
		if data, err := os.ReadFile(agreementsPath); err == nil {
			body := strings.TrimSpace(string(data))
			if body != "" {
				fmt.Println("Working agreements (" + agreementsPath + "):")
				for _, line := range strings.Split(body, "\n") {
					if strings.TrimSpace(line) == "" {
						continue
					}
					fmt.Printf("  %s\n", line)
				}
				fmt.Println()
			}
		} else if !os.IsNotExist(err) {
			return err
		}

		// 6. Recent learnings. Show bullets only; skip headers and prose.
		learningsPath := filepath.Join(root, "learnings.md")
		if data, err := os.ReadFile(learningsPath); err == nil {
			var bullets []string
			for _, line := range strings.Split(string(data), "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
					bullets = append(bullets, trimmed)
				}
			}
			if len(bullets) > 5 {
				bullets = bullets[len(bullets)-5:]
			}
			if len(bullets) > 0 {
				fmt.Println("Recent learnings:")
				for _, b := range bullets {
					fmt.Printf("  %s\n", b)
				}
				fmt.Println()
			}
		}

		// 7. Surface a missing team-agreements.md as a soft prompt. The
		// coach doc says to read it every session; the human cannot
		// produce one without being prompted.
		if _, err := os.Stat(agreementsPath); os.IsNotExist(err) {
			fmt.Println("No team-agreements.md present yet. Bootstrap with:")
			fmt.Println("  am team-agreements --add \"<your first agreement>\"")
			fmt.Println()
		}

		return nil
	},
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
