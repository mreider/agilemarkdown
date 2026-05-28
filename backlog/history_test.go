package backlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mreider/agilemarkdown/config"
)

func makeItem(t *testing.T, dir, name string, frontmatter map[string]string) *BacklogItem {
	t.Helper()
	path := filepath.Join(dir, name+".md")
	body := "---\n"
	for k, v := range frontmatter {
		body += k + ": " + v + "\n"
	}
	body += "type: feature\n---\n\nbody.\n"
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	it, err := LoadBacklogItem(path)
	if err != nil {
		t.Fatal(err)
	}
	return it
}

func TestCFDRowsThreeBands(t *testing.T) {
	dir := t.TempDir()
	day := func(s string) time.Time {
		v, _ := time.Parse(time.RFC3339, s+"T00:00:00Z")
		return v
	}
	// Three stories:
	//   a) accepted on D-1
	//   b) started on D-3, still in flight on D-1
	//   c) created on D-3, never started: backlog on D-1
	makeItem(t, dir, "a", map[string]string{
		"title": "a", "status": "accepted",
		"created":  "2026-05-01T09:00:00Z",
		"started":  "2026-05-02T09:00:00Z",
		"finished": "2026-05-05T09:00:00Z",
		"accepted": "2026-05-05T15:00:00Z",
	})
	makeItem(t, dir, "b", map[string]string{
		"title": "b", "status": "started",
		"created": "2026-05-01T09:00:00Z",
		"started": "2026-05-02T09:00:00Z",
	})
	makeItem(t, dir, "c", map[string]string{
		"title": "c", "status": "unstarted",
		"created": "2026-05-01T09:00:00Z",
	})

	items := make([]*BacklogItem, 0)
	for _, name := range []string{"a", "b", "c"} {
		it, _ := LoadBacklogItem(filepath.Join(dir, name+".md"))
		items = append(items, it)
	}

	rows := CFDRows(items, day("2026-05-01"), day("2026-05-07"))
	// On day 2026-05-06 (the last row) we expect accepted=1, in_flight=1, backlog=1
	last := rows[len(rows)-1]
	if last.Accepted != 1 || last.InFlight != 1 || last.Backlog != 1 {
		t.Errorf("final-day CFD wrong: accepted=%d in_flight=%d backlog=%d (rows: %+v)",
			last.Accepted, last.InFlight, last.Backlog, rows)
	}
	// On day 2026-05-01 nothing has started or accepted yet (started/accepted are after that day)
	// so all three sit in backlog.
	first := rows[0]
	if first.Backlog != 3 {
		t.Errorf("day-1 backlog wrong: backlog=%d (row: %+v)", first.Backlog, first)
	}
}

func TestItemIterationFromAccepted(t *testing.T) {
	c := config.Defaults()
	dir := t.TempDir()
	it := makeItem(t, dir, "x", map[string]string{
		"title": "x", "status": "accepted",
		"accepted": "2026-05-06T15:00:00Z",
	})
	num, label := ItemIteration(it, c, 0, 0)
	if num <= 0 {
		t.Fatalf("expected iteration number > 0, got %d", num)
	}
	if label != "" {
		t.Fatalf("expected empty label for accepted item, got %q", label)
	}
	// Same date -> same iteration as IterationNumberFor returns.
	want := IterationNumberFor(it.Accepted(), c)
	if num != want {
		t.Fatalf("iteration mismatch: got %d want %d", num, want)
	}
}

func TestCFDASCIIAccurate(t *testing.T) {
	// Same three items as TestCFDRowsThreeBands; the ASCII output
	// must surface the exact counts on the final day so a reader can
	// verify accuracy by eye.
	dir := t.TempDir()
	day := func(s string) time.Time {
		v, _ := time.Parse(time.RFC3339, s+"T00:00:00Z")
		return v
	}
	makeItem(t, dir, "a", map[string]string{
		"title": "a", "status": "accepted",
		"created": "2026-05-01T09:00:00Z", "started": "2026-05-02T09:00:00Z",
		"finished": "2026-05-05T09:00:00Z", "accepted": "2026-05-05T15:00:00Z",
	})
	makeItem(t, dir, "b", map[string]string{
		"title": "b", "status": "started",
		"created": "2026-05-01T09:00:00Z", "started": "2026-05-02T09:00:00Z",
	})
	makeItem(t, dir, "c", map[string]string{
		"title": "c", "status": "unstarted",
		"created": "2026-05-01T09:00:00Z",
	})
	items := make([]*BacklogItem, 0)
	for _, name := range []string{"a", "b", "c"} {
		it, _ := LoadBacklogItem(filepath.Join(dir, name+".md"))
		items = append(items, it)
	}
	rows := CFDRows(items, day("2026-05-01"), day("2026-05-07"))
	out := CFDASCII(rows)
	// Final-day counts: 1 accepted, 1 in-flight, 1 backlog.
	want := "2026-05-06    1     1     1"
	if !strings.Contains(out, want) {
		t.Fatalf("CFDASCII missing final-day row %q:\n%s", want, out)
	}
	// Day 0 should still show all three in backlog (none started yet).
	want = "2026-05-01    0     0     3"
	if !strings.Contains(out, want) {
		t.Fatalf("CFDASCII missing day-0 row %q:\n%s", want, out)
	}
	// Legend present so a reader can decode A/I/B columns.
	if !strings.Contains(out, "legend:") {
		t.Errorf("CFDASCII missing legend:\n%s", out)
	}
}

