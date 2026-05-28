package commands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

// TestSetReleaseDate_WritesFrontmatter confirms the CLI verb stamps
// `release_date:` and survives a clear (empty arg).
func TestSetReleaseDate_WritesFrontmatter(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(dir)

	if err := os.MkdirAll(filepath.Join(dir, "stories"), 0755); err != nil {
		t.Fatal(err)
	}
	itemPath := filepath.Join(dir, "stories", "Frost.md")
	body := "---\ntitle: Frost\nstatus: unstarted\ntype: release\n---\n\nbody\n"
	if err := os.WriteFile(itemPath, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	app := &cli.Command{Commands: []*cli.Command{SetReleaseDateCommand}, Writer: &buf, ErrWriter: &buf}
	if err := app.Run(context.Background(), []string{"am", "set-release-date", "stories/Frost.md", "2026-07-01"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	out, _ := os.ReadFile(itemPath)
	if !strings.Contains(string(out), "release_date:") {
		t.Fatalf("expected release_date stamped; got:\n%s", out)
	}
	if !strings.Contains(string(out), "2026-07-01") {
		t.Fatalf("expected target date in body; got:\n%s", out)
	}

	// Clearing with an empty arg leaves the file but with an empty
	// release_date, useful when a marker is mistakenly typed.
	if err := app.Run(context.Background(), []string{"am", "set-release-date", "stories/Frost.md", ""}); err != nil {
		t.Fatalf("run clear: %v", err)
	}
	out, _ = os.ReadFile(itemPath)
	if strings.Contains(string(out), "2026-07-01") {
		t.Fatalf("expected date cleared; still present:\n%s", out)
	}
}

// TestSetReleaseDate_RefusesMissingPath surfaces a clear error when
// the file isn't under any backlog.
func TestSetReleaseDate_RefusesMissingPath(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(dir)

	var buf bytes.Buffer
	app := &cli.Command{Commands: []*cli.Command{SetReleaseDateCommand}, Writer: &buf, ErrWriter: &buf}
	err := app.Run(context.Background(), []string{"am", "set-release-date", "stories/Ghost.md", "2026-07-01"})
	if err == nil {
		t.Fatalf("expected error for missing file")
	}
}
