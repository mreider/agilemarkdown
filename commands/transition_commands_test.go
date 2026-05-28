package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// requireItemFile is the guard the transition verbs (start, finish,
// deliver, accept, reject) all share. Without it, an `am start
// Login-Flow.md` run from the repo root would silently create a
// phantom file at <root>/Login-Flow.md because markdown.LoadFrontmatter
// returns an empty in-memory file for missing paths. These tests
// exercise the guard directly so a regression shows up before the CLI
// is wired into the rest of the extension.

func TestRequireItemFileRejectsMissingPath(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "Login-Flow.md")
	_, err := requireItemFile(missing)
	if err == nil {
		t.Fatalf("expected an error for missing file %q, got nil", missing)
	}
	if !strings.Contains(err.Error(), "no such item") {
		t.Fatalf("error %q did not mention 'no such item'", err)
	}
	// Critically: the guard must not have created the file as a side
	// effect of the stat. Drift here would put us right back where the
	// original phantom-file bug lived.
	if _, statErr := os.Stat(missing); !os.IsNotExist(statErr) {
		t.Fatalf("requireItemFile should not have created %q (stat err: %v)", missing, statErr)
	}
}

func TestRequireItemFileAppendsMdSuffix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Login-Flow.md")
	if err := os.WriteFile(path, []byte("---\ntitle: x\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// User-typed argument without the .md extension. The CLI should
	// resolve it the same way as the full filename.
	got, err := requireItemFile(filepath.Join(dir, "Login-Flow"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != path {
		t.Fatalf("expected %q, got %q", path, got)
	}
}

func TestRequireItemFileAcceptsRelativeArg(t *testing.T) {
	// The CLI is normally invoked from the repo root with arguments
	// like `product/Login-Flow.md`. filepath.Abs resolves that against
	// cwd, so the guard works against the absolute form. Switch cwd
	// into a tmpdir to keep the assertion self-contained.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "product"), 0755); err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(dir, "product", "Login-Flow.md")
	if err := os.WriteFile(abs, []byte("---\ntitle: x\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	got, err := requireItemFile("product/Login-Flow.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// On macOS the tmpdir resolves through /private/var/..., so
	// compare via EvalSymlinks rather than literal strings.
	wantResolved, _ := filepath.EvalSymlinks(abs)
	gotResolved, _ := filepath.EvalSymlinks(got)
	if wantResolved != gotResolved {
		t.Fatalf("expected resolved path %q, got %q", wantResolved, gotResolved)
	}
}
