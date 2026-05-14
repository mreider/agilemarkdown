package mcpserver

import (
	"context"
	"path/filepath"
	"time"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type CycleTimeChartArgs struct {
	Backlog string `json:"backlog"`
}

func cycleTimeChartTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, CycleTimeChartArgs) (*mcp.CallToolResult, ChartResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args CycleTimeChartArgs) (*mcp.CallToolResult, ChartResult, error) {
		dir, err := resolveBacklogDir(root, args.Backlog)
		if err != nil {
			return nil, ChartResult{}, err
		}
		bck, err := backlog.LoadBacklog(dir)
		if err != nil {
			return nil, ChartResult{}, err
		}
		text := backlog.CycleTimeASCII(bck)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, ChartResult{ASCII: text}, nil
	}
}

type RejectionRateArgs struct {
	Backlog string `json:"backlog"`
}

type RejectionRateRow struct {
	Iteration int     `json:"iteration"`
	Start     string  `json:"start"`
	Accepted  int     `json:"accepted"`
	Rejected  int     `json:"rejected"`
	Percent   float64 `json:"percent"`
}

type RejectionRateResult struct {
	Rows  []RejectionRateRow `json:"rows"`
	ASCII string             `json:"ascii"`
}

func rejectionRateTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, RejectionRateArgs) (*mcp.CallToolResult, RejectionRateResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args RejectionRateArgs) (*mcp.CallToolResult, RejectionRateResult, error) {
		dir, err := resolveBacklogDir(root, args.Backlog)
		if err != nil {
			return nil, RejectionRateResult{}, err
		}
		cfg, err := config.LoadConfig(filepath.Join(root.Root(), ".am", "config.yaml"))
		if err != nil {
			return nil, RejectionRateResult{}, err
		}
		bck, err := backlog.LoadBacklog(dir)
		if err != nil {
			return nil, RejectionRateResult{}, err
		}
		now := time.Now()
		rows := backlog.RejectionRates(now, bck.AllItems(), cfg)
		out := make([]RejectionRateRow, 0, len(rows))
		for _, r := range rows {
			out = append(out, RejectionRateRow{
				Iteration: r.Iteration, Start: r.Start.Format("2006-01-02"),
				Accepted: r.Accepted, Rejected: r.Rejected, Percent: r.Percent,
			})
		}
		text := backlog.RejectionRateASCII(now, bck.AllItems(), cfg)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, RejectionRateResult{Rows: out, ASCII: text}, nil
	}
}
