package markdown

import (
	"github.com/mreider/agilemarkdown/utils"
)

type Group struct {
	content *Content
	title   string
	lines   []string
}

func NewGroup(content *Content, title string, lines []string) *Group {
	return &Group{
		content: content,
		title:   title,
		lines:   lines,
	}
}

func (g *Group) Title() string {
	return g.title
}

func (g *Group) Count() int {
	return len(g.lines)
}

func (g *Group) Line(index int) string {
	return g.lines[index]
}

func (g *Group) SetLine(index int, line string) {
	if g.lines[index] == line {
		return
	}

	g.lines[index] = line
	g.content.markDirty()
}

func (g *Group) AddLine(line string) {
	g.lines = append(g.lines, line)
	g.content.markDirty()
}

func (g *Group) DeleteLine(index int) {
	copy(g.lines[index:], g.lines[index+1:])
	g.lines = g.lines[:len(g.lines)-1]
	g.content.markDirty()
}

func (g *Group) RawLines() []string {
	return g.lines
}

func (g *Group) ReplaceLines(lines []string) {
	if utils.AreEqualStrings(g.lines, lines) {
		return
	}

	g.lines = lines
	g.content.markDirty()
}

func (g *Group) Lines() []string {
	return g.lines
}
