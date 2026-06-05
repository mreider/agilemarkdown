package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// expectedTools is the parity contract between the CLI and MCP. Any new
// CLI verb that mutates state must have an MCP equivalent listed here.
// Adding a CLI command without updating this list (and an `AddTool` in
// Run) trips this test.
var expectedTools = []string{
	"add_comment",
	"add_task",
	"archive_items",
	"block_item",
	"burnup_chart",
	"change_tag",
	"coach_check",
	"create_backlog",
	"create_item",
	"cycle_time_chart",
	"dashboard",
	"delete_tag",
	"done_list",
	"epic_progress",
	"cumulative_flow",
	"get_comments",
	"get_item",
	"icebox_list",
	"inception_doc",
	"iteration_fit",
	"iteration_view",
	"journal",
	"list_backlogs",
	"list_items",
	"list_iteration_overrides",
	"list_tasks",
	"move_to_icebox",
	"move_to_priority",
	"next_item",
	"priority_list",
	"rank_item",
	"record_learning",
	"reject_item",
	"rejection_rate",
	"search",
	"set_assigned",
	"set_description",
	"set_epic",
	"set_estimate",
	"set_hypothesis",
	"set_iteration_override",
	"set_mode",
	"set_status",
	"set_tags",
	"set_task_done",
	"sprint_plan",
	"sync",
	"team_agreements",
	"timeline_chart",
	"type_mix",
	"unblock_item",
	"validate",
	"velocity_chart",
	"velocity_history",
}

// TestToolParity boots the server in-memory and asserts that the
// advertised tool set exactly matches expectedTools. If a tool is added
// to the server, this test will catch a missing entry here. If a CLI
// verb is added without an MCP tool, the developer is forced to update
// both.
func TestToolParity(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".am"), 0755); err != nil {
		t.Fatal(err)
	}
	// minimal repo layout so Run() does not blow up loading config
	if err := os.WriteFile(filepath.Join(dir, ".am", "config.yaml"), []byte("estimation:\n  scale: fibonacci\niteration:\n  length_weeks: 1\nvelocity:\n  strategy: rolling\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := listToolsViaMemoryTransport(t, dir)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	sort.Strings(got)

	want := append([]string(nil), expectedTools...)
	sort.Strings(want)

	if len(got) != len(want) {
		t.Errorf("tool count mismatch:\n  got:  %v\n  want: %v", got, want)
		t.FailNow()
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("tool[%d] = %q want %q (full got=%v want=%v)", i, got[i], want[i], got, want)
		}
	}
}

