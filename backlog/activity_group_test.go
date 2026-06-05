package backlog

import "testing"

func TestGroupActivityByItemPreservesFirstSeenOrder(t *testing.T) {
	entries := []ActivityEntry{
		{Item: "a.md", Verb: "started"},
		{Item: "b.md", Verb: "created"},
		{Item: "a.md", Verb: "finished"},
		{Item: "", Verb: "mode"},
		{Item: "a.md", Verb: "accepted"},
	}
	groups := GroupActivityByItem(entries)
	if len(groups) != 3 {
		t.Fatalf("want 3 groups, got %d", len(groups))
	}
	if groups[0].Item != "a.md" || len(groups[0].Entries) != 3 {
		t.Fatalf("group[0] should be a.md with 3 entries, got %q/%d", groups[0].Item, len(groups[0].Entries))
	}
	if groups[1].Item != "b.md" {
		t.Fatalf("group[1] should be b.md, got %q", groups[1].Item)
	}
	if groups[2].Item != "" {
		t.Fatalf("group[2] should be the repo-wide bucket (empty item), got %q", groups[2].Item)
	}
}
