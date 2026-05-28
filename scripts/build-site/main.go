// build-site renders docs/tutorial/<chapter>/README.md into static
// HTML pages matching the existing agilemarkdown.com look. Each
// chapter becomes docs/tutorial-<NN>-<slug>.html; the chapter list
// lands at docs/tutorial.html (overwriting the old hand-crafted
// walkthrough page).
//
// Run: go run ./scripts/build-site
//
// Inputs:
//   docs/tutorial/<NN>-<slug>/README.md
//   docs/tutorial/<NN>-<slug>/images/*.png
//   docs/tutorial/<NN>-<slug>/cli/*.txt        (relative-pathed in the .md)
//   docs/tutorial/<NN>-<slug>/mcp/*.json
//
// Outputs:
//   docs/tutorial.html                          (index, with chapter cards)
//   docs/tutorial-<NN>-<slug>.html              (per-chapter)
//
// Images and other relative assets keep their source paths
// (docs/tutorial/<NN>/images/foo.png), the rendered HTML uses
// `./tutorial/<NN>/images/foo.png` because the output sits one
// directory above the markdown.
package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

type chapter struct {
	Dir       string // 04-current-sprint
	Num       string // 04
	Slug      string // current-sprint
	Title     string // "The current sprint"
	Output    string // tutorial-04-current-sprint.html
	BodyHTML  template.HTML
	IndexHTML template.HTML // shorter HTML for the index card
}

var titleRe = regexp.MustCompile(`(?m)^#\s+\d*\.?\s*(.+)$`)

func main() {
	repo := mustRepoRoot()
	tutorialRoot := filepath.Join(repo, "docs", "tutorial")
	chapters, err := loadChapters(tutorialRoot)
	if err != nil {
		fail(err)
	}
	for i := range chapters {
		if err := renderChapter(&chapters[i]); err != nil {
			fail(err)
		}
	}
	for _, c := range chapters {
		out := filepath.Join(repo, "docs", c.Output)
		if err := writeChapterPage(c, chapters, out); err != nil {
			fail(err)
		}
		fmt.Printf("  wrote %s\n", out)
	}
	indexPath := filepath.Join(repo, "docs", "tutorial.html")
	if err := writeIndexPage(chapters, indexPath); err != nil {
		fail(err)
	}
	fmt.Printf("  wrote %s\n", indexPath)
}

func loadChapters(root string) ([]chapter, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []chapter
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !chapterDirRe.MatchString(name) {
			continue
		}
		num := name[:2]
		slug := name[3:]
		title := titleFromReadme(filepath.Join(root, name, "README.md"))
		out = append(out, chapter{
			Dir:    name,
			Num:    num,
			Slug:   slug,
			Title:  title,
			Output: fmt.Sprintf("tutorial-%s-%s.html", num, slug),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Num < out[j].Num })
	return out, nil
}

var chapterDirRe = regexp.MustCompile(`^\d{2}-`)

func titleFromReadme(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	m := titleRe.FindSubmatch(raw)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(string(m[1]))
}

func renderChapter(c *chapter) error {
	mdPath := filepath.Join("docs", "tutorial", c.Dir, "README.md")
	raw, err := os.ReadFile(mdPath)
	if err != nil {
		return err
	}
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Table),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)
	var buf bytes.Buffer
	if err := md.Convert(raw, &buf); err != nil {
		return err
	}
	// Rewrite runs on the rendered HTML so it catches both raw HTML
	// href="..." attributes (which goldmark passes through unchanged)
	// AND markdown-style [text](url) links (which goldmark turns
	// into anchor attributes only after rendering). Doing this on
	// the markdown source missed every [..](path) link.
	c.BodyHTML = template.HTML(transformMoments(rewriteAssetLinks(buf.String(), c.Dir)))
	return nil
}

// transformMoments rewrites each rendered Board/CLI/MCP moment table
// into a tabbed switcher (the same .surface-shot widget the landing
// page uses). The chapter markdown keeps authoring moments as plain
// three-column tables, readable on GitHub, while the site collapses
// them into one-surface-at-a-time tabs, defaulting to VS Code. The
// cell contents (screenshot, CLI <pre>, MCP prose + <details>) are
// carried over verbatim; only the wrapper changes.
var (
	momentTableRe = regexp.MustCompile(`(?s)<table>\s*<tr><th>Board</th><th>CLI</th><th>MCP</th></tr>(.*?)</table>`)
	cellRe        = regexp.MustCompile(`(?s)<td[^>]*>(.*?)</td>`)
)

