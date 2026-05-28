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

// TestFinishStampsFinishedTimestamp confirms that walking forward
// from started to finished writes finished: now and leaves the
// earlier timestamps untouched.
func TestFinishStampsFinishedTimestamp(t *testing.T) {
	dir := t.TempDir()
	item := writeItem(t, dir, "story", "unstarted")
	ApplyStatusTransition(item, backlog.StartedStatus)
	originalStarted := item.Started()
	ApplyStatusTransition(item, backlog.FinishedStatus)
	if item.Finished().IsZero() {
		t.Fatal("finished: was not stamped on transition to finished")
	}
	if !item.Started().Equal(originalStarted) {
		t.Fatal("started: must not change on the finish transition")
	}
}

// TestDeliverStampsDeliveredTimestamp ensures the delivery transition
// records its own timestamp and back-fills finished: if missing.
func TestDeliverStampsDeliveredTimestamp(t *testing.T) {
	dir := t.TempDir()
	item := writeItem(t, dir, "story", "unstarted")
	ApplyStatusTransition(item, backlog.StartedStatus)
	ApplyStatusTransition(item, backlog.DeliveredStatus)
	if item.Finished().IsZero() {
		t.Fatal("finished: must be back-filled if missing when delivering")
	}
	if item.Delivered().IsZero() {
		t.Fatal("delivered: was not stamped")
	}
}

// TestAcceptStampsAllPriorTimestamps: jumping to accepted from any
// state must populate started, finished, delivered, and accepted so
// cycle-time and velocity math have complete data.
func TestAcceptStampsAllPriorTimestamps(t *testing.T) {
	dir := t.TempDir()
	item := writeItem(t, dir, "story", "unstarted")
	ApplyStatusTransition(item, backlog.AcceptedStatus)
	if item.Started().IsZero() || item.Finished().IsZero() ||
		item.Delivered().IsZero() || item.Accepted().IsZero() {
		t.Fatalf("expected every transition timestamp filled on accept; got %+v",
			map[string]string{
				"started":   item.Started().String(),
				"finished":  item.Finished().String(),
				"delivered": item.Delivered().String(),
				"accepted":  item.Accepted().String(),
			})
	}
}

// TestRejectClearsAccepted: a rejection wipes the accepted timestamp
// while leaving the rest of the audit trail intact. The reject path
// in the MCP layer additionally appends a comment and re-routes the
// story back to started, but ApplyStatusTransition itself is pure.
func TestRejectClearsAccepted(t *testing.T) {
	dir := t.TempDir()
	item := writeItem(t, dir, "story", "unstarted")
	ApplyStatusTransition(item, backlog.AcceptedStatus)
	if item.Accepted().IsZero() {
		t.Fatal("setup precondition: expected accepted timestamp before reject")
	}
	ApplyStatusTransition(item, backlog.RejectedStatus)
	if !item.Accepted().IsZero() {
		t.Fatal("expected accepted: cleared after reject")
	}
	if item.Started().IsZero() {
		t.Fatal("expected started: retained across reject so cycle-time still has an anchor")
	}
}

// TestChoreShortcutsFinishedDelivered ensures chores collapse the
// state machine: anything past unstarted lands directly at accepted.
func TestChoreShortcutsFinishedDelivered(t *testing.T) {
	dir := t.TempDir()
	item := writeItem(t, dir, "story", "unstarted")
	item.SetType("chore")
	ApplyStatusTransition(item, backlog.FinishedStatus)
	if item.Status() != backlog.AcceptedStatus.Name {
		t.Fatalf("expected chore to short-circuit finished -> accepted; got %s", item.Status())
	}
}

// TestReleaseShortcutsInFlight: release markers don't go through
// started/finished/delivered. Any in-flight transition collapses to
// accepted so the ship moment is what gets recorded.
func TestReleaseShortcutsInFlight(t *testing.T) {
	dir := t.TempDir()
	item := writeItem(t, dir, "release-stake", "unstarted")
	item.SetType("release")
	ApplyStatusTransition(item, backlog.StartedStatus)
	if item.Status() != backlog.AcceptedStatus.Name {
		t.Fatalf("expected release to short-circuit started -> accepted; got %s", item.Status())
	}
	if item.Accepted().IsZero() {
		t.Fatal("expected accepted: stamped on the release ship event")
	}
}

// guards the rest of the file from being optimized into a single huge
// regression test; keep this so split test failures stay readable.
var _ = strings.TrimSpace
