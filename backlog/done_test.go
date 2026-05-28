package backlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mreider/agilemarkdown/config"
)

// writeStoryWithAccepted writes a minimal story file under dir with
// the given status and (optional) RFC3339 accepted timestamp.
func writeStoryWithAccepted(t *testing.T, dir, name, status, accepted string) *BacklogItem {
	t.Helper()
	body := "---\ntitle: " + name + "\nstatus: " + status + "\n"
	if accepted != "" {
		body += "accepted: " + accepted + "\n"
	}
	body += "---\n\nbody\n"
	path := filepath.Join(dir, name+".md")
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	item, err := LoadBacklogItem(path)
	if err != nil {
		t.Fatal(err)
	}
	return item
}

func TestIsDone_AcceptedPriorIteration(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Defaults()
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	priorAccepted := now.AddDate(0, 0, -7).Format(time.RFC3339)
	item := writeStoryWithAccepted(t, dir, "old", "accepted", priorAccepted)
	if !IsDone(item, cfg, now) {
		t.Fatalf("accepted one week ago must be Done")
	}
}

func TestIsDone_AcceptedInCurrentIterationIsNotDone(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Defaults()
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	current := now.AddDate(0, 0, -1).Format(time.RFC3339)
	item := writeStoryWithAccepted(t, dir, "now", "accepted", current)
	if IsDone(item, cfg, now) {
		t.Fatalf("accepted yesterday (same iteration) must NOT be Done")
	}
	if !AcceptedInCurrentIteration(item, cfg, now) {
		t.Fatalf("expected AcceptedInCurrentIteration true")
	}
}

func TestIsDone_NotAcceptedNeverDone(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Defaults()
	now := time.Now()
	for _, st := range []string{"unstarted", "started", "finished", "delivered", "rejected"} {
		item := writeStoryWithAccepted(t, dir, "x-"+st, st, "")
		if IsDone(item, cfg, now) {
			t.Fatalf("%s must not be Done", st)
		}
		if AcceptedInCurrentIteration(item, cfg, now) {
			t.Fatalf("%s must not be accepted-current", st)
		}
	}
}

func TestIsDone_AcceptedWithoutTimestampIsNotDone(t *testing.T) {
	// A story whose status says accepted but with no accepted:
	// timestamp shouldn't silently disappear into Done. Leave it in
	// the active view so the PM can fix the bad data.
	dir := t.TempDir()
	cfg := config.Defaults()
	item := writeStoryWithAccepted(t, dir, "bad", "accepted", "")
	if IsDone(item, cfg, time.Now()) {
		t.Fatalf("accepted without timestamp must NOT be Done")
	}
}

func TestDoneASCII_GroupsByIteration(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Defaults()
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	a := now.AddDate(0, 0, -7).Format(time.RFC3339)
	b := now.AddDate(0, 0, -8).Format(time.RFC3339)
	c := now.AddDate(0, 0, -14).Format(time.RFC3339)
	writeStoryWithAccepted(t, dir, "A", "accepted", a)
	writeStoryWithAccepted(t, dir, "B", "accepted", b)
	writeStoryWithAccepted(t, dir, "C", "accepted", c)
	// One current-iteration accepted story to confirm it does not
	// bleed into Done.
	writeStoryWithAccepted(t, dir, "D", "accepted", now.Format(time.RFC3339))

	bck, err := LoadBacklog(dir)
	if err != nil {
		t.Fatal(err)
	}
	out, err := DoneASCII(bck, dir, cfg, now)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Done") {
		t.Fatalf("expected Done header:\n%s", out)
	}
	for _, want := range []string{"A", "B", "C"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in Done:\n%s", want, out)
		}
	}
	if strings.Contains(out, "\n   1. ★  D") || strings.Contains(out, "\n   2. ★  D") {
		t.Fatalf("current-iteration story D leaked into Done:\n%s", out)
	}
	if strings.Count(out, "── Iteration") < 2 {
		t.Fatalf("expected at least two iteration bands in Done:\n%s", out)
	}
}

func TestDoneASCII_EmptyCase(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Defaults()
	bck, err := LoadBacklog(dir)
	if err != nil {
		t.Fatal(err)
	}
	out, err := DoneASCII(bck, dir, cfg, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "no accepted stories from prior iterations") {
		t.Fatalf("expected empty-state message:\n%s", out)
	}
}
