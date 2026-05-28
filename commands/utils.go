package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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

func checkIsRootDirectory(dir string) error {
	if !git.IsRootGitDirectory(dir) {
		return errors.New("Error, please change directory to a root git folder")
	}
	return nil
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