// TestChartASCIIAccurateFromFixture builds a deterministic mini-repo
// with known timestamps and asserts that the MCP chart tools return
// ASCII text whose visible counts match the fixture.
//
// The point is to catch silent regressions where the structured rows
// stay correct but the ASCII layer drifts. Each assertion targets a
// specific number in the output.
func TestChartASCIIAccurateFromFixture(t *testing.T) {
	dir := t.TempDir()
	mustInitRepo(t, dir)

	// Two accepted stories of 3 + 5 pts in the current iteration,
	// one started, one unstarted. CFD across last 7 days will see:
	//   - 2 accepted by today
	//   - 1 in-flight by today
	//   - 1 backlog by today
	// Burnup over the current iteration: scope = 8, done = 8.
	now := time.Now().UTC()
	yday := now.AddDate(0, 0, -1).Format("2006-01-02T15:04:05Z")
	twoDays := now.AddDate(0, 0, -2).Format("2006-01-02T15:04:05Z")
	threeDays := now.AddDate(0, 0, -3).Format("2006-01-02T15:04:05Z")

	mustWriteItem(t, dir, "alpha", map[string]string{
		"status": "accepted", "type": "feature", "estimate": "3",
		"created": threeDays, "started": twoDays,
		"finished": yday, "delivered": yday, "accepted": yday,
	})
	mustWriteItem(t, dir, "beta", map[string]string{
		"status": "accepted", "type": "feature", "estimate": "5",
		"created": threeDays, "started": twoDays,
		"finished": yday, "delivered": yday, "accepted": yday,
	})
	mustWriteItem(t, dir, "gamma", map[string]string{
		"status": "started", "type": "feature", "estimate": "2",
		"created": threeDays, "started": twoDays,
	})
	mustWriteItem(t, dir, "delta", map[string]string{
		"status": "unstarted", "type": "feature", "estimate": "1",
		"created": threeDays,
	})

	// ---- cumulative_flow ----
	res, err := callToolViaMemoryTransport(t, dir, "cumulative_flow", map[string]any{"days": 7})
	if err != nil {
		t.Fatalf("cumulative_flow: %v", err)
	}
	text := flattenContent(res)
	if !strings.Contains(text, "Cumulative flow") {
		t.Fatalf("cumulative_flow ASCII missing title:\n%s", text)
	}
	// The fixture has 4 stories. The CFD window is [now-7, now)
	// exclusive on the right, so the last row is yesterday. By
	// yesterday: 2 accepted, 1 in-flight, 1 backlog. Format from
	// CFDASCII: "  <date>    2     1     1".
	lastRowDate := now.AddDate(0, 0, -1).Format("2006-01-02")
	wantRow := fmt.Sprintf("%s    2     1     1", lastRowDate)
	if !strings.Contains(text, wantRow) {
		t.Errorf("cumulative_flow ASCII missing last-day row %q:\n%s", wantRow, text)
	}
	if !strings.Contains(text, "legend: A=accepted") {
		t.Errorf("cumulative_flow ASCII missing legend:\n%s", text)
	}

	// ---- burnup_chart ----
	res, err = callToolViaMemoryTransport(t, dir, "burnup_chart", map[string]any{"backlog": "am-product", "offset": 0})
	if err != nil {
		t.Fatalf("burnup_chart: %v", err)
	}
	text = flattenContent(res)
	if !strings.Contains(text, "Burnup ") {
		t.Fatalf("burnup_chart ASCII missing title:\n%s", text)
	}
	// All four items existed when the iteration started, so scope is
	// 3+5+2+1 = 11 (release type would skip but none here). Two
	// accepted by yesterday => done = 8.
	if !strings.Contains(text, "8.0") || !strings.Contains(text, "11.0") {
		t.Errorf("burnup_chart ASCII missing the scope/done numbers (want 8.0 and 11.0):\n%s", text)
	}

	// ---- type_mix ----
	res, err = callToolViaMemoryTransport(t, dir, "type_mix", map[string]any{})
	if err != nil {
		t.Fatalf("type_mix: %v", err)
	}
	text = flattenContent(res)
	if !strings.Contains(text, "Story type mix") {
		t.Fatalf("type_mix ASCII missing title:\n%s", text)
	}
	// Two accepted features in this window. The ASCII bar row reads
	// "feature  <bar>   2  (X%)". Just assert "feature" and " 2" are
	// adjacent enough to be part of the same row.
	if !strings.Contains(text, "feature") {
		t.Errorf("type_mix ASCII missing feature row:\n%s", text)
	}
	if !strings.Contains(text, "2 accepted") {
		t.Errorf("type_mix ASCII missing accepted total (want 2):\n%s", text)
	}
}

// TestVelocityChartTool drives the velocity_chart tool end-to-end so the
// CI catches breakage in the chart pipeline (config loading, item sweep,
// ASCII rendering).
func TestVelocityChartTool(t *testing.T) {
	dir := t.TempDir()
	mustInitRepo(t, dir)

	res, err := callToolViaMemoryTransport(t, dir, "velocity_chart", map[string]any{
		"backlog":         "am-product",
		"iteration_count": 4,
	})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if res == nil || len(res.Content) == 0 {
		t.Fatalf("empty content")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok || tc.Text == "" {
		t.Fatalf("expected non-empty text content; got %T", res.Content[0])
	}
}

// TestReadOnlyToolsSmoke calls each read-only tool against an empty
// repo and asserts the wire-protocol round trip succeeds. Catches arg
// shape mismatches and panics; not a behavioral assertion. Add an
// entry when a new read-only tool ships.
func TestReadOnlyToolsSmoke(t *testing.T) {
	dir := t.TempDir()
	mustInitRepo(t, dir)

	cases := []struct {
		tool string
		args map[string]any
	}{
		{"list_backlogs", nil},
		{"list_items", map[string]any{"backlog": "am-product"}},
		{"next_item", map[string]any{}},
		{"iteration_view", map[string]any{"backlog": "am-product"}},
		{"type_mix", map[string]any{}},
		{"velocity_history", map[string]any{"backlog": "am-product"}},
		{"cycle_time_chart", map[string]any{"backlog": "am-product"}},
		{"burnup_chart", map[string]any{"backlog": "am-product"}},
		{"cumulative_flow", map[string]any{"days": 7}},
		{"search", map[string]any{"query": "no-such-thing"}},
		{"validate", map[string]any{}},
		{"timeline_chart", map[string]any{"tag": "milestone"}},
		{"epic_progress", map[string]any{"slug": "no-such-epic"}},
	}

	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			res, err := callToolViaMemoryTransport(t, dir, tc.tool, tc.args)
			if err != nil {
				t.Fatalf("call %s: %v", tc.tool, err)
			}
			if res == nil {
				t.Fatalf("nil result")
			}
		})
	}
}

