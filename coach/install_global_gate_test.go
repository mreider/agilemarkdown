package coach

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readSettings loads the global settings.json under home for assertions.
func readSettings(t *testing.T, home string) map[string]interface{} {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("settings.json is not valid JSON: %v", err)
	}
	return m
}

// gateMatchers returns the PreToolUse matchers wired to the work-gate.
func gateMatchers(t *testing.T, root map[string]interface{}) []string {
	t.Helper()
	hooks, _ := root["hooks"].(map[string]interface{})
	pre, _ := hooks["PreToolUse"].([]interface{})
	var out []string
	for _, e := range pre {
		m, _ := e.(map[string]interface{})
		hs, _ := m["hooks"].([]interface{})
		for _, h := range hs {
			hm, _ := h.(map[string]interface{})
			if c, _ := hm["command"].(string); strings.Contains(c, "am-work-gate.sh") {
				out = append(out, m["matcher"].(string))
			}
		}
	}
	return out
}

func TestInstallGlobalGate_FreshHome(t *testing.T) {
	home := t.TempDir()
	changed, err := InstallGlobalGate(home)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if len(changed) != 2 {
		t.Fatalf("fresh install should change script + settings, got %v", changed)
	}
	// Script exists and is executable.
	info, err := os.Stat(filepath.Join(home, ".claude", "hooks", "am-work-gate.sh"))
	if err != nil {
		t.Fatalf("script not written: %v", err)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("script not executable: %v", info.Mode())
	}
	// All three edit tools are gated.
	got := gateMatchers(t, readSettings(t, home))
	if len(got) != 3 {
		t.Fatalf("want Edit/Write/NotebookEdit gated, got %v", got)
	}
}

func TestInstallGlobalGate_Idempotent(t *testing.T) {
	home := t.TempDir()
	if _, err := InstallGlobalGate(home); err != nil {
		t.Fatalf("first install: %v", err)
	}
	changed, err := InstallGlobalGate(home)
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	if len(changed) != 0 {
		t.Fatalf("second install should be a no-op, changed %v", changed)
	}
}

func TestInstallGlobalGate_PreservesExistingSettings(t *testing.T) {
	home := t.TempDir()
	claude := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claude, 0o755); err != nil {
		t.Fatal(err)
	}
	// Pre-existing settings with an unrelated key and an unrelated hook.
	seed := `{
  "effortLevel": "high",
  "hooks": { "PreToolUse": [
    { "matcher": "Bash", "hooks": [ { "type": "command", "command": "bash other.sh" } ] }
  ] }
}`
	if err := os.WriteFile(filepath.Join(claude, "settings.json"), []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallGlobalGate(home); err != nil {
		t.Fatalf("install: %v", err)
	}
	root := readSettings(t, home)
	if root["effortLevel"] != "high" {
		t.Fatalf("unrelated key dropped: %v", root["effortLevel"])
	}
	if len(gateMatchers(t, root)) != 3 {
		t.Fatalf("gate not added alongside existing hooks")
	}
	// The unrelated Bash hook must survive.
	hooks := root["hooks"].(map[string]interface{})
	pre := hooks["PreToolUse"].([]interface{})
	foundBash := false
	for _, e := range pre {
		if e.(map[string]interface{})["matcher"] == "Bash" {
			foundBash = true
		}
	}
	if !foundBash {
		t.Fatalf("existing Bash hook was clobbered")
	}
}
