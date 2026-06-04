package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

// realpath collapses symlinks so comparisons survive macOS where
// t.TempDir() lives under /var -> /private/var.
func realpath(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", p, err)
	}
	return r
}

// chdir switches cwd for the duration of the test.
func chdir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}

// canonicalProject lays down a git root with one canonical backlog
// (stories/ + stories.md alongside) and returns (root, storiesDir).
func canonicalProject(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	stories := filepath.Join(root, "stories")
	if err := os.MkdirAll(stories, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "stories.md"), []byte("# stories\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return root, stories
}

// The read verbs (show/velocity/sprint plan/cycle-time/...) used to
// demand cwd *be* the backlog folder while dashboard/next ran from the
// project root — same repo, contradictory ergonomics. resolveBacklogDir
// closes that gap: from the project root it discovers the single
// canonical backlog, and from inside the backlog folder cwd still wins.
func TestResolveBacklogDirFromProjectRoot(t *testing.T) {
	root, stories := canonicalProject(t)

	chdir(t, root)
	got, err := resolveBacklogDir()
	if err != nil {
		t.Fatalf("resolveBacklogDir from project root: %v", err)
	}
	if realpath(t, got) != realpath(t, stories) {
		t.Fatalf("from root: got %q, want %q", got, stories)
	}
}

func TestResolveBacklogDirFromInsideBacklog(t *testing.T) {
	_, stories := canonicalProject(t)

	chdir(t, stories)
	got, err := resolveBacklogDir()
	if err != nil {
		t.Fatalf("resolveBacklogDir from inside backlog: %v", err)
	}
	if realpath(t, got) != realpath(t, stories) {
		t.Fatalf("from backlog dir: got %q, want %q", got, stories)
	}
}

// With more than one backlog and an ambiguous cwd, resolveBacklogDir
// must refuse and name the candidates rather than silently picking one.
func TestResolveBacklogDirAmbiguousErrors(t *testing.T) {
	root, _ := canonicalProject(t)
	other := filepath.Join(root, "spikes")
	if err := os.MkdirAll(other, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "spikes.md"), []byte("# spikes\n"), 0644); err != nil {
		t.Fatal(err)
	}

	chdir(t, root)
	_, err := resolveBacklogDir()
	if err == nil {
		t.Fatal("expected ambiguity error with two backlogs at root; got nil")
	}
	for _, name := range []string{"stories", "spikes"} {
		if !strings.Contains(err.Error(), name) {
			t.Fatalf("ambiguity error %q should name %q", err, name)
		}
	}
}

// End-to-end through the cli wiring: `am show priority` from the project
// root must render instead of erroring with "backlog folder".
func TestShowPriorityRunsFromProjectRoot(t *testing.T) {
	root, _ := canonicalProject(t)
	chdir(t, root)

	app := &cli.Command{Commands: []*cli.Command{ShowCommand}}
	stdout, stderr := captureStdio(t)
	defer stdout.restore()
	defer stderr.restore()

	if err := app.Run(context.Background(), []string{"am", "show", "priority"}); err != nil {
		t.Fatalf("am show priority from project root: %v", err)
	}
}