func TestMoveToIceboxRefusesActiveStory(t *testing.T) {
	dir := t.TempDir()
	mustInitRepo(t, dir)
	mustWriteItem(t, dir, "Login-flow", map[string]string{
		"status":   "started",
		"estimate": "3",
		"type":     "feature",
	})

	// Seed _priority.md directly so this test focuses on the guard,
	// not on sync wiring (which is exercised separately).
	priFile := filepath.Join(dir, "am-product", "_priority.md")
	if err := os.WriteFile(priFile,
		[]byte("# Priority\n\n- [Login-flow](Login-flow.md)\n"), 0644); err != nil {
		t.Fatal(err)
	}

	res, err := callToolViaMemoryTransport(t, dir, "move_to_icebox", map[string]any{
		"backlog":   "am-product",
		"item_path": "Login-flow.md",
	})
	if err == nil && (res == nil || !res.IsError) {
		t.Fatalf("expected move_to_icebox to refuse a started story; got res=%+v err=%v", res, err)
	}
	msg := ""
	if err != nil {
		msg = err.Error()
	} else if res != nil {
		msg = flattenContent(res)
	}
	if !strings.Contains(msg, "started") {
		t.Fatalf("expected error to mention status; got %q", msg)
	}
}

// extractItemPath pulls the path from a create_item result. The
// MCP SDK serializes the typed return value into StructuredContent
// as a map, so we read the "path" key directly without binding to
// the CreateItemResult struct.
func extractItemPath(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if res == nil {
		return ""
	}
	if sc, ok := res.StructuredContent.(map[string]any); ok {
		if p, ok := sc["path"].(string); ok {
			return p
		}
	}
	return ""
}

// flattenContent concatenates every TextContent block in the result.
func flattenContent(res *mcp.CallToolResult) string {
	if res == nil {
		return ""
	}
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// listToolsViaMemoryTransport spins up the server in-process via an
// in-memory transport pair, calls tools/list, and returns tool names.
func listToolsViaMemoryTransport(t *testing.T, rootDir string) ([]string, error) {
	t.Helper()
	cs := connectServer(t, rootDir)
	defer cs.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var names []string
	for tool, err := range cs.Tools(ctx, nil) {
		if err != nil {
			return nil, err
		}
		names = append(names, tool.Name)
	}
	return names, nil
}

// callToolViaMemoryTransport calls a single tool and returns the result.
func callToolViaMemoryTransport(t *testing.T, rootDir, toolName string, args map[string]any) (*mcp.CallToolResult, error) {
	t.Helper()
	cs := connectServer(t, rootDir)
	defer cs.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rawArgs, _ := json.Marshal(args)
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: json.RawMessage(rawArgs),
	})
	return res, err
}

// connectServer starts a server bound to rootDir on one end of an
// in-memory transport pair and returns a connected client session.
func connectServer(t *testing.T, rootDir string) *mcp.ClientSession {
	t.Helper()

	abs, err := filepath.Abs(rootDir)
	if err != nil {
		t.Fatal(err)
	}

	srvT, cliT := mcp.NewInMemoryTransports()

	// Build the server identically to Run() but bind to the in-memory
	// transport instead of stdio. Keep this in sync with Run().
	srv := buildServer(abs, "test")

	go func() {
		if err := srv.Run(context.Background(), srvT); err != nil {
			// Server exits when client disconnects; that's not a test failure.
			_ = err
		}
	}()

	c := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	cs, err := c.Connect(context.Background(), cliT, nil)
	if err != nil {
		t.Fatal(err)
	}
	return cs
}

func mustInitRepo(t *testing.T, dir string) {
	t.Helper()
	// Backlog folders now require the `am-` prefix on disk. The
	// fixture creates `am-product/` with a matching `am-product.md`
	// overview so resolveBacklogDir picks it up. Tests query by the
	// prefixed name; pre-prefix names like "product" no longer
	// resolve.
	if err := os.MkdirAll(filepath.Join(dir, "am-product"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "am-product.md"), []byte("# am-product\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".am"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".am", "config.yaml"),
		[]byte("estimation:\n  scale: fibonacci\niteration:\n  length_weeks: 1\nvelocity:\n  strategy: rolling\n  lookback: 3\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

// mustWriteItem drops a deterministic story file under product/ for
// the chart-accuracy tests. Caller passes the timestamps it wants to
// see reflected in CFD/Burnup/TypeMix output; the test then calls the
// MCP tool and asserts the ASCII contains expected counts.
func mustWriteItem(t *testing.T, dir, basename string, frontmatter map[string]string) {
	t.Helper()
	path := filepath.Join(dir, "am-product", basename+".md")
	body := "---\n"
	body += "title: " + basename + "\n"
	for k, v := range frontmatter {
		body += k + ": " + v + "\n"
	}
	body += "---\n\nbody.\n"
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}
