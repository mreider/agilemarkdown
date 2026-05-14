package backlog

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// AppendComment writes a new dated comment under "## Comments" in body.
// The comment is inserted inside the section (before any later heading)
// so the body stays parseable. The section is created when missing.
func AppendComment(body, author, text string) string {
	stamp := time.Now().UTC().Format("2006-01-02")
	header := fmt.Sprintf("@%s %s", strings.TrimPrefix(strings.TrimSpace(author), "@"), stamp)

	lines := strings.Split(body, "\n")
	startIdx := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if commentsTitleRe.MatchString(lines[i]) {
			startIdx = i + 1
			break
		}
	}
	if startIdx < 0 {
		trimmed := strings.TrimRight(body, "\n")
		if trimmed != "" {
			trimmed += "\n\n"
		}
		return trimmed + "## Comments\n\n" + header + "\n" + text + "\n"
	}
	end := len(lines)
	for i := startIdx; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "#") {
			end = i
			break
		}
	}
	insertAt := end
	for insertAt > startIdx && strings.TrimSpace(lines[insertAt-1]) == "" {
		insertAt--
	}
	out := make([]string, 0, len(lines)+3)
	out = append(out, lines[:insertAt]...)
	if insertAt > startIdx {
		out = append(out, "")
	}
	out = append(out, header, text)
	out = append(out, lines[insertAt:]...)
	return strings.Join(out, "\n")
}

type Commented interface {
	Comments() []*Comment
	UpdateComments(comments []*Comment) error
	Path() string
	Title() string
}

type Comment struct {
	Users   []string
	Text    []string
	rawText []string
	Closed  bool
	Unsent  bool
}

func (c *Comment) AddLine(line string) {
	c.Text = append(c.Text, line)

	c.rawText = append(c.rawText, line)
	i := len(c.rawText) - 1
	for i > 0 && strings.TrimSpace(c.rawText[i-1]) == "" {
		c.rawText[i-1], c.rawText[i] = c.rawText[i], c.rawText[i-1]
		i--
	}
}

func parseBodyComments(body string) []*Comment {
	lines := strings.Split(body, "\n")
	startIdx := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if commentsTitleRe.MatchString(lines[i]) {
			startIdx = i + 1
			break
		}
	}
	if startIdx == -1 {
		return nil
	}

	var comments []*Comment
	var comment *Comment
	for i := startIdx; i < len(lines); i++ {
		line := strings.TrimRightFunc(lines[i], unicode.IsSpace)
		if line == "" {
			if comment != nil {
				comment.rawText = append(comment.rawText, lines[i])
			}
			continue
		}
		if strings.HasPrefix(line, "#") {
			break
		}
		matches := commentRe.FindStringSubmatch(line)
		if len(matches) > 0 {
			if comment != nil {
				comments = append(comments, comment)
			}
			rawUsers := commentUserSeparatorRe.Split(matches[2], -1)
			seen := make(map[string]bool)
			users := make([]string, 0, len(rawUsers))
			for _, u := range rawUsers {
				u = strings.TrimPrefix(u, "@")
				u = strings.TrimSuffix(u, ".")
				if u == "" || seen[u] {
					continue
				}
				users = append(users, u)
				seen[u] = true
			}
			comment = &Comment{Users: users}
			if len(matches[1]) > 0 {
				comment.Closed = true
			}
			text := strings.TrimSpace(matches[4])
			if text != "" {
				comment.Text = append(comment.Text, text)
			}
			comment.rawText = append(comment.rawText, lines[i])
		} else if comment != nil {
			trimmed := strings.TrimSpace(line)
			low := strings.ToLower(trimmed)
			if strings.HasPrefix(low, "sent by ") {
				comment.Closed = true
			} else if strings.HasPrefix(low, "can't send by ") {
				comment.Unsent = true
			}
			comment.Text = append(comment.Text, trimmed)
			comment.rawText = append(comment.rawText, lines[i])
		}
	}
	if comment != nil {
		comments = append(comments, comment)
	}
	return comments
}

func updateBodyComments(body string, comments []*Comment) string {
	lines := strings.Split(body, "\n")
	startIdx := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if commentsTitleRe.MatchString(lines[i]) {
			startIdx = i + 1
			break
		}
	}
	if startIdx == -1 {
		return body
	}
	for startIdx < len(lines) && strings.TrimSpace(lines[startIdx]) == "" {
		startIdx++
	}
	finishIdx := startIdx
	for i := startIdx; i < len(lines); i++ {
		finishIdx = i
		if strings.HasPrefix(strings.TrimRightFunc(lines[i], unicode.IsSpace), "#") {
			break
		}
	}

	out := make([]string, 0, len(lines))
	out = append(out, lines[:startIdx]...)
	for _, c := range comments {
		out = append(out, c.rawText...)
	}
	out = append(out, lines[finishIdx:]...)
	return strings.Join(out, "\n")
}
