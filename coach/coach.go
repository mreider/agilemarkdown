// Package coach embeds and installs the four coach-mode template
// files (CLAUDE.md, AGENTS.md, copilot-instructions.md, cursor-coach.mdc).
// All four files share the same body. The drift check at
// tests/check-coach-projections.sh enforces that.
package coach

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/mreider/agilemarkdown/git"
)

//go:embed CLAUDE.md AGENTS.md copilot-instructions.md cursor-coach.mdc agilemarkdown-coach.md skills hooks settings.json
var templatesFS embed.FS

type installEntry struct {
	source string
	target string
	mode   os.FileMode
}

// install maps each embedded template to its target path inside the
// user's repo (relative to the project root). Order is fixed for
// stable output. Mode 0o755 marks executable scripts.
var install = []installEntry{
	// Canonical coach body. Claude Code's CLAUDE.md @-imports this; the
	// other agent conventions ship the same body inline.
	{"agilemarkdown-coach.md", ".claude/agilemarkdown-coach.md", 0o644},
	// CLAUDE.md is a thin @-import so a user's existing CLAUDE.md, if
	// present, is not overwritten. When missing, the projection installs
	// a one-line file that pulls in the canonical body.
	{"CLAUDE.md", "CLAUDE.md", 0o644},
	{"AGENTS.md", "AGENTS.md", 0o644},
	{"copilot-instructions.md", ".github/copilot-instructions.md", 0o644},
	{"cursor-coach.mdc", ".cursor/rules/coach.mdc", 0o644},
	{"skills/am-accept/SKILL.md", ".claude/skills/am-accept/SKILL.md", 0o644},
	{"skills/am-align/SKILL.md", ".claude/skills/am-align/SKILL.md", 0o644},
	{"skills/am-decompose/SKILL.md", ".claude/skills/am-decompose/SKILL.md", 0o644},
	{"skills/am-inception/SKILL.md", ".claude/skills/am-inception/SKILL.md", 0o644},
	{"skills/am-plan/SKILL.md", ".claude/skills/am-plan/SKILL.md", 0o644},
	{"skills/am-retro/SKILL.md", ".claude/skills/am-retro/SKILL.md", 0o644},
	{"hooks/coach-gate.sh", ".claude/hooks/coach-gate.sh", 0o755},
	{"settings.json", ".claude/settings.json", 0o644},
}

// InstallTemplates writes the four coach-mode templates into rootDir
// when missing. Returns the list of paths written (relative to
// rootDir). Idempotent: existing files are never overwritten.
//
// Newly written files are staged via `git add` if a repo is present.
// The caller commits.
func InstallTemplates(rootDir string) ([]string, error) {
	var written []string
	for _, t := range install {
		dst := filepath.Join(rootDir, t.target)
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return written, err
		}
		data, err := fs.ReadFile(templatesFS, t.source)
		if err != nil {
			return written, err
		}
		mode := t.mode
		if mode == 0 {
			mode = 0o644
		}
		if err := os.WriteFile(dst, data, mode); err != nil {
			return written, err
		}
		_ = git.Add(dst)
		written = append(written, t.target)
	}
	return written, nil
}
