package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type SyncTimelineStep struct {
	root *backlog.BacklogsStructure
}

func NewSyncTimelineStep(root *backlog.BacklogsStructure) *SyncTimelineStep {
	return &SyncTimelineStep{root: root}
}

func (s *SyncTimelineStep) Execute() error {
	allTags, itemsTags, _, _, err := backlog.ItemsAndIdeasTags(s.root)
	if err != nil {
		return err
	}

	timelineGenerator := backlog.NewTimelineGenerator(s.root)
	for tag, tagItems := range itemsTags {
		hasTimeline := false
		for _, item := range tagItems {
			startDate, endDate := item.Timeline()
			if !startDate.IsZero() && !endDate.IsZero() {
				hasTimeline = true
				break
			}
		}
		fmt.Println(tag, hasTimeline)
		if hasTimeline {
			timelineGenerator.ExecuteForTag(tag)
		} else {
			timelineGenerator.RemoveTimeline(tag)
		}
	}

	timelineDir := s.root.TimelineDirectory()
	items, err := ioutil.ReadDir(timelineDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	lines := []string{"# Timelines", ""}
	lines = append(lines,
		fmt.Sprintf(utils.JoinMarkdownLinks(backlog.MakeStandardLinks(s.root.Root(), s.root.Root())...)))
	lines = append(lines, "")

	for _, item := range items {
		if strings.HasSuffix(item.Name(), ".png") {
			timelineImagePath := filepath.Join(timelineDir, item.Name())
			timelineTag := strings.TrimSuffix(item.Name(), ".png")
			if _, ok := allTags[timelineTag]; ok {
				lines = append(lines, fmt.Sprintf("## Tag: %s", utils.MakeMarkdownLink(timelineTag, filepath.Join(s.root.TagsDirectory(), timelineTag), s.root.Root())))
				lines = append(lines, "")
				lines = append(lines, fmt.Sprintf("%s", utils.MakeMarkdownImageLink(timelineTag, timelineImagePath, s.root.Root())))
				lines = append(lines, "")
			} else {
				os.Remove(timelineImagePath)
			}
		}
	}

	return ioutil.WriteFile(s.root.TimelineFile(), []byte(strings.Join(lines, "\n")), 0644)
}
