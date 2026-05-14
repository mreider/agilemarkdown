package mcpserver

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// AcceptanceBulletRow is the JSON shape for one parsed acceptance
// bullet returned by list_acceptance and acceptance_prompt.
type AcceptanceBulletRow struct {
	Index     int    `json:"index"`
	State     string `json:"state" jsonschema:"open | claimed | verified"`
	Text      string `json:"text"`
	ClaimNote string `json:"claim_note,omitempty"`
}

type ListAcceptanceArgs struct {
	Path string `json:"path" jsonschema:"file path relative to project root"`
}

type ListAcceptanceResult struct {
	Path    string                `json:"path"`
	Bullets []AcceptanceBulletRow `json:"bullets"`
	Count   int                   `json:"count"`
}

func listAcceptanceTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, ListAcceptanceArgs) (*mcp.CallToolResult, ListAcceptanceResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ListAcceptanceArgs) (*mcp.CallToolResult, ListAcceptanceResult, error) {
		path := filepath.Join(root.Root(), args.Path)
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return nil, ListAcceptanceResult{}, err
		}
		bullets := backlog.ParseAcceptance(item.Body())
		out := make([]AcceptanceBulletRow, 0, len(bullets))
		for _, b := range bullets {
			out = append(out, AcceptanceBulletRow{
				Index:     b.Index,
				State:     string(b.State),
				Text:      b.Text,
				ClaimNote: b.ClaimNote,
			})
		}
		return nil, ListAcceptanceResult{Path: args.Path, Bullets: out, Count: len(out)}, nil
	}
}

type SetAcceptanceStateArgs struct {
	Path      string `json:"path"`
	Index     int    `json:"index" jsonschema:"1-based bullet index; indices are valid only against the body as it was when list_acceptance was called, so re-list after any writer call"`
	State     string `json:"state" jsonschema:"open | claimed | verified"`
	ClaimNote string `json:"claim_note,omitempty" jsonschema:"optional trailing claim note; ignored unless state=claimed"`
}

func setAcceptanceStateTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, SetAcceptanceStateArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args SetAcceptanceStateArgs) (*mcp.CallToolResult, OkResult, error) {
		state, err := parseAcceptanceState(args.State)
		if err != nil {
			return nil, OkResult{}, err
		}
		path := filepath.Join(root.Root(), args.Path)
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return nil, OkResult{}, err
		}
		body, err := backlog.SetAcceptanceState(item.Body(), args.Index, state, args.ClaimNote)
		if err != nil {
			return nil, OkResult{}, err
		}
		item.SetBody(body)
		item.SetModified(utils.GetCurrentTimestamp())
		if err := item.Save(); err != nil {
			return nil, OkResult{}, err
		}
		return nil, OkResult{OK: true}, nil
	}
}

type AppendAcceptanceBulletArgs struct {
	Path string `json:"path"`
	Text string `json:"text"`
}

func appendAcceptanceBulletTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, AppendAcceptanceBulletArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args AppendAcceptanceBulletArgs) (*mcp.CallToolResult, OkResult, error) {
		if args.Text == "" {
			return nil, OkResult{}, fmt.Errorf("text is required")
		}
		path := filepath.Join(root.Root(), args.Path)
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return nil, OkResult{}, err
		}
		item.SetBody(backlog.AppendAcceptanceBullet(item.Body(), args.Text))
		item.SetModified(utils.GetCurrentTimestamp())
		if err := item.Save(); err != nil {
			return nil, OkResult{}, err
		}
		return nil, OkResult{OK: true}, nil
	}
}

// parseAcceptanceState validates a state string from MCP args and maps
// it to the typed enum. Returns a friendly error when the input is not
// one of the three legal values.
func parseAcceptanceState(s string) (backlog.AcceptanceState, error) {
	switch s {
	case "open":
		return backlog.AcceptanceOpen, nil
	case "claimed":
		return backlog.AcceptanceClaimed, nil
	case "verified":
		return backlog.AcceptanceVerified, nil
	}
	return "", fmt.Errorf("state must be open, claimed, or verified (got %q)", s)
}

// bulletsToRows converts parser output to JSON rows. Used by
// acceptance_prompt to embed structured bullets alongside the legacy
// Verify []string field.
func bulletsToRows(bullets []backlog.AcceptanceBullet) []AcceptanceBulletRow {
	out := make([]AcceptanceBulletRow, 0, len(bullets))
	for _, b := range bullets {
		out = append(out, AcceptanceBulletRow{
			Index:     b.Index,
			State:     string(b.State),
			Text:      b.Text,
			ClaimNote: b.ClaimNote,
		})
	}
	return out
}

// bulletMarker renders one bullet's state as the two-character checkbox
// shown in prompt_text. The PM ceremony uses these to show progress at
// a glance: [ ] open, [~] claimed, [x] verified.
func bulletMarker(state backlog.AcceptanceState) string {
	switch state {
	case backlog.AcceptanceClaimed:
		return "[~]"
	case backlog.AcceptanceVerified:
		return "[x]"
	}
	return "[ ]"
}
