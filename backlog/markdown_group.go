package backlog

import (
	"github.com/mreider/agilemarkdown/utils"
)

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
	if g.lines[index] == line {
		return
	}

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

func (g *MarkdownGroup) RawLines() []string {
	return g.lines
}

func (g *MarkdownGroup) ReplaceLines(lines []string) {
	if utils.AreEqualStrings(g.lines, lines) {
		return
	}

	g.lines = lines
	g.content.markDirty()
}
