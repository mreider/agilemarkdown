package actions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mreider/agilemarkdown/backlog"
)

func writeItem(t *testing.T, dir, name, status string) *backlog.BacklogItem {
	t.Helper()
	path := filepath.Join(dir, name+".md")
	body := "---\n" +
		"title: " + name + "\n" +
		"status: " + status + "\n" +
		"type: feature\n" +
		"---\n\nSome body.\n"
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	item, err := backlog.LoadBacklogItem(path)
	if err != nil {
		t.Fatal(err)
	}
	return item
}

func TestStartedStampedOnTransition(t *testing.T) {
	dir := t.TempDir()
	item := writeItem(t, dir, "story", "unstarted")
	ApplyStatusTransition(item, backlog.StartedStatus)
	if item.Started().IsZero() {
		t.Fatal("expected started: to be stamped on transition to started")
	}
	if !item.Finished().IsZero() || !item.Delivered().IsZero() || !item.Accepted().IsZero() {
		t.Fatal("expected completion timestamps to be zero on freshly started")
	}
}

func TestStartedBackfilledWhenSkippingForward(t *testing.T) {
	dir := t.TempDir()
	item := writeItem(t, dir, "story", "unstarted")
	ApplyStatusTransition(item, backlog.AcceptedStatus)
	if item.Started().IsZero() {
		t.Fatal("expected started: to be back-filled when jumping straight to accepted")
	}
	if item.Accepted().IsZero() {
		t.Fatal("expected accepted: to be stamped")
	}
}

func TestUnstartedClearsStarted(t *testing.T) {
	dir := t.TempDir()
	item := writeItem(t, dir, "story", "unstarted")
	ApplyStatusTransition(item, backlog.StartedStatus)
	ApplyStatusTransition(item, backlog.UnstartedStatus)
	if !item.Started().IsZero() {
		t.Fatal("expected started: to be cleared on full reset to unstarted")
	}
}

func TestStartedPreservedOnSecondStart(t *testing.T) {
	dir := t.TempDir()
	item := writeItem(t, dir, "story", "unstarted")
	ApplyStatusTransition(item, backlog.StartedStatus)
	originalStarted := item.Started()
	ApplyStatusTransition(item, backlog.RejectedStatus)
	ApplyStatusTransition(item, backlog.StartedStatus)
	if !item.Started().Equal(originalStarted) {
		t.Fatalf("expected started: to be preserved across restart, got %v vs %v",
			item.Started(), originalStarted)
	}
}

// guards the rest of the file from being optimized into a single huge
// regression test; keep this so split test failures stay readable.
var _ = strings.TrimSpace
