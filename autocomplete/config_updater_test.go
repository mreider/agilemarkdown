package autocomplete

import (
	"os"
	"path/filepath"
	"testing"
)

// A missing shell rc must be a silent no-op, not an error. The
// autocomplete setup runs on *every* `am` invocation, and any error it
// returns gets printed ahead of real output — which corrupts the
// machine-readable stdout the VS Code extension parses (`--json`).
// zsh users have no ~/.bash_profile, so this is the common path.
func TestAddLineToConfigMissingFileIsSilentNoOp(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-rc")

	if err := addLineToConfig(missing, "export FOO=bar"); err != nil {
		t.Fatalf("addLineToConfig on a missing rc should be a silent no-op, got: %v", err)
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("addLineToConfig must not create a missing rc (stat err: %v)", err)
	}
}

// When the rc exists, the line is appended (and is idempotent).
func TestAddLineToConfigAppendsWhenPresent(t *testing.T) {
	rc := filepath.Join(t.TempDir(), ".bash_profile")
	if err := os.WriteFile(rc, []byte("# existing\n"), 0644); err != nil {
		t.Fatal(err)
	}
	line := "PROG=am source /tmp/am-autocomplete"

	for i := 0; i < 2; i++ {
		if err := addLineToConfig(rc, line); err != nil {
			t.Fatalf("addLineToConfig: %v", err)
		}
	}
	content, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, ln := range splitLines(string(content)) {
		if ln == line {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected the line exactly once (idempotent), found %d times", count)
	}
}

func splitLines(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == '\n' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	out = append(out, cur)
	return out
}
