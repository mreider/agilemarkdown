package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

// Locks in the contract of `am set-type`:
//   - writes the new type to frontmatter
//   - accepts feature/bug/chore/release
//   - rejects anything else, with a clear error
//   - refuses missing paths via requireItemFile

func TestSetTypeWritesFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Foo.md")
	if err := os.WriteFile(path, []byte("---\ntitle: Foo\ntype: feature\nstatus: unstarted\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}
	prev, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	app := &cli.Command{Commands: []*cli.Command{SetTypeCommand}}
	if err := app.Run(context.Background(), []string{"am", "set-type", "Foo.md", "bug"}); err != nil {
		t.Fatalf("set-type bug failed: %v", err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "type: bug") {
		t.Fatalf("expected type: bug in frontmatter, got:\n%s", body)
	}
}

func TestSetTypeRejectsInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Foo.md")
	if err := os.WriteFile(path, []byte("---\ntitle: Foo\ntype: feature\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}
	prev, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(prev) })

	app := &cli.Command{Commands: []*cli.Command{SetTypeCommand}}
	err := app.Run(context.Background(), []string{"am", "set-type", "Foo.md", "epic"})
	if err == nil {
		t.Fatalf("expected error for invalid type, got nil")
	}
	if !strings.Contains(err.Error(), "invalid type") {
		t.Fatalf("error %q did not mention 'invalid type'", err)
	}
}

func TestSetTypeRefusesMissingPath(t *testing.T) {
	dir := t.TempDir()
	prev, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(prev) })

	app := &cli.Command{Commands: []*cli.Command{SetTypeCommand}}
	err := app.Run(context.Background(), []string{"am", "set-type", "Ghost.md", "bug"})
	if err == nil {
		t.Fatalf("expected refusal for missing path, got nil")
	}
	if !strings.Contains(err.Error(), "no such item") {
		t.Fatalf("error %q did not mention 'no such item'", err)
	}
}