func TestBurnupASCIIAccurate(t *testing.T) {
	dir := t.TempDir()
	// Two stories worth 3 + 5 = 8 points of scope. One accepts on
	// day 2, one accepts on day 4. By the last day scope=done=8.
	makeItem(t, dir, "x", map[string]string{
		"title": "x", "status": "accepted", "type": "feature", "estimate": "3",
		"created": "2026-05-01T09:00:00Z", "started": "2026-05-01T10:00:00Z",
		"finished": "2026-05-02T08:00:00Z", "accepted": "2026-05-02T15:00:00Z",
	})
	makeItem(t, dir, "y", map[string]string{
		"title": "y", "status": "accepted", "type": "feature", "estimate": "5",
		"created": "2026-05-01T09:00:00Z", "started": "2026-05-02T10:00:00Z",
		"finished": "2026-05-04T08:00:00Z", "accepted": "2026-05-04T15:00:00Z",
	})
	items := make([]*BacklogItem, 0)
	for _, name := range []string{"x", "y"} {
		it, _ := LoadBacklogItem(filepath.Join(dir, name+".md"))
		items = append(items, it)
	}
	day := func(s string) time.Time { v, _ := time.Parse(time.RFC3339, s+"T00:00:00Z"); return v }
	rows := BurnupRows(items, day("2026-05-01"), day("2026-05-06"))
	out := BurnupASCII(rows, day("2026-05-01"), day("2026-05-06"))
	// Day-2 row must show 3.0 / 8.0 done/scope after the first accept.
	if !strings.Contains(out, " 3.0 /  8.0") {
		t.Errorf("BurnupASCII missing the day-2 done=3/scope=8 row:\n%s", out)
	}
	// Day-4 row must show 8.0 / 8.0 done/scope after the second accept.
	if !strings.Contains(out, " 8.0 /  8.0") {
		t.Errorf("BurnupASCII missing the day-4 done=8/scope=8 row:\n%s", out)
	}
}

func TestTypeMixASCIIAccurate(t *testing.T) {
	rows := []TypeMixRow{
		{Type: "feature", Count: 18, Percent: 50},
		{Type: "bug", Count: 7, Percent: 19.4},
		{Type: "chore", Count: 6, Percent: 16.7},
		{Type: "release", Count: 5, Percent: 13.9},
	}
	out := TypeMixASCII(rows, 36)
	// All four counts must show up.
	for _, want := range []string{"feature", " 18", "bug", "  7", "chore", "  6", "release", "  5"} {
		if !strings.Contains(out, want) {
			t.Errorf("TypeMixASCII missing %q:\n%s", want, out)
		}
	}
	// Total in the header.
	if !strings.Contains(out, "36 accepted") {
		t.Errorf("TypeMixASCII missing total %q:\n%s", "36 accepted", out)
	}
}

func TestTypeMixASCIIEmpty(t *testing.T) {
	out := TypeMixASCII(nil, 0)
	if !strings.Contains(out, "no accepted stories") {
		t.Errorf("TypeMixASCII empty path should produce a friendly message, got:\n%s", out)
	}
}

func TestItemIterationIceboxAndBacklog(t *testing.T) {
	c := config.Defaults()
	dir := t.TempDir()
	it := makeItem(t, dir, "y", map[string]string{
		"title": "y", "status": "unstarted",
		"created": "2026-05-01T09:00:00Z",
	})
	num, label := ItemIteration(it, c, 0, 0)
	if num != 0 || label != "icebox" {
		t.Fatalf("expected (0,'icebox') for orphan unstarted, got (%d,%q)", num, label)
	}
	num, label = ItemIteration(it, c, 3, 0)
	if num != 0 || label != "backlog" {
		t.Fatalf("expected (0,'backlog') for unstarted-in-priority with no velocity, got (%d,%q)", num, label)
	}
	num, label = ItemIteration(it, c, 7, 5)
	// position 7 with velocity 5 -> band 1 (second iteration). number = current + 1
	if num == 0 || label != "" {
		t.Fatalf("expected numeric band for pos=7 vel=5, got (%d,%q)", num, label)
	}
}
