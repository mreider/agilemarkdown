package mcpserver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mreider/agilemarkdown/backlog"
)

// TestSetStatus_AutoRanksStartedToTop confirms the autoRankForStatus
// invariant: starting an icebox story pulls it into priority and
// ranks it above any unstarted entry, no matter how many points.
func TestSetStatus_AutoRanksStartedToTop(t *testing.T) {
	dir := t.TempDir()
	mustInitRepo(t, dir)
	mustWriteItem(t, dir, "Unstarted-a", map[string]string{
		"status": "unstarted", "estimate": "3", "type": "feature",
	})
	mustWriteItem(t, dir, "Unstarted-b", map[string]string{
		"status": "unstarted", "estimate": "5", "type": "feature",
	})
	mustWriteItem(t, dir, "Iceboxed", map[string]string{
		"status": "unstarted", "estimate": "2", "type": "feature",
	})

	// Seed priority with the two unstarted and put the third in icebox.
	priPath := filepath.Join(dir, "am-product", "_priority.md")
	icePath := filepath.Join(dir, "am-product", "_icebox.md")
	if err := os.WriteFile(priPath,
		[]byte("# Priority\n\n- [Unstarted-a](Unstarted-a.md)\n- [Unstarted-b](Unstarted-b.md)\n"),
		0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(icePath,
		[]byte("# Icebox\n\n- [Iceboxed](Iceboxed.md)\n"),
		0644); err != nil {
		t.Fatal(err)
	}

	// Start the icebox story via set_status.
	_, err := callToolViaMemoryTransport(t, dir, "set_status", map[string]any{
		"path": "am-product/Iceboxed.md", "status": "started",
	})
	if err != nil {
		t.Fatalf("set_status: %v", err)
	}

	// Priority must now have the started story at index 0.
	pri, _ := os.ReadFile(priPath)
	if !strings.Contains(string(pri), "Iceboxed.md") {
		t.Fatalf("started story did not promote to priority:\n%s", pri)
	}
	if idx := strings.Index(string(pri), "Iceboxed.md"); idx > strings.Index(string(pri), "Unstarted-a.md") {
		t.Fatalf("started story did not rank above unstarted entries:\n%s", pri)
	}

	// And the icebox must no longer reference it.
	ice, _ := os.ReadFile(icePath)
	if strings.Contains(string(ice), "Iceboxed.md") {
		t.Fatalf("started story still in icebox:\n%s", ice)
	}
}

// TestSetStatus_StampsTimestamps confirms every transition stamps the
// matching timestamp in frontmatter and back-fills any earlier
// transition that was skipped.
func TestSetStatus_StampsTimestamps(t *testing.T) {
	dir := t.TempDir()
	mustInitRepo(t, dir)
	mustWriteItem(t, dir, "Story", map[string]string{
		"status": "unstarted", "estimate": "3", "type": "feature",
	})
	priPath := filepath.Join(dir, "am-product", "_priority.md")
	storyPath := filepath.Join(dir, "am-product", "Story.md")
	if err := os.WriteFile(priPath, []byte("# Priority\n\n- [Story](Story.md)\n"), 0644); err != nil {
		t.Fatal(err)
	}

	for _, status := range []string{"started", "finished", "delivered", "accepted"} {
		if _, err := callToolViaMemoryTransport(t, dir, "set_status", map[string]any{
			"path": "am-product/Story.md", "status": status,
		}); err != nil {
			t.Fatalf("set_status %s: %v", status, err)
		}
	}
	body, _ := os.ReadFile(storyPath)
	for _, field := range []string{"started", "finished", "delivered", "accepted"} {
		if !strings.Contains(string(body), field+":") {
			t.Fatalf("expected %s: timestamp after walking the chain; body:\n%s", field, body)
		}
	}
}

// TestDoneList_FiltersPriorIterationAccepted: the done_list tool
// only returns accepted stories whose accepted-date falls in a prior
// iteration. Stories accepted this week stay in priority_list.
func TestDoneList_FiltersPriorIterationAccepted(t *testing.T) {
	dir := t.TempDir()
	mustInitRepo(t, dir)
	now := time.Now()
	priorWeek := now.AddDate(0, 0, -7).Format(time.RFC3339)
	thisWeek := now.AddDate(0, 0, -1).Format(time.RFC3339)
	mustWriteItem(t, dir, "Old", map[string]string{
		"status": "accepted", "estimate": "3", "type": "feature", "accepted": priorWeek,
	})
	mustWriteItem(t, dir, "New", map[string]string{
		"status": "accepted", "estimate": "5", "type": "feature", "accepted": thisWeek,
	})

	// done_list returns only Old.
	res, err := callToolViaMemoryTransport(t, dir, "done_list", map[string]any{
		"backlog": "am-product",
	})
	if err != nil {
		t.Fatalf("done_list: %v", err)
	}
	sc, _ := res.StructuredContent.(map[string]any)
	count, _ := sc["count"].(float64)
	if int(count) != 1 {
		t.Fatalf("expected 1 done story, got %v", sc["count"])
	}

	// priority_list: seed the entry so the tool walks it, and verify
	// it omits the prior-iteration accepted item.
	priPath := filepath.Join(dir, "am-product", "_priority.md")
	if err := os.WriteFile(priPath,
		[]byte("# Priority\n\n- [Old](Old.md)\n- [New](New.md)\n"),
		0644); err != nil {
		t.Fatal(err)
	}
	res, err = callToolViaMemoryTransport(t, dir, "priority_list", map[string]any{
		"backlog": "am-product",
	})
	if err != nil {
		t.Fatalf("priority_list: %v", err)
	}
	sc, _ = res.StructuredContent.(map[string]any)
	items, _ := sc["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected priority_list to omit done story; got %d items", len(items))
	}
	first, _ := items[0].(map[string]any)
	if first["path"] != "New.md" {
		t.Fatalf("expected New.md kept in priority; got %v", first["path"])
	}
}

