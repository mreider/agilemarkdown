package backlog

import (
	"fmt"
	"regexp"
	"strings"
)

// Task is a checkbox item parsed from a "## Tasks" section in an item body.
// Index is 1-based and stable for the lifetime of one body parse.
type Task struct {
	Index int
	Done  bool
	Text  string
}

var (
	tasksHeadingRe = regexp.MustCompile(`(?i)^#{1,3}\s+Tasks\s*$`)
	taskLineRe     = regexp.MustCompile(`^\s*[-*]\s+\[( |x|X)\]\s+(.+?)\s*$`)
)

// ParseTasks scans body for a "## Tasks" heading and returns the checkbox
// items underneath it. Returns nil when no Tasks section is present.
func ParseTasks(body string) []Task {
	lines := strings.Split(body, "\n")
	start := tasksSectionStart(lines)
	if start < 0 {
		return nil
	}
	var out []Task
	idx := 0
	for i := start; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimRight(line, " \t\r")
		if strings.HasPrefix(trimmed, "#") {
			break
		}
		m := taskLineRe.FindStringSubmatch(trimmed)
		if m == nil {
			if strings.TrimSpace(trimmed) != "" && len(out) > 0 {
				continue
			}
			continue
		}
		idx++
		out = append(out, Task{Index: idx, Done: m[1] == "x" || m[1] == "X", Text: m[2]})
	}
	return out
}

// AppendTask returns body with `text` appended as a new unchecked task. If
// no Tasks section exists yet, one is created at the end of the body.
func AppendTask(body, text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return body
	}
	lines := strings.Split(body, "\n")
	start := tasksSectionStart(lines)
	if start < 0 {
		// Append a fresh section at the end.
		body = strings.TrimRight(body, "\n")
		if body != "" {
			body += "\n\n"
		}
		body += "## Tasks\n\n- [ ] " + text + "\n"
		return body
	}
	// Find insertion point: after the last task (or first non-task line).
	insertAt := start
	for i := start; i < len(lines); i++ {
		trimmed := strings.TrimRight(lines[i], " \t\r")
		if strings.HasPrefix(trimmed, "#") {
			break
		}
		if taskLineRe.MatchString(trimmed) || strings.TrimSpace(trimmed) == "" {
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

// SetTaskDone flips the checkbox state of the task at 1-based `index` to
// `done`. Returns an error if the index is out of range. Body is returned
// unchanged on error.
func SetTaskDone(body string, index int, done bool) (string, error) {
	if index < 1 {
		return body, fmt.Errorf("task index must be 1-based; got %d", index)
	}
	lines := strings.Split(body, "\n")
	start := tasksSectionStart(lines)
	if start < 0 {
		return body, fmt.Errorf("no Tasks section in item body")
	}
	cur := 0
	for i := start; i < len(lines); i++ {
		trimmed := strings.TrimRight(lines[i], " \t\r")
		if strings.HasPrefix(trimmed, "#") {
			break
		}
		m := taskLineRe.FindStringSubmatchIndex(trimmed)
		if m == nil {
			continue
		}
		cur++
		if cur == index {
			marker := " "
			if done {
				marker = "x"
			}
			// rewrite line preserving leading whitespace
			text := trimmed[m[4]:m[5]]
			leading := lines[i][:strings.Index(lines[i], "[")]
			lines[i] = leading + "[" + marker + "] " + text
			return strings.Join(lines, "\n"), nil
		}
	}
	return body, fmt.Errorf("task %d not found", index)
}

func tasksSectionStart(lines []string) int {
	for i := len(lines) - 1; i >= 0; i-- {
		if tasksHeadingRe.MatchString(lines[i]) {
			return i + 1
		}
	}
	return -1
}
