package backlog

import "fmt"

type MarkdownField struct {
	field string
	value string
}

func (f *MarkdownField) Lines() []string {
	return []string{fmt.Sprintf("%s: %s", f.field, f.value), "", ""}
}