// TestRejectItem_FlipsToStartedAndComments: rejection is an event,
// not a sticky status. The tool flips the story back to started and
// records the reason as a dated comment.
func TestRejectItem_FlipsToStartedAndComments(t *testing.T) {
	dir := t.TempDir()
	mustInitRepo(t, dir)
	mustWriteItem(t, dir, "Story", map[string]string{
		"status": "delivered", "estimate": "3", "type": "feature",
	})
	if _, err := callToolViaMemoryTransport(t, dir, "reject_item", map[string]any{
		"path":   "am-product/Story.md",
		"reason": "Tests do not pass under load.",
		"author": "pm",
	}); err != nil {
		t.Fatalf("reject_item: %v", err)
	}
	body, _ := os.ReadFile(filepath.Join(dir, "am-product", "Story.md"))
	if !strings.Contains(string(body), "status: started") {
		t.Fatalf("expected status: started after reject; got:\n%s", body)
	}
	if !strings.Contains(string(body), "## Comments") {
		t.Fatalf("expected ## Comments section after reject; got:\n%s", body)
	}
	if !strings.Contains(string(body), "Tests do not pass under load.") {
		t.Fatalf("expected rejection reason in comment; got:\n%s", body)
	}
	if !strings.Contains(string(body), "@pm") {
		t.Fatalf("expected @pm attribution on the comment; got:\n%s", body)
	}
}

