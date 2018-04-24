package backlog

import "fmt"

type MarkdownGroup struct {
	content *MarkdownContent
	title   string
	lines   []string
}

func (g *MarkdownGroup) Title() string {
	return g.title
}

func (g *MarkdownGroup) Count() int {
	return len(g.lines)
}

func (g *MarkdownGroup) Line(index int) string {
	return g.lines[index]
}

func (g *MarkdownGroup) SetLine(index int, line string) {
	g.lines[index] = line
	g.content.markDirty()
}

func (g *MarkdownGroup) AddLine(line string) {
	g.lines = append(g.lines, line)
	g.content.markDirty()
}

func (g *MarkdownGroup) DeleteLine(index int) {
	copy(g.lines[index:], g.lines[index+1:])
	g.lines = g.lines[:len(g.lines)-1]
	g.content.markDirty()
}

func (g *MarkdownGroup) Lines() []string {
	result := make([]string, 0, (1+len(g.lines))*2+1)
	result = append(result, fmt.Sprintf("%s%s", GroupTitlePrefix, g.title), "")
	for _, line := range g.lines {
		result = append(result, line, "")
	}
	result = append(result, "")
	return result
}

func (g *MarkdownGroup) ReplaceLines(lines []string) {
	g.lines = lines
	g.content.markDirty()
}
