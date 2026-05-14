package mcpserver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mreider/agilemarkdown/actions"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	agreementsFileName = "team-agreements.md"
	learningsFileName  = "learnings.md"
)

type TeamAgreementsArgs struct {
	Set string `json:"set,omitempty" jsonschema:"if present, overwrite the agreements file with this content"`
}
type TeamAgreementsResult struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func teamAgreementsTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, TeamAgreementsArgs) (*mcp.CallToolResult, TeamAgreementsResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args TeamAgreementsArgs) (*mcp.CallToolResult, TeamAgreementsResult, error) {
		path := filepath.Join(root.Root(), agreementsFileName)
		if args.Set != "" {
			if err := os.WriteFile(path, []byte(args.Set), 0644); err != nil {
				return nil, TeamAgreementsResult{}, err
			}
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				stub := "# Team agreements\n\nNo agreements recorded yet.\n"
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: stub}},
				}, TeamAgreementsResult{Path: agreementsFileName, Content: stub}, nil
			}
			return nil, TeamAgreementsResult{}, err
		}
		text := string(raw)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, TeamAgreementsResult{Path: agreementsFileName, Content: text}, nil
	}
}

type RecordLearningArgs struct {
	Note string `json:"note" jsonschema:"one-line learning entry. The current date is prefixed automatically."`
}
type RecordLearningResult struct {
	Path  string `json:"path"`
	Entry string `json:"entry"`
}

func recordLearningTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, RecordLearningArgs) (*mcp.CallToolResult, RecordLearningResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args RecordLearningArgs) (*mcp.CallToolResult, RecordLearningResult, error) {
		note := strings.TrimSpace(args.Note)
		if note == "" {
			return nil, RecordLearningResult{}, fmt.Errorf("note is required")
		}
		path := filepath.Join(root.Root(), learningsFileName)
		entry := fmt.Sprintf("- %s: %s\n", time.Now().UTC().Format("2006-01-02"), note)

		var existing string
		if raw, err := os.ReadFile(path); err == nil {
			existing = string(raw)
		} else if !os.IsNotExist(err) {
			return nil, RecordLearningResult{}, err
		}
		if existing == "" {
			existing = "# Learnings\n\nA running log of one-line learnings from removal experiments, scratch refactors, retros, and surprises.\n\n"
		}
		if !strings.HasSuffix(existing, "\n") {
			existing += "\n"
		}
		out := existing + entry
		if err := os.WriteFile(path, []byte(out), 0644); err != nil {
			return nil, RecordLearningResult{}, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: entry}},
		}, RecordLearningResult{Path: learningsFileName, Entry: entry}, nil
	}
}

type CreateBacklogArgs struct {
	Name string `json:"name" jsonschema:"backlog folder name; becomes a top-level directory under the project root"`
}
type CreateBacklogResult struct {
	Backlog string `json:"backlog"`
}

func createBacklogTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, CreateBacklogArgs) (*mcp.CallToolResult, CreateBacklogResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args CreateBacklogArgs) (*mcp.CallToolResult, CreateBacklogResult, error) {
		name := strings.TrimSpace(args.Name)
		if name == "" {
			return nil, CreateBacklogResult{}, fmt.Errorf("name is required")
		}
		if err := actions.NewCreateBacklogAction(root.Root(), name).Execute(); err != nil {
			return nil, CreateBacklogResult{}, err
		}
		return nil, CreateBacklogResult{Backlog: utils.GetValidFileName(name)}, nil
	}
}

type ArchiveItemsArgs struct {
	Backlog string `json:"backlog"`
	Before  string `json:"before" jsonschema:"YYYY-MM-DD; items modified on or before this date are archived"`
}
type ArchiveItemsResult struct {
	Archived int `json:"archived"`
}

func archiveItemsTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, ArchiveItemsArgs) (*mcp.CallToolResult, ArchiveItemsResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ArchiveItemsArgs) (*mcp.CallToolResult, ArchiveItemsResult, error) {
		dir, err := resolveBacklogDir(root, args.Backlog)
		if err != nil {
			return nil, ArchiveItemsResult{}, err
		}
		before, err := time.Parse("2006-01-02", strings.TrimSpace(args.Before))
		if err != nil {
			return nil, ArchiveItemsResult{}, fmt.Errorf("before must be YYYY-MM-DD: %w", err)
		}

		bck, err := backlog.LoadBacklog(dir)
		if err != nil {
			return nil, ArchiveItemsResult{}, err
		}
		count := 0
		for _, item := range bck.ActiveItems() {
			if item.Modified().Before(before.Add(24 * time.Hour)) {
				item.SetArchived(true)
				if err := item.Save(); err != nil {
					return nil, ArchiveItemsResult{}, err
				}
				count++
			}
		}
		return nil, ArchiveItemsResult{Archived: count}, nil
	}
}

type SetHypothesisArgs struct {
	Path       string `json:"path" jsonschema:"item file path relative to project root"`
	Hypothesis string `json:"hypothesis" jsonschema:"what we expect to be true if this story works"`
}

func setHypothesisTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, SetHypothesisArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args SetHypothesisArgs) (*mcp.CallToolResult, OkResult, error) {
		path := filepath.Join(root.Root(), args.Path)
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return nil, OkResult{}, err
		}
		item.SetHypothesis(args.Hypothesis)
		item.SetModified(utils.GetCurrentTimestamp())
		if err := item.Save(); err != nil {
			return nil, OkResult{}, err
		}
		return nil, OkResult{OK: true}, nil
	}
}
