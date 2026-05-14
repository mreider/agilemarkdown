package actions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/coach"
	"github.com/mreider/agilemarkdown/git"
	"github.com/mreider/agilemarkdown/utils"
)

type CreateBacklogAction struct {
	root        *backlog.BacklogsStructure
	backlogName string
}

func NewCreateBacklogAction(rootDir, backlogName string) *CreateBacklogAction {
	return &CreateBacklogAction{root: backlog.NewBacklogsStructure(rootDir), backlogName: backlogName}
}

func (a *CreateBacklogAction) Execute() error {
	if backlog.IsForbiddenBacklogName(a.backlogName) {
		fmt.Printf("'%s' can't be used as a backlog name\n", a.backlogName)
		return nil
	}

	backlogFileName := utils.GetValidFileName(a.backlogName)
	backlogDir := filepath.Join(a.root.Root(), backlogFileName)
	if info, err := os.Stat(backlogDir); err != nil && !os.IsNotExist(err) {
		return err
	} else if err == nil {
		if info.IsDir() {
			fmt.Println("the backlog directory already exists")
		} else {
			fmt.Println("a file with the same name already exists")
		}
		return nil
	}

	_ = git.SetUpstream()

	err := os.MkdirAll(backlogDir, 0777)
	if err != nil {
		return err
	}

	overviewFileName := fmt.Sprintf("%s.md", backlogFileName)
	overviewPath := filepath.Join(a.root.Root(), overviewFileName)
	overview, err := backlog.LoadBacklogOverview(overviewPath)
	if err != nil {
		return err
	}
	overview.SetTitle(a.backlogName)
	err = overview.UpdateLinks("archive", filepath.Join(backlogDir, backlog.ArchiveFileName), a.root.Root(), a.root.Root())
	if err != nil {
		return err
	}
	overview.SetCreated(utils.GetCurrentTimestamp())
	if err := overview.Save(); err != nil {
		return err
	}

	if err := seedSampleStories(backlogDir, a.backlogName); err != nil {
		return err
	}

	written, err := coach.InstallTemplates(a.root.Root())
	if err != nil {
		return err
	}
	for _, p := range written {
		fmt.Printf("wrote %s\n", p)
	}
	return nil
}

// seedSampleStories writes one feature, one bug, and one chore so a
// brand new backlog has demo content that teaches the Pivotal type
// semantics:
//   - feature counts toward velocity and carries an estimate
//   - bug stays at 0 points by default ("don't reward fixing your own breakage")
//   - chore stays at 0 points ("toil is not progress")
func seedSampleStories(backlogDir, backlogName string) error {
	currentUser := "you"
	if name, _, err := git.CurrentUser(); err == nil && name != "" {
		currentUser = name
	}
	now := utils.GetCurrentTimestamp()

	stories := []struct {
		path    string
		content string
	}{
		{
			path: filepath.Join(backlogDir, "Sample-feature-set-up-login.md"),
			content: stitchItem(map[string]string{
				"title":    "Sample feature: set up login",
				"project":  backlogName,
				"type":     "feature",
				"status":   "unstarted",
				"author":   currentUser,
				"created":  now,
				"modified": now,
				"estimate": "3",
				"tags":     "[onboarding, sample]",
			}, sampleFeatureBody),
		},
		{
			path: filepath.Join(backlogDir, "Sample-bug-typo-on-landing.md"),
			content: stitchItem(map[string]string{
				"title":    "Sample bug: typo on landing page",
				"project":  backlogName,
				"type":     "bug",
				"status":   "unstarted",
				"author":   currentUser,
				"created":  now,
				"modified": now,
				"tags":     "[onboarding, sample]",
			}, sampleBugBody),
		},
		{
			path: filepath.Join(backlogDir, "Sample-chore-rotate-api-key.md"),
			content: stitchItem(map[string]string{
				"title":    "Sample chore: rotate API key",
				"project":  backlogName,
				"type":     "chore",
				"status":   "unstarted",
				"author":   currentUser,
				"created":  now,
				"modified": now,
				"tags":     "[onboarding, sample]",
			}, sampleChoreBody),
		},
	}
	for _, s := range stories {
		if existsFile(s.path) {
			continue
		}
		if err := os.WriteFile(s.path, []byte(s.content), 0644); err != nil {
			return err
		}
	}

	return nil
}

// stitchItem assembles a frontmatter+body string with a stable key order
// so sample files look uniform on disk.
func stitchItem(fields map[string]string, body string) string {
	order := []string{"title", "project", "type", "status", "tags", "assigned", "estimate", "author", "created", "modified"}
	return stitch(order, fields, body)
}

func stitch(order []string, fields map[string]string, body string) string {
	listKeys := map[string]bool{"tags": true}
	var fm string
	fm += "---\n"
	for _, k := range order {
		v, ok := fields[k]
		if !ok || v == "" {
			continue
		}
		if listKeys[k] || strings.HasPrefix(v, "[") {
			// already serialized as a YAML flow sequence
			fm += fmt.Sprintf("%s: %s\n", k, v)
			continue
		}
		fm += fmt.Sprintf("%s: %s\n", k, yamlScalar(v))
	}
	fm += "---\n\n"
	return fm + body
}

// yamlScalar quotes a value when it contains characters that YAML would
// otherwise interpret (`:` is the obvious one). Conservative: always
// double-quote when in doubt.
func yamlScalar(v string) string {
	if strings.ContainsAny(v, ":#&*!|>'\"%@`") || strings.HasPrefix(v, "- ") {
		return `"` + strings.ReplaceAll(v, `"`, `\"`) + `"`
	}
	return v
}

const sampleFeatureBody = `## Why this is a feature

Features are user-visible product changes. Estimated in points. Only features
that reach **accepted** count toward velocity. (Pivotal Tracker convention.)

## Problem statement

Users cannot sign in.

## Possible solution

Email + password form. Session cookie. "Forgot password" link.

## Acceptance

- [ ] visitor can register
- [ ] visitor can sign in and out
- [ ] session survives page reload

## Comments
`

const sampleBugBody = `## Why this is a bug

Bugs are defects in shipped behavior. They are **not estimated by default** in
Pivotal Tracker (and here): "you don't get points for fixing your own breakage."
Flip ` + "`story_types.bug_estimable: true`" + ` in ` + "`.am/config.yaml`" + ` to override.

## Steps to reproduce

1. open landing page
2. read first paragraph
3. observe the typo

## Expected

Correct spelling.

## Actual

The misspelling.

## Comments
`

const sampleChoreBody = `## Why this is a chore

Chores are toil: infrastructure work, audits, dependency bumps. They are
**not estimated** by default ("toil is not progress"). Flip
` + "`story_types.chore_estimable: true`" + ` in ` + "`.am/config.yaml`" + ` to override.

## What to do

- generate a new key
- update the deploy secret
- rotate in production
- revoke the old key

## Comments
`


