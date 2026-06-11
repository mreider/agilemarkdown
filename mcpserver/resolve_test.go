package mcpserver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The husk bug: markdown.LoadFrontmatter returns an empty in-memory
// file for a path that doesn't exist, so any tool that joined
// root+args.Path and then saved would silently CREATE a frontmatter
// husk wherever the bad path landed — typically the repo root, when a
// caller passed a bare file name without the backlog folder prefix.
// These tests pin the fix: bare names resolve into the backlog when
// unique; paths that match nothing error and create nothing.

// toolFailure flattens the two ways a tool call reports failure (a
// Go-level error, or an isError result with the message in content)
// into one string; empty means the call succeeded.
func toolFailure(res *mcp.CallToolResult, err error) string {
	if err != nil {
		return err.Error()
	}
	if res != nil && res.IsError {
		return flattenContent(res)
	}
	return ""
}

// A bare file name (no backlog prefix) must resolve to the backlog's
// item — the status really changes, and no husk appears at the root.
func TestSetStatus_BareNameResolvesIntoBacklog(t *testing.T) {
	dir := t.TempDir()
	mustInitRepo(t, dir)
	mustWriteItem(t, dir, "Story", map[string]string{
		"status": "unstarted", "estimate": "3", "type": "feature",
	})
	priPath := filepath.Join(dir, "am-product", "_priority.md")
	if err := os.WriteFile(priPath, []byte("# Priority\n\n- [Story](Story.md)\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := callToolViaMemoryTransport(t, dir, "set_status", map[string]any{
		"path": "Story.md", "status": "started",
	}); err != nil {
		t.Fatalf("set_status with bare name: %v", err)
	}

	body, _ := os.ReadFile(filepath.Join(dir, "am-product", "Story.md"))
	if !strings.Contains(string(body), "status: started") {
		t.Fatalf("backlog item not updated; got:\n%s", body)
	}
	if _, err := os.Stat(filepath.Join(dir, "Story.md")); !os.IsNotExist(err) {
		t.Fatalf("husk created at repo root")
	}
}

// A path that matches nothing must error — and must not write a husk.
func TestSetStatus_UnknownPathErrorsAndCreatesNothing(t *testing.T) {
	dir := t.TempDir()
	mustInitRepo(t, dir)

	res, err := callToolViaMemoryTransport(t, dir, "set_status", map[string]any{
		"path": "No-such-story.md", "status": "started",
	})
	failure := toolFailure(res, err)
	if failure == "" {
		t.Fatalf("expected an error for an unknown item path")
	}
	if !strings.Contains(failure, "not found") {
		t.Fatalf("expected a not-found error; got: %s", failure)
	}
	for _, husk := range []string{
		filepath.Join(dir, "No-such-story.md"),
		filepath.Join(dir, "_priority.md"),
	} {
		if _, serr := os.Stat(husk); !os.IsNotExist(serr) {
			t.Fatalf("husk created: %s", husk)
		}
	}
}

// The same guard holds for the other writing tools — set_estimate as
// the representative (every handler goes through the shared resolver).
func TestSetEstimate_UnknownPathErrorsAndCreatesNothing(t *testing.T) {
	dir := t.TempDir()
	mustInitRepo(t, dir)

	res, err := callToolViaMemoryTransport(t, dir, "set_estimate", map[string]any{
		"path": "Ghost.md", "estimate": "3",
	})
	if toolFailure(res, err) == "" {
		t.Fatalf("expected an error for an unknown item path")
	}
	if _, serr := os.Stat(filepath.Join(dir, "Ghost.md")); !os.IsNotExist(serr) {
		t.Fatalf("husk created at repo root")
	}
}

// get_item must error on a missing item rather than report a blank one.
func TestGetItem_UnknownPathErrors(t *testing.T) {
	dir := t.TempDir()
	mustInitRepo(t, dir)

	res, err := callToolViaMemoryTransport(t, dir, "get_item", map[string]any{
		"path": "am-product/Missing.md",
	})
	if toolFailure(res, err) == "" {
		t.Fatalf("expected an error for a missing item")
	}
}

// A bare name that exists in two backlogs is ambiguous: error, name
// the cure (prefix with the backlog folder), and write nothing.
func TestResolve_AmbiguousBareNameErrors(t *testing.T) {
	dir := t.TempDir()
	mustInitRepo(t, dir)
	mustWriteItem(t, dir, "Story", map[string]string{
		"status": "unstarted", "type": "feature",
	})
	// A second backlog with the same item file name.
	if err := os.MkdirAll(filepath.Join(dir, "am-other"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "am-other.md"), []byte("# am-other\n"), 0644); err != nil {
		t.Fatal(err)
	}
	src, _ := os.ReadFile(filepath.Join(dir, "am-product", "Story.md"))
	if err := os.WriteFile(filepath.Join(dir, "am-other", "Story.md"), src, 0644); err != nil {
		t.Fatal(err)
	}

	res, err := callToolViaMemoryTransport(t, dir, "set_status", map[string]any{
		"path": "Story.md", "status": "started",
	})
	failure := toolFailure(res, err)
	if failure == "" {
		t.Fatalf("expected an ambiguity error")
	}
	if !strings.Contains(failure, "more than one backlog") {
		t.Fatalf("expected ambiguity message; got: %s", failure)
	}
}
