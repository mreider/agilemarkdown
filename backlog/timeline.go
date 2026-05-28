package backlog

import (
	"os"
	"path/filepath"
	"sort"
	"time"
)

// TimelineGenerator builds per-tag timelines as ASCII text saved to
// timeline/<tag>.txt files. The wider timeline.md page links to each.
type TimelineGenerator struct {
	root *BacklogsStructure
}

type timelineItem struct {
	item      *BacklogItem
	startDate time.Time
	endDate   time.Time
}

func NewTimelineGenerator(root *BacklogsStructure) *TimelineGenerator {
	return &TimelineGenerator{root: root}
}

func (tg *TimelineGenerator) Execute() error {
	return tg.ExecuteForTag("")
}

func (tg *TimelineGenerator) ExecuteForTag(tag string) error {
	_, itemsTags, _, err := ItemsTags(tg.root)
	if err != nil {
		return err
	}
	for itemsTag, items := range itemsTags {
		if tag != "" && itemsTag != tag {
			continue
		}

		tlItems := make([]*timelineItem, 0, len(items))
		for _, item := range items {
			start, end := item.Timeline()
			if !start.IsZero() && !end.IsZero() {
				tlItems = append(tlItems, &timelineItem{item: item, startDate: start, endDate: end})
			}
		}
		sort.Slice(tlItems, func(i, j int) bool {
			a, b := tlItems[i], tlItems[j]
			if !a.startDate.Equal(b.startDate) {
				return a.startDate.Before(b.startDate)
			}
			return a.item.Name() < b.item.Name()
		})

		dir := tg.root.TimelineDirectory()
		if err := os.MkdirAll(dir, 0777); err != nil {
			return err
		}
		path := filepath.Join(dir, itemsTag+".txt")

		if len(tlItems) == 0 {
			_ = os.Remove(path)
			continue
		}

		// Reuse the ASCII renderer that the CLI / MCP also serves.
		items := make([]*BacklogItem, 0, len(tlItems))
		for _, t := range tlItems {
			items = append(items, t.item)
		}
		text := TimelineASCII(items, itemsTag)
		if err := os.WriteFile(path, []byte(text), 0644); err != nil {
			return err
		}
	}
	return nil
}

func (tg *TimelineGenerator) RenameTimeline(oldTag, newTag string) error {
	dir := tg.root.TimelineDirectory()
	oldPath := filepath.Join(dir, oldTag+".txt")
	newPath := filepath.Join(dir, newTag+".txt")
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return nil
	}
	return os.Rename(oldPath, newPath)
}

func (tg *TimelineGenerator) RemoveTimeline(tag string) {
	dir := tg.root.TimelineDirectory()
	_ = os.Remove(filepath.Join(dir, tag+".txt"))
}