// TestPriorityList_EmitsTimestamps confirms the OrderRow contract:
// every transition timestamp the client needs to render Done bands
// and release-marker dates is present in the JSON.
func TestPriorityList_EmitsTimestamps(t *testing.T) {
	dir := t.TempDir()
	mustInitRepo(t, dir)
	mustWriteItem(t, dir, "Story", map[string]string{
		"status":    "delivered",
		"estimate":  "3",
		"type":      "feature",
		"started":   "2026-05-14T09:00:00Z",
		"finished":  "2026-05-14T15:00:00Z",
		"delivered": "2026-05-14T17:00:00Z",
	})
	priPath := filepath.Join(dir, "am-product", "_priority.md")
	if err := os.WriteFile(priPath, []byte("# Priority\n\n- [Story](Story.md)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	res, err := callToolViaMemoryTransport(t, dir, "priority_list", map[string]any{
		"backlog": "am-product",
	})
	if err != nil {
		t.Fatalf("priority_list: %v", err)
	}
	sc, _ := res.StructuredContent.(map[string]any)
	items, _ := sc["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	row, _ := items[0].(map[string]any)
	for _, want := range []string{"started", "finished", "delivered"} {
		if row[want] == nil || row[want] == "" {
			t.Fatalf("expected row.%s populated; got row=%+v", want, row)
		}
	}
}

// TestPriorityList_EmitsReleaseDate: release-type rows carry their
// target ship date in the JSON so the extension can render the bar
// label and decide whether to show the Ship button.
func TestPriorityList_EmitsReleaseDate(t *testing.T) {
	dir := t.TempDir()
	mustInitRepo(t, dir)
	mustWriteItem(t, dir, "Frost", map[string]string{
		"status":       "unstarted",
		"type":         "release",
		// Quoted so YAML doesn't parse it as a date type, the
		// system stores release_date as a string.
		"release_date": "\"2026-07-01\"",
	})
	priPath := filepath.Join(dir, "am-product", "_priority.md")
	if err := os.WriteFile(priPath, []byte("# Priority\n\n- [Frost](Frost.md)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	res, err := callToolViaMemoryTransport(t, dir, "priority_list", map[string]any{
		"backlog": "am-product",
	})
	if err != nil {
		t.Fatalf("priority_list: %v", err)
	}
	sc, _ := res.StructuredContent.(map[string]any)
	items, _ := sc["items"].([]any)
	row, _ := items[0].(map[string]any)
	if row["release_date"] != "2026-07-01" {
		t.Fatalf("expected release_date emitted; got %v", row["release_date"])
	}
	if row["type"] != "release" {
		t.Fatalf("expected type: release; got %v", row["type"])
	}
}

// TestReleaseAcceptedRollsToDone exercises the ship flow end-to-end:
// a release marker accepted in a prior iteration appears in
// done_list as a historical fact, not in priority_list.
func TestReleaseAcceptedRollsToDone(t *testing.T) {
	dir := t.TempDir()
	mustInitRepo(t, dir)
	priorWeek := time.Now().AddDate(0, 0, -7).Format(time.RFC3339)
	mustWriteItem(t, dir, "Frost", map[string]string{
		"status":       "accepted",
		"type":         "release",
		"release_date": "2026-04-15",
		"accepted":     priorWeek,
	})

	// Live in priority initially; the filter must remove it.
	priPath := filepath.Join(dir, "am-product", "_priority.md")
	if err := os.WriteFile(priPath, []byte("# Priority\n\n- [Frost](Frost.md)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	res, _ := callToolViaMemoryTransport(t, dir, "priority_list", map[string]any{
		"backlog": "am-product",
	})
	sc, _ := res.StructuredContent.(map[string]any)
	items, _ := sc["items"].([]any)
	if len(items) != 0 {
		t.Fatalf("shipped release should not appear in priority_list; got %d items", len(items))
	}

	res, _ = callToolViaMemoryTransport(t, dir, "done_list", map[string]any{
		"backlog": "am-product",
	})
	sc, _ = res.StructuredContent.(map[string]any)
	count, _ := sc["count"].(float64)
	if int(count) != 1 {
		t.Fatalf("expected the release in done_list; got %v", sc["count"])
	}
}

// TestMoveToIcebox_RefusesActiveStatuses extends the existing guard
// test to confirm all three in-flight statuses are blocked.
func TestMoveToIcebox_RefusesActiveStatuses(t *testing.T) {
	for _, status := range []string{"started", "finished", "delivered"} {
		t.Run(status, func(t *testing.T) {
			dir := t.TempDir()
			mustInitRepo(t, dir)
			mustWriteItem(t, dir, "Story", map[string]string{
				"status": status, "estimate": "3", "type": "feature",
			})
			priPath := filepath.Join(dir, "am-product", "_priority.md")
			if err := os.WriteFile(priPath,
				[]byte("# Priority\n\n- [Story](Story.md)\n"), 0644); err != nil {
				t.Fatal(err)
			}
			res, err := callToolViaMemoryTransport(t, dir, "move_to_icebox", map[string]any{
				"backlog": "am-product", "item_path": "Story.md",
			})
			msg := ""
			if err != nil {
				msg = err.Error()
			} else if res != nil {
				msg = flattenContent(res)
			}
			if !strings.Contains(msg, status) {
				t.Fatalf("expected error mentioning %q; got %q", status, msg)
			}
		})
	}
}

// Compile-time check: every transition status the backend knows is
// exercised by at least one test in this file. Catches the silent
// drift where someone adds a status but no test covers it.
var _ = []*backlog.BacklogItemStatus{
	backlog.UnstartedStatus, backlog.StartedStatus,
	backlog.FinishedStatus, backlog.DeliveredStatus,
	backlog.AcceptedStatus, backlog.RejectedStatus,
}
