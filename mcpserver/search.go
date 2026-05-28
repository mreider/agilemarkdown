package mcpserver

import (
	"context"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SearchArgs struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty" jsonschema:"max results to return; default 20"`
}

type SearchHit struct {
	Path    string `json:"path"`
	Title   string `json:"title"`
	Status  string `json:"status,omitempty"`
	Type    string `json:"type,omitempty"`
	Snippet string `json:"snippet,omitempty"`
	Score   int    `json:"score"`
}

type SearchResult struct {
	Query string      `json:"query"`
	Hits  []SearchHit `json:"hits"`
	Count int         `json:"count"`
}

// searchTool implements a substring scorer across title, tags, path,
// and body. Case-insensitive. Hits ranked by score, ties broken by
// path so the order is stable across runs. Body matches contribute up
// to 5 points so a story with the query buried in long prose doesn't
// outrank one with the query in the title.
func searchTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, SearchArgs) (*mcp.CallToolResult, SearchResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args SearchArgs) (*mcp.CallToolResult, SearchResult, error) {
		q := strings.ToLower(strings.TrimSpace(args.Query))
		limit := args.Limit
		if limit <= 0 {
			limit = 20
		}
		if q == "" {
			return nil, SearchResult{Query: args.Query, Hits: nil, Count: 0}, nil
		}
		dirs, err := root.BacklogDirs()
		if err != nil {
			return nil, SearchResult{}, err
		}
		hits := make([]SearchHit, 0)
		for _, d := range dirs {
			bck, err := backlog.LoadBacklog(d)
			if err != nil {
				return nil, SearchResult{}, err
			}
			for _, it := range bck.ActiveItems() {
				score := 0
				title := strings.ToLower(it.Title())
				if strings.Contains(title, q) {
					score += 10
				}
				for _, t := range it.Tags() {
					if strings.Contains(strings.ToLower(t), q) {
						score += 5
					}
				}
				rel, _ := filepath.Rel(root.Root(), it.Path())
				if strings.Contains(strings.ToLower(rel), q) {
					score += 3
				}
				body := strings.ToLower(it.Body())
				if strings.Contains(body, q) {
					n := strings.Count(body, q)
					if n > 5 {
						n = 5
					}
					score += n
				}
				if score == 0 {
					continue
				}
				hits = append(hits, SearchHit{
					Path:    rel,
					Title:   it.Title(),
					Status:  it.Status(),
					Type:    it.Type(),
					Snippet: snippet(it.Body(), q),
					Score:   score,
				})
			}
		}
		sort.Slice(hits, func(i, j int) bool {
			if hits[i].Score != hits[j].Score {
				return hits[i].Score > hits[j].Score
			}
			return hits[i].Path < hits[j].Path
		})
		if len(hits) > limit {
			hits = hits[:limit]
		}
		return nil, SearchResult{Query: args.Query, Hits: hits, Count: len(hits)}, nil
	}
}

// snippet returns a short excerpt around the first body match, with
// ellipses on either side. Caller's responsibility to ensure q is
// already lowercased; body is matched case-insensitively here.
func snippet(body, q string) string {
	lower := strings.ToLower(body)
	idx := strings.Index(lower, q)
	if idx < 0 {
		return ""
	}
	const radius = 60
	start := idx - radius
	if start < 0 {
		start = 0
	}
	end := idx + len(q) + radius
	if end > len(body) {
		end = len(body)
	}
	out := strings.TrimSpace(body[start:end])
	out = strings.ReplaceAll(out, "\n", " ")
	if start > 0 {
		out = "…" + out
	}
	if end < len(body) {
		out += "…"
	}
	return out
}