func transformMoments(htmlOut string) string {
	type surface struct{ key, label, content string }
	return momentTableRe.ReplaceAllStringFunc(htmlOut, func(match string) string {
		body := momentTableRe.FindStringSubmatch(match)[1]
		cells := cellRe.FindAllStringSubmatch(body, -1)
		if len(cells) != 3 {
			return match // not the expected three-cell shape; leave as-is
		}
		all := []surface{
			{"extension", "VS Code", cells[0][1]},
			{"cli", "Shell", cells[1][1]},
			{"claude", "MCP", cells[2][1]},
		}
		// Only surfaces with content get a tab. A moment with no board
		// screenshot (e.g. a CLI/MCP-only operation) must not open to a
		// blank VS Code panel, drop the empty tab and default to the
		// first surface that has something to show.
		var present []surface
		for _, s := range all {
			if strings.TrimSpace(s.content) != "" {
				present = append(present, s)
			}
		}
		if len(present) == 0 {
			return match
		}
		var tabs, panels strings.Builder
		for i, s := range present {
			selected, tabindex := "false", "-1"
			if i == 0 {
				selected, tabindex = "true", "0"
			}
			fmt.Fprintf(&tabs, `<button role="tab" aria-selected="%s" tabindex="%s" data-surface="%s">%s</button>`+"\n",
				selected, tabindex, s.key, s.label)
			fmt.Fprintf(&panels, `<div class="surface-panel" data-surface="%s" role="tabpanel">%s</div>`+"\n",
				s.key, s.content)
		}
		return `<figure class="surface-shot tut" data-active="` + present[0].key + `">
<div class="surface-tabs" role="tablist" aria-label="The same moment on each surface">
` + tabs.String() + `</div>
<div class="surface-panels">
` + panels.String() + `</div>
</figure>`
	})
}

// rewriteAssetLinks fixes every relative link in the rendered HTML
// so the generated chapter page (which sits in docs/) resolves:
//
//   images/foo.png       → tutorial/<dir>/images/foo.png
//   cli/foo.txt          → tutorial/<dir>/cli/foo.txt
//   mcp/foo.json         → tutorial/<dir>/mcp/foo.json
//   ../NN-slug/          → tutorial-NN-slug.html
//   ../NN-slug           → tutorial-NN-slug.html
//   ../NN-slug/#anchor   → tutorial-NN-slug.html#anchor
//
// Only attributes that start with one of those prefixes are
// touched; absolute URLs, anchors, and other paths pass through.
func rewriteAssetLinks(htmlOut, chapterDir string) string {
	rules := []struct {
		pattern, replace string
	}{
		{`(src|href)="images/`, fmt.Sprintf(`$1="tutorial/%s/images/`, chapterDir)},
		{`(src|href)="cli/`, fmt.Sprintf(`$1="tutorial/%s/cli/`, chapterDir)},
		{`(src|href)="mcp/`, fmt.Sprintf(`$1="tutorial/%s/mcp/`, chapterDir)},
		{`href="\.\./(\d{2})-([a-z0-9-]+)/?(#[^"]*)?"`, `href="tutorial-$1-$2.html$3"`},
	}
	for _, r := range rules {
		re := regexp.MustCompile(r.pattern)
		htmlOut = re.ReplaceAllString(htmlOut, r.replace)
	}
	return htmlOut
}

func writeChapterPage(c chapter, all []chapter, outPath string) error {
	var prev, next *chapter
	for i := range all {
		if all[i].Dir == c.Dir {
			if i > 0 {
				prev = &all[i-1]
			}
			if i+1 < len(all) {
				next = &all[i+1]
			}
			break
		}
	}
	data := struct {
		Title    string
		Num      string
		BodyHTML template.HTML
		Prev     *chapter
		Next     *chapter
	}{c.Title, c.Num, c.BodyHTML, prev, next}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return chapterTpl.Execute(f, data)
}

func writeIndexPage(chapters []chapter, outPath string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return indexTpl.Execute(f, map[string]any{"Chapters": chapters})
}

func mustRepoRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		fail(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(cwd, "go.mod")); err == nil {
			return cwd
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			fail(fmt.Errorf("not inside a Go module"))
		}
		cwd = parent
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "build-site:", err)
	os.Exit(1)
}
