package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"
	"github.com/mreider/agilemarkdown/git"
)

func checkIsBacklogDirectory() error {
	_, ok := backlog.FindOverviewFileInRootDirectory(".")
	if !ok {
		return errors.New("Error, please change directory to a backlog folder")
	}
	return nil
}

// resolveBacklogDir returns the absolute path of the backlog directory a
// per-backlog view verb should operate on. It honors the historical
// contract first: if the current directory is itself a backlog folder,
// that wins. Otherwise it falls back to discovering backlogs from the
// project (git) root the same way `dashboard`/`next`/`pull` already do,
// so read verbs run from the repo root too. With the canonical
// one-project-one-backlog layout this resolves unambiguously; when a repo
// has several backlogs and cwd isn't one of them, it names them and asks
// the caller to cd into one rather than guessing.
func resolveBacklogDir() (string, error) {
	if _, ok := backlog.FindOverviewFileInRootDirectory("."); ok {
		return filepath.Abs(".")
	}
	root, err := findRootDirectory()
	if err != nil {
		return "", errors.New("Error, please change directory to a backlog folder")
	}
	dirs, err := backlog.NewBacklogsStructure(root).BacklogDirs()
	if err != nil {
		return "", err
	}
	switch len(dirs) {
	case 0:
		return "", errors.New("Error, please change directory to a backlog folder")
	case 1:
		return filepath.Abs(dirs[0])
	default:
		names := make([]string, len(dirs))
		for i, d := range dirs {
			names[i] = filepath.Base(d)
		}
		return "", fmt.Errorf("multiple backlogs found (%s); cd into one of them", strings.Join(names, ", "))
	}
}

func checkIsRootDirectory(dir string) error {
	if !git.IsRootGitDirectory(dir) {
		return errors.New("Error, please change directory to a root git folder")
	}
	return nil
}

// currentActor returns the git user.name for stamping the audit trail,
// falling back to "unknown" when git has no identity configured.
func currentActor() string {
	name, _, err := git.CurrentUser()
	if err != nil || name == "" {
		return "unknown"
	}
	return name
}

// recordActivity appends one semantic line to the audit trail
// (`.am/activity.log`). It is best-effort: a failure to write the trail
// must never fail the command the user actually asked for, so every
// error here is swallowed. itemPath may be absolute, relative, or empty
// (for repo-wide events); it is normalized to a path relative to the
// repo root for stable, location-independent log lines.
func recordActivity(verb, itemPath, detail string) {
	root, err := findRootDirectory()
	if err != nil || root == "" {
		return
	}
	item := ""
	if itemPath != "" {
		if abs, aerr := filepath.Abs(itemPath); aerr == nil {
			if rel, rerr := filepath.Rel(root, abs); rerr == nil {
				item = rel
			} else {
				item = filepath.Base(itemPath)
			}
		} else {
			item = filepath.Base(itemPath)
		}
	}
	_ = backlog.AppendActivity(root, currentActor(), verb, item, detail)
}

func findRootDirectory() (string, error) {
	dir, _ := filepath.Abs(".")
	for dir != "" {
		_, err := os.Stat(filepath.Join(dir, ".git"))
		if err == nil {
			break
		}
		newDir := filepath.Dir(dir)
		if newDir == dir {
			dir = ""
		} else {
			dir = newDir
		}
	}
	if dir == "" {
		return "", fmt.Errorf("can't find root directory from '%s'", dir)
	}
	return dir, nil
}

// AddConfigAndGitIgnore creates `.am/config.yaml` with sensible defaults
// when missing, and writes a baseline `.gitignore`. Both are committed
// in one shot so a fresh repo starts clean.
func AddConfigAndGitIgnore(root *backlog.BacklogsStructure) error {
	hasChanges := false

	configPath := root.ConfigFile()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := config.Defaults().Save(configPath); err != nil {
			return err
		}
		if err := git.Add(configPath); err != nil {
			return err
		}
		hasChanges = true
	}
	gitIgnorePath := filepath.Join(root.Root(), ".gitignore")
	if _, err := os.Stat(gitIgnorePath); os.IsNotExist(err) {
		// Tracked files only. Generated indexes (index.md, velocity.md,
		// timeline.md, users.md, tags/) stay tracked so a clone renders
		// on GitHub without an `am sync`. Local build/binary outputs are
		// ignored.
		gitignore := "agilemarkdown\nagilemarkdown_bash_autocomplete\n"
		if err := os.WriteFile(gitIgnorePath, []byte(gitignore), 0644); err != nil {
			return err
		}
		if err := git.Add(gitIgnorePath); err != nil {
			return err
		}
		hasChanges = true
	}

	if hasChanges {
		if err := git.Commit("agilemarkdown init", ""); err != nil {
			return err
		}
	}
	return nil
}
