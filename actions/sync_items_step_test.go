package actions

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mreider/agilemarkdown/backlog"
)

// The icebox-dump bug: when a story's title changes, sync renames the
// file to the new slug and updates the overview link — but left the
// _priority.md / _icebox.md entry pointing at the old name. The next
// invariant pass dropped the orphaned entry and re-ingested the item
// as brand new at the icebox bottom, silently destroying its rank.
// This test pins the fix: the rename follows the item into its order
// file, same position, fresh title.
func TestSyncItemsStep_RenameFollowsIntoOrderFiles(t *testing.T) {
	dir := t.TempDir()
	// git repo: the sync step shells out to git for modified files,
	// repo versions, and staging the renamed file.
	mustRunGit(t, dir, "init")
	mustRunGit(t, dir, "config", "user.email", "test@example.com")
	mustRunGit(t, dir, "config", "user.name", "Test")

	backlogDir := filepath.Join(dir, "am-product")
	if err := os.MkdirAll(backlogDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "am-product.md"), []byte("# am-product\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Three stories so the renamed one sits in the MIDDLE of priority —
	// position must survive the rename.
	for _, name := range []string{"First", "Story", "Last"} {
		body := "---\ntitle: " + name + "\nstatus: unstarted\ntype: feature\n---\n\nBody.\n"
		if err := os.WriteFile(filepath.Join(backlogDir, name+".md"), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	priPath := filepath.Join(backlogDir, "_priority.md")
	if err := os.WriteFile(priPath,
		[]byte("# Priority\n\n- [First](First.md)\n- [Story](Story.md)\n- [Last](Last.md)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mustRunGit(t, dir, "add", "-A")
	mustRunGit(t, dir, "commit", "-m", "seed")

	// Retitle the middle story; its slug no longer matches the file name.
	storyPath := filepath.Join(backlogDir, "Story.md")
	item, err := backlog.LoadBacklogItem(storyPath)
	if err != nil {
		t.Fatal(err)
	}
	item.SetTitle("Renamed story with a new slug")
	if err := item.Save(); err != nil {
		t.Fatal(err)
	}

	// git.Add inside the step runs in the process cwd.
	t.Chdir(dir)
	root := backlog.NewBacklogsStructure(dir)
	if err := NewSyncItemsStep(root).Execute(); err != nil {
		t.Fatalf("sync items step: %v", err)
	}

	newName := "renamed-story-with-a-new-slug.md"
	if _, err := os.Stat(filepath.Join(backlogDir, newName)); err != nil {
		t.Fatalf("expected renamed file %s: %v", newName, err)
	}

	pri, err := backlog.LoadPriority(backlogDir)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, pri.Len())
	for _, e := range pri.Entries() {
		got = append(got, e.Path)
	}
	want := []string{"First.md", newName, "Last.md"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("priority after rename: want %v got %v", want, got)
	}

	// The end-to-end symptom: the invariant pass must keep the item in
	// priority (it used to drop the orphan and re-ingest at icebox).
	bck, err := backlog.LoadBacklog(backlogDir)
	if err != nil {
		t.Fatal(err)
	}
	priOut, iceOut, _, err := backlog.EnforceOrderInvariant(bck, backlogDir)
	if err != nil {
		t.Fatal(err)
	}
	if priOut.IndexOf(newName) != 1 {
		t.Fatalf("renamed item lost its priority rank: %v", priOut.Entries())
	}
	if iceOut.IndexOf(newName) >= 0 {
		t.Fatalf("renamed item was dumped to the icebox")
	}
}

func mustRunGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
