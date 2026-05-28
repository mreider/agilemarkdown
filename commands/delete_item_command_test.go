package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

// Lock in the contract of `am delete-item`:
//   - removes the story file from disk
//   - removes the entry from _priority.md and _icebox.md
//   - refuses a missing path via the requireItemFile guard
//
// Without these guarantees a wrong-cwd `am delete-item Foo.md` could
// silently no-op, or the order files could drift out of sync with the
// disk state (a stale `[Foo](Foo.md)` link pointing at a deleted file).

func TestDeleteItemRemovesFileAndOrderEntries(t *testing.T) {
	dir := t.TempDir()
	storyPath := filepath.Join(dir, "Foo.md")
	if err := os.WriteFile(storyPath, []byte("---\ntitle: Foo\nstatus: unstarted\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Seed an _icebox.md that references the story; the verb should
	// drop the entry, not just delete the .md file.
	icePath := filepath.Join(dir, "_icebox.md")
	if err := os.WriteFile(icePath, []byte("# Icebox\n\n- [Foo](Foo.md)\n"), 0644); err != nil {
		t.Fatal(err)
	}

	prev, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	app := &cli.Command{Commands: []*cli.Command{DeleteItemCommand}}
	if err := app.Run(context.Background(), []string{"am", "delete-item", "Foo.md"}); err != nil {
		t.Fatalf("delete-item failed: %v", err)
	}

	if _, err := os.Stat(storyPath); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be removed, stat err: %v", storyPath, err)
	}
	body, err := os.ReadFile(icePath)
	if err != nil {
		t.Fatalf("read _icebox.md: %v", err)
	}
	if strings.Contains(string(body), "Foo.md") {
		t.Fatalf("expected Foo.md entry to be removed from _icebox.md, got:\n%s", body)
	}
}

func TestDeleteItemRefusesMissingPath(t *testing.T) {
	dir := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	app := &cli.Command{Commands: []*cli.Command{DeleteItemCommand}}
	err := app.Run(context.Background(), []string{"am", "delete-item", "Ghost.md"})
	if err == nil {
		t.Fatalf("expected delete-item to refuse a missing path, got nil")
	}
	if !strings.Contains(err.Error(), "no such item") {
		t.Fatalf("error %q did not mention 'no such item'", err)
	}
	// No phantom side-effect: the guard must not create the file.
	if _, err := os.Stat(filepath.Join(dir, "Ghost.md")); !os.IsNotExist(err) {
		t.Fatalf("expected Ghost.md not to exist, stat err: %v", err)
	}
}
