package mcpserver

import (
	"context"
	"fmt"

	"github.com/mreider/agilemarkdown/backlog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SetIterationOverrideArgs struct {
	Number       int     `json:"number" jsonschema:"1-based iteration number"`
	TeamStrength float64 `json:"team_strength,omitempty" jsonschema:"0.0 excludes the iteration from velocity, 1.0 is full strength, >1.0 is allowed; pass -1 to leave unchanged"`
	LengthWeeks  int     `json:"length_weeks,omitempty" jsonschema:"override the iteration length in weeks; 0 leaves unchanged"`
	Unset        bool    `json:"unset,omitempty" jsonschema:"remove the override record entirely"`
}
type IterationOverrideResult struct {
	Number       int     `json:"number"`
	TeamStrength float64 `json:"team_strength"`
	LengthWeeks  int     `json:"length_weeks"`
	Cleared      bool    `json:"cleared,omitempty"`
}

func setIterationOverrideTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, SetIterationOverrideArgs) (*mcp.CallToolResult, IterationOverrideResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args SetIterationOverrideArgs) (*mcp.CallToolResult, IterationOverrideResult, error) {
		if args.Number <= 0 {
			return nil, IterationOverrideResult{}, fmt.Errorf("number must be positive")
		}
		overrides, err := backlog.LoadIterationOverrides(root.Root())
		if err != nil {
			return nil, IterationOverrideResult{}, err
		}
		if args.Unset {
			overrides.Clear(args.Number)
			if err := overrides.Save(root.Root()); err != nil {
				return nil, IterationOverrideResult{}, err
			}
			return nil, IterationOverrideResult{Number: args.Number, Cleared: true}, nil
		}
		strength := args.TeamStrength
		if strength < 0 {
			strength = -1 // sentinel: leave unchanged
		}
		overrides.Set(args.Number, strength, args.LengthWeeks)
		if err := overrides.Save(root.Root()); err != nil {
			return nil, IterationOverrideResult{}, err
		}
		rec := overrides.Find(args.Number)
		return nil, IterationOverrideResult{
			Number: rec.Number, TeamStrength: rec.TeamStrength, LengthWeeks: rec.LengthWeeks,
		}, nil
	}
}

type ListIterationOverridesArgs struct{}
type ListIterationOverridesResult struct {
	Overrides []IterationOverrideResult `json:"overrides"`
}

func listIterationOverridesTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, ListIterationOverridesArgs) (*mcp.CallToolResult, ListIterationOverridesResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, _ ListIterationOverridesArgs) (*mcp.CallToolResult, ListIterationOverridesResult, error) {
		overrides, err := backlog.LoadIterationOverrides(root.Root())
		if err != nil {
			return nil, ListIterationOverridesResult{}, err
		}
		out := make([]IterationOverrideResult, 0, len(overrides.Overrides))
		for _, o := range overrides.Overrides {
			out = append(out, IterationOverrideResult{
				Number: o.Number, TeamStrength: o.TeamStrength, LengthWeeks: o.LengthWeeks,
			})
		}
		return nil, ListIterationOverridesResult{Overrides: out}, nil
	}
}
