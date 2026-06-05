package mcpserver

import (
	"path/filepath"
	"testing"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"
)

// set_mode persists the mode; an autonomous self-accept via set_status is
// journaled (with an honest marker when no note is given, or the note
// verbatim); the journal tool round-trips.
func TestMCPModeJournalRoundTrip(t *testing.T) {
	dir := t.TempDir()
	mustInitRepo(t, dir)
	mustWriteItem(t, dir, "alpha", map[string]string{"status": "unstarted", "type": "feature", "estimate": "3"})
	mustWriteItem(t, dir, "beta", map[string]string{"status": "unstarted", "type": "feature", "estimate": "2"})

	if _, err := callToolViaMemoryTransport(t, dir, "set_mode", map[string]any{"mode": "autonomous"}); err != nil {
		t.Fatalf("set_mode: %v", err)
	}
	cfg, err := config.LoadConfig(filepath.Join(dir, ".am", "config.yaml"))
	if err != nil || !cfg.IsAutonomous() {
		t.Fatalf("set_mode did not persist autonomous (err=%v cfg=%+v)", err, cfg)
	}

	// Self-accept, no note -> honest autonomous marker.
	if _, err := callToolViaMemoryTransport(t, dir, "set_status", map[string]any{
		"path": "am-product/alpha.md", "status": "accepted",
	}); err != nil {
		t.Fatalf("set_status accepted (alpha): %v", err)
	}
	// Self-accept with explicit note.
	if _, err := callToolViaMemoryTransport(t, dir, "set_status", map[string]any{
		"path": "am-product/beta.md", "status": "accepted", "note": "AC met: verified",
	}); err != nil {
		t.Fatalf("set_status accepted (beta): %v", err)
	}

	entries, err := backlog.LoadActivity(dir)
	if err != nil {
		t.Fatal(err)
	}
	var sawAuto, sawNote bool
	for _, e := range entries {
		if e.Item == "am-product/alpha.md" && e.Verb == "accepted" && e.Detail == "self-accepted (autonomous mode)" {
			sawAuto = true
		}
		if e.Item == "am-product/beta.md" && e.Verb == "accepted" && e.Detail == "AC met: verified" {
			sawNote = true
		}
	}
	if !sawAuto {
		t.Errorf("expected autonomous self-accept marker for alpha; entries=%+v", entries)
	}
	if !sawNote {
		t.Errorf("expected explicit note for beta; entries=%+v", entries)
	}

	if _, err := callToolViaMemoryTransport(t, dir, "journal", map[string]any{}); err != nil {
		t.Fatalf("journal: %v", err)
	}
}
