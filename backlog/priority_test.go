package backlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeItem(t *testing.T, dir, name, status string) string {
	t.Helper()
	path := filepath.Join(dir, name+".md")
	body := "---\ntitle: " + name + "\nstatus: " + status + "\n---\n\nbody\n"
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestOrderFileMoveSemantics(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "_priority.md")
	f, err := LoadOrderFile(path, "Priority")
	if err != nil {
		t.Fatal(err)
	}
	f.InsertBottom(OrderEntry{Title: "A", Path: "a.md"})
	f.InsertBottom(OrderEntry{Title: "B", Path: "b.md"})
	f.InsertBottom(OrderEntry{Title: "C", Path: "c.md"})
	f.InsertBottom(OrderEntry{Title: "D", Path: "d.md"})

	// Move D to top.
	if !f.MoveTo("d.md", 0) {
		t.Fatal("move d to top failed")
	}
	if f.Entries()[0].Path != "d.md" {
		t.Fatalf("want d.md at 0, got %v", f.Entries()[0])
	}

	// MoveAfter b.md -> c.md (D after C).
	if !f.MoveAfter("d.md", "c.md") {
		t.Fatal("move after failed")
	}
	got := []string{}
	for _, e := range f.Entries() {
		got = append(got, e.Path)
	}
	wantAfter := []string{"a.md", "b.md", "c.md", "d.md"}
	if strings.Join(got, ",") != strings.Join(wantAfter, ",") {
		t.Fatalf("after MoveAfter want %v got %v", wantAfter, got)
	}

	// MoveBefore: A before D.
	if !f.MoveBefore("a.md", "d.md") {
		t.Fatal("move before failed")
	}
	got = got[:0]
	for _, e := range f.Entries() {
		got = append(got, e.Path)
	}
	wantBefore := []string{"b.md", "c.md", "a.md", "d.md"}
	if strings.Join(got, ",") != strings.Join(wantBefore, ",") {
		t.Fatalf("after MoveBefore want %v got %v", wantBefore, got)
	}

	// Round-trip through disk.
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	g, err := LoadOrderFile(path, "Priority")
	if err != nil {
		t.Fatal(err)
	}
	got = got[:0]
	for _, e := range g.Entries() {
		got = append(got, e.Path)
	}
	if strings.Join(got, ",") != strings.Join(wantBefore, ",") {
		t.Fatalf("after reload want %v got %v", wantBefore, got)
	}
}

func TestEnforceOrderInvariant(t *testing.T) {
	dir := t.TempDir()
	writeItem(t, dir, "alpha", "unstarted")
	writeItem(t, dir, "beta", "started")
	writeItem(t, dir, "gamma", "unstarted")

	bck, err := LoadBacklog(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Pre-seed _priority.md with alpha + a stale orphan.
	pri, _ := LoadPriority(dir)
	pri.InsertBottom(OrderEntry{Title: "Alpha", Path: "alpha.md"})
	pri.InsertBottom(OrderEntry{Title: "Stale", Path: "ghost.md"})
	if err := pri.Save(); err != nil {
		t.Fatal(err)
	}

	// Pre-seed _icebox.md with alpha (collision) + beta.
	ice, _ := LoadIcebox(dir)
	ice.InsertBottom(OrderEntry{Title: "Alpha", Path: "alpha.md"})
	ice.InsertBottom(OrderEntry{Title: "Beta", Path: "beta.md"})
	if err := ice.Save(); err != nil {
		t.Fatal(err)
	}

	priOut, iceOut, changed, err := EnforceOrderInvariant(bck, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected changes")
	}

	gotPri := []string{}
	for _, e := range priOut.Entries() {
		gotPri = append(gotPri, e.Path)
	}
	if strings.Join(gotPri, ",") != "alpha.md" {
		t.Fatalf("priority want [alpha.md] got %v", gotPri)
	}

	gotIce := []string{}
	for _, e := range iceOut.Entries() {
		gotIce = append(gotIce, e.Path)
	}
	// beta from icebox (alpha collision dropped, ghost orphan dropped),
	// plus gamma appended (was in neither).
	want := []string{"beta.md", "gamma.md"}
	if strings.Join(gotIce, ",") != strings.Join(want, ",") {
		t.Fatalf("icebox want %v got %v", want, gotIce)
	}
}
