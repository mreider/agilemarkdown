// Package coach embeds and installs the four coach-mode template
// files (CLAUDE.md, AGENTS.md, copilot-instructions.md, cursor-coach.mdc).
// All four files share the same body. The drift check at
// tests/check-coach-projections.sh enforces that.
package coach

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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
	{"hooks/session-context.sh", ".claude/hooks/session-context.sh", 0o755},
	{"hooks/am-work-gate.sh", ".claude/hooks/am-work-gate.sh", 0o755},
	{"settings.json", ".claude/settings.json", 0o644},
}

// globalGateCommand is the hook command wired into the user's global
// settings; `~` is left literal so Claude Code expands it per-user.
const globalGateCommand = "bash ~/.claude/hooks/am-work-gate.sh"

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

// InstallGlobalGate installs the cwd-independent work-gate into the user's
// global Claude config under homeDir/.claude: it writes the gate script to
// hooks/am-work-gate.sh and merges a PreToolUse Edit/Write/NotebookEdit hook
// into settings.json. This is what makes "no edits without a started story"
// hold for sessions launched from outside the repo (e.g. a parent directory)
// — the per-repo coach gate only fires when the session runs from the repo.
//
// It returns the paths it changed. Idempotent: the script is rewritten only
// when its content drifts, and the settings hook is added only when absent.
// Non-am projects are unaffected at runtime — the script itself no-ops
// outside an am repo.
func InstallGlobalGate(homeDir string) ([]string, error) {
	var changed []string
	claudeDir := filepath.Join(homeDir, ".claude")
	hooksDir := filepath.Join(claudeDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return nil, err
	}

	// 1) The gate script (write when missing or out of date).
	data, err := fs.ReadFile(templatesFS, "hooks/am-work-gate.sh")
	if err != nil {
		return nil, err
	}
	scriptPath := filepath.Join(hooksDir, "am-work-gate.sh")
	if cur, _ := os.ReadFile(scriptPath); !bytes.Equal(cur, data) {
		if err := os.WriteFile(scriptPath, data, 0o755); err != nil {
			return nil, err
		}
		changed = append(changed, scriptPath)
	}

	// 2) Merge the hook into settings.json, preserving every other key.
	settingsPath := filepath.Join(claudeDir, "settings.json")
	root := map[string]interface{}{}
	if raw, err := os.ReadFile(settingsPath); err == nil && len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &root); err != nil {
			return changed, fmt.Errorf("parse %s: %w", settingsPath, err)
		}
	}
	if ensureGateHook(root) {
		out, err := json.MarshalIndent(root, "", "  ")
		if err != nil {
			return changed, err
		}
		if err := os.WriteFile(settingsPath, append(out, '\n'), 0o644); err != nil {
			return changed, err
		}
		changed = append(changed, settingsPath)
	}
	return changed, nil
}

// ensureGateHook adds PreToolUse Edit/Write/NotebookEdit entries pointing at
// the global gate to `root` if they are not already present. It returns
// whether `root` was modified.
func ensureGateHook(root map[string]interface{}) bool {
	hooks, _ := root["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = map[string]interface{}{}
		root["hooks"] = hooks
	}
	pre, _ := hooks["PreToolUse"].([]interface{})
	for _, e := range pre {
		m, _ := e.(map[string]interface{})
		hs, _ := m["hooks"].([]interface{})
		for _, h := range hs {
			hm, _ := h.(map[string]interface{})
			if c, _ := hm["command"].(string); strings.Contains(c, "am-work-gate.sh") {
				return false // already wired
			}
		}
	}
	for _, matcher := range []string{"Edit", "Write", "NotebookEdit"} {
		pre = append(pre, map[string]interface{}{
			"matcher": matcher,
			"hooks": []interface{}{
				map[string]interface{}{"type": "command", "command": globalGateCommand},
			},
		})
	}
	hooks["PreToolUse"] = pre
	return true
}
