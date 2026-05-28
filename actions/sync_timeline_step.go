package actions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
)

type SyncTimelineStep struct {
	root *backlog.BacklogsStructure
}

func NewSyncTimelineStep(root *backlog.BacklogsStructure) *SyncTimelineStep {
	return &SyncTimelineStep{root: root}
}

func (s *SyncTimelineStep) Execute() error {
	fmt.Println("Generating timeline page")

	allTags, itemsTags, _, err := backlog.ItemsTags(s.root)
	if err != nil {
		return err
	}

	gen := backlog.NewTimelineGenerator(s.root)
	for tag, tagItems := range itemsTags {
		hasTimeline := false
		for _, item := range tagItems {
			start, end := item.Timeline()
			if !start.IsZero() && !end.IsZero() {
				hasTimeline = true
				break
			}
		}
		if hasTimeline {
			if err := gen.ExecuteForTag(tag); err != nil {
				return err
			}
		} else {
			gen.RemoveTimeline(tag)
		}
	}

	timelineDir := s.root.TimelineDirectory()
	items, err := os.ReadDir(timelineDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	lines := []string{"# Timelines", ""}
	lines = append(lines, utils.JoinMarkdownLinks(backlog.MakeStandardLinks(s.root.Root(), s.root.Root())...))
	lines = append(lines, "")

	for _, item := range items {
		if !strings.HasSuffix(item.Name(), ".txt") {
			continue
		}
		timelineTag := strings.TrimSuffix(item.Name(), ".txt")
		if _, ok := allTags[timelineTag]; !ok {
			_ = os.Remove(filepath.Join(timelineDir, item.Name()))
			continue
		}
		lines = append(lines,
			fmt.Sprintf("## Tag: %s",
				utils.MakeMarkdownLink(timelineTag, filepath.Join(s.root.TagsDirectory(), timelineTag), s.root.Root())))
		lines = append(lines, "")
		body, err := os.ReadFile(filepath.Join(timelineDir, item.Name()))
		if err != nil {
			return err
		}
		lines = append(lines, "```")
		lines = append(lines, strings.TrimRight(string(body), "\n"))
		lines = append(lines, "```", "")
	}

	return os.WriteFile(s.root.TimelineFile(), []byte(strings.Join(lines, "\n")), 0644)
}
