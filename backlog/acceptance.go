package backlog

import (
	"fmt"
	"regexp"
	"strings"
)

// AcceptanceState is the state of one acceptance bullet.
type AcceptanceState string

const (
	AcceptanceOpen     AcceptanceState = "open"
	AcceptanceClaimed  AcceptanceState = "claimed"
	AcceptanceVerified AcceptanceState = "verified"
)

// AcceptanceBullet is one parsed bullet from the body's "## Acceptance"
// section. Index is 1-based and stable for one parse only; re-parse
// after any writer call before reusing an index.
type AcceptanceBullet struct {
	Index     int
	State     AcceptanceState
	Text      string
	ClaimNote string
}

var (
	acceptanceLineRe = regexp.MustCompile(`^\s*[-*]\s+\[( |~|x|X)\]\s+(.+?)\s*$`)
	acceptanceLegacyRe = regexp.MustCompile(`^\s*[-*]\s+(.+?)\s*$`)
	claimNoteRe        = regexp.MustCompile(`\s*<!--\s*claim:\s*(.+?)\s*-->\s*$`)
)

// ParseAcceptance scans body for a "## Acceptance" section and returns
// the bullets underneath it. Returns nil when no section is present.
// Legacy bare bullets parse as AcceptanceOpen.
func ParseAcceptance(body string) []AcceptanceBullet {
	lines := strings.Split(body, "\n")
	start := acceptanceSectionStart(lines)
	if start < 0 {
		return nil
	}
	var out []AcceptanceBullet
	idx := 0
	for i := start; i < len(lines); i++ {
		trimmed := strings.TrimRight(lines[i], " \t\r")
		if strings.HasPrefix(strings.TrimSpace(trimmed), "#") {
			break
		}
		state, text, claim, ok := parseAcceptanceLine(trimmed)
		if !ok {
			continue
		}
		idx++
		out = append(out, AcceptanceBullet{
			Index:     idx,
			State:     state,
			Text:      text,
			ClaimNote: claim,
		})
	}
	return out
}

// AcceptanceBulletTexts returns just the bullet text strings in order.
// Replaces the legacy free-text helpers and preserves their contract.
func AcceptanceBulletTexts(body string) []string {
	bullets := ParseAcceptance(body)
	if bullets == nil {
		return nil
	}
	out := make([]string, 0, len(bullets))
	for _, b := range bullets {
		out = append(out, b.Text)
	}
	return out
}

// SetAcceptanceState rewrites one bullet's checkbox marker and optional
// trailing claim note. index is 1-based.
func SetAcceptanceState(body string, index int, state AcceptanceState, claimNote string) (string, error) {
	if index < 1 {
		return body, fmt.Errorf("acceptance bullet index must be 1-based; got %d", index)
	}
	marker, ok := acceptanceMarker(state)
	if !ok {
		return body, fmt.Errorf("invalid acceptance state %q (want open|claimed|verified)", state)
	}
	lines := strings.Split(body, "\n")
	start := acceptanceSectionStart(lines)
	if start < 0 {
		return body, fmt.Errorf("no Acceptance section in item body")
	}
	cur := 0
	for i := start; i < len(lines); i++ {
		trimmed := strings.TrimRight(lines[i], " \t\r")
		if strings.HasPrefix(strings.TrimSpace(trimmed), "#") {
			break
		}
		_, text, _, ok := parseAcceptanceLine(trimmed)
		if !ok {
			continue
		}
		cur++
		if cur != index {
			continue
		}
		leading := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " \t"))]
		newLine := leading + "- [" + marker + "] " + text
		if state == AcceptanceClaimed && strings.TrimSpace(claimNote) != "" {
			newLine += " <!-- claim: " + strings.TrimSpace(claimNote) + " -->"
		}
		lines[i] = newLine
		return strings.Join(lines, "\n"), nil
	}
	return body, fmt.Errorf("acceptance bullet %d not found", index)
}

// AppendAcceptanceBullet adds `text` as a new open bullet under the
// existing "## Acceptance" section. If no section exists, one is
// created at the end of the body.
func AppendAcceptanceBullet(body, text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return body
	}
	lines := strings.Split(body, "\n")
	start := acceptanceSectionStart(lines)
	if start < 0 {
		body = strings.TrimRight(body, "\n")
		if body != "" {
			body += "\n\n"
		}
		body += "## Acceptance\n\n- [ ] " + text + "\n"
		return body
	}
	insertAt := start
	for i := start; i < len(lines); i++ {
		trimmed := strings.TrimRight(lines[i], " \t\r")
		if strings.HasPrefix(strings.TrimSpace(trimmed), "#") {
			break
		}
		if _, _, _, ok := parseAcceptanceLine(trimmed); ok {
			insertAt = i + 1
			continue
		}
		if strings.TrimSpace(trimmed) == "" {
			insertAt = i + 1
			continue
		}
		insertAt = i + 1
	}
	newLine := "- [ ] " + text
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:insertAt]...)
	out = append(out, newLine)
	out = append(out, lines[insertAt:]...)
	return strings.Join(out, "\n")
}

// acceptanceSectionStart returns the line index just after the first
// "## Acceptance" heading, or -1 when no section exists. The match
// mirrors the legacy helpers: any line starting with "##" whose
// lowercase contains "acceptance".
func acceptanceSectionStart(lines []string) int {
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "##") && strings.Contains(strings.ToLower(t), "acceptance") {
			return i + 1
		}
	}
	return -1
}

// parseAcceptanceLine returns the state, text, and claim note (if any)
// of one bullet line, or ok=false when the line is not a bullet.
func parseAcceptanceLine(line string) (state AcceptanceState, text string, claim string, ok bool) {
	if m := acceptanceLineRe.FindStringSubmatch(line); m != nil {
		state = stateFromMarker(m[1])
		text = m[2]
		if cm := claimNoteRe.FindStringSubmatch(text); cm != nil {
			claim = cm[1]
			text = strings.TrimRight(claimNoteRe.ReplaceAllString(text, ""), " \t")
		}
		return state, text, claim, true
	}
	if m := acceptanceLegacyRe.FindStringSubmatch(line); m != nil {
		text = m[1]
		return AcceptanceOpen, text, "", true
	}
	return "", "", "", false
}

func stateFromMarker(m string) AcceptanceState {
	switch m {
	case "x", "X":
		return AcceptanceVerified
	case "~":
		return AcceptanceClaimed
	default:
		return AcceptanceOpen
	}
}

func acceptanceMarker(s AcceptanceState) (string, bool) {
	switch s {
	case AcceptanceOpen:
		return " ", true
	case AcceptanceClaimed:
		return "~", true
	case AcceptanceVerified:
		return "x", true
	}
	return "", false
}
