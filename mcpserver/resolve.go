package mcpserver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
)

// resolveItemPath resolves a tool-supplied item path to an EXISTING
// story file under the project root, returning its absolute path and
// its root-relative path.
//
// The guard this provides: markdown.LoadFrontmatter returns an empty
// in-memory file for a path that doesn't exist, so a handler that
// joined a bad path and then saved would silently create a
// frontmatter husk wherever the path landed — typically the repo
// root, when the caller omitted the backlog folder prefix. Every
// item-path tool goes through here instead, so a wrong path is an
// error, never a new file.
//
// Resolution order:
//  1. root/<path> exists → use it.
//  2. A bare file name (no folder) is searched across every backlog
//     dir: a unique match resolves (so callers may omit the backlog
//     prefix), an ambiguous one errors and names the cure.
//  3. Anything else → "item not found".
func resolveItemPath(root *backlog.BacklogsStructure, argPath string) (abs, rel string, err error) {
	p := strings.TrimSpace(argPath)
	if p == "" {
		return "", "", fmt.Errorf("item path required")
	}
	cand := filepath.Join(root.Root(), p)
	if isExistingFile(cand) {
		rel, rerr := filepath.Rel(root.Root(), cand)
		if rerr != nil {
			rel = p
		}
		return cand, rel, nil
	}
	if !strings.ContainsAny(p, `/\`) {
		dirs, derr := root.BacklogDirs()
		if derr != nil {
			return "", "", derr
		}
		matches := make([]string, 0, 1)
		for _, d := range dirs {
			c := filepath.Join(d, p)
			if isExistingFile(c) {
				matches = append(matches, c)
			}
		}
		if len(matches) == 1 {
			rel, rerr := filepath.Rel(root.Root(), matches[0])
			if rerr != nil {
				rel = p
			}
			return matches[0], rel, nil
		}
		if len(matches) > 1 {
			return "", "", fmt.Errorf("item %q exists in more than one backlog; use <backlog>/<file>", p)
		}
	}
	return "", "", fmt.Errorf("item not found: %s", p)
}

// loadExistingItem resolves argPath (see resolveItemPath) and loads
// the story it points at. The returned rel path is what handlers
// should use for activity records and rank operations — args.Path may
// have been a bare name.
func loadExistingItem(root *backlog.BacklogsStructure, argPath string) (*backlog.BacklogItem, string, error) {
	abs, rel, err := resolveItemPath(root, argPath)
	if err != nil {
		return nil, "", err
	}
	item, err := backlog.LoadBacklogItem(abs)
	if err != nil {
		return nil, "", err
	}
	return item, rel, nil
}

func isExistingFile(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
