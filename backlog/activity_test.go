package backlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// LoadActivity on a repo with no log yet is not an error: callers
// (am log) render "(no activity recorded yet)" rather than crashing.
func TestLoadActivityMissingIsEmpty(t *testing.T) {
	dir := t.TempDir()
	entries, err := LoadActivity(dir)
	if err != nil {
		t.Fatalf("expected nil error for missing log, got %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no entries, got %d", len(entries))
	}
}

// Append then load round-trips fields, preserves chronological (append)
// order, and creates the .am directory on demand.
func TestAppendAndLoadActivity(t *testing.T) {
	dir := t.TempDir()
	if err := AppendActivity(dir, "Alice", "created", "stories/Foo.md", "target=priority position=top"); err != nil {
		t.Fatal(err)
	}
	if err := AppendActivity(dir, "Alice", "started", "stories/Foo.md", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".am", "activity.log")); err != nil {
		t.Fatalf("expected .am/activity.log to be created: %v", err)
	}
	entries, err := LoadActivity(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Verb != "created" || entries[1].Verb != "started" {
		t.Fatalf("expected chronological order created,started; got %s,%s", entries[0].Verb, entries[1].Verb)
	}
	if entries[0].Actor != "Alice" || entries[0].Item != "stories/Foo.md" {
		t.Fatalf("fields not round-tripped: %+v", entries[0])
	}
	if entries[0].Detail != "target=priority position=top" {
		t.Fatalf("detail not round-tripped: %q", entries[0].Detail)
	}
	if entries[0].Time.IsZero() {
		t.Fatal("timestamp not parsed")
	}
}

// A blank actor is normalized to "unknown" so every line attributes to
// something greppable.
func TestAppendActivityBlankActor(t *testing.T) {
	dir := t.TempDir()
	if err := AppendActivity(dir, "", "started", "x.md", ""); err != nil {
		t.Fatal(err)
	}
	entries, _ := LoadActivity(dir)
	if len(entries) != 1 || entries[0].Actor != "unknown" {
		t.Fatalf("expected actor 'unknown', got %+v", entries)
	}
}

// Tabs and newlines in a field (e.g. a multi-line rejection reason)
// must not break the one-line-per-event format.
func TestAppendActivitySanitizesSeparators(t *testing.T) {
	dir := t.TempDir()
	reason := "broke\twith\na\rnasty reason"
	if err := AppendActivity(dir, "Bob", "rejected", "x.md", reason); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".am", "activity.log"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "\n") != 1 {
		t.Fatalf("expected exactly one line, got %q", data)
	}
	entries, _ := LoadActivity(dir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if strings.ContainsAny(entries[0].Detail, "\t\r\n") {
		t.Fatalf("separators leaked into detail: %q", entries[0].Detail)
	}
}

// Malformed lines (hand-edits, truncation) are skipped, never fatal.
func TestLoadActivitySkipsMalformed(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".am"), 0755); err != nil {
		t.Fatal(err)
	}
	content := "garbage line without tabs\n" +
		"also\tonly\ttwo\n" + // missing a parseable timestamp in field 0
		"2026-06-04T10:00:00Z\tAlice\tstarted\tstories/Foo.md\t\n"
	if err := os.WriteFile(filepath.Join(dir, ".am", "activity.log"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	entries, err := LoadActivity(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Verb != "started" {
		t.Fatalf("expected only the one valid entry, got %+v", entries)
	}
}
