package backlog

import (
	"os"
	"path/filepath"
	"testing"
)

// BacklogDirs recognizes any folder with a matching overview file at
// the root, regardless of name. Single-backlog convention says the
// canonical folder is `stories/`, but legacy `am-*` folders and
// arbitrary names still validate as long as the overview is there.
// Folders without an overview are silently skipped (docs/, etc.).

func TestBacklogDirsRequiresOverviewAlongside(t *testing.T) {
	root := t.TempDir()

	// Canonical: `stories/` plus `stories.md`.
	if err := os.MkdirAll(filepath.Join(root, "stories"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "stories.md"), []byte("# stories\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Folder with no overview, ignored.
	if err := os.MkdirAll(filepath.Join(root, "confluence"), 0755); err != nil {
		t.Fatal(err)
	}

	dirs, err := NewBacklogsStructure(root).BacklogDirs()
	if err != nil {
		t.Fatalf("BacklogDirs: %v", err)
	}
	if len(dirs) != 1 {
		t.Fatalf("expected 1 validated backlog, got %d: %v", len(dirs), dirs)
	}
	if filepath.Base(dirs[0]) != "stories" {
		t.Fatalf("expected stories, got %s", dirs[0])
	}
}
