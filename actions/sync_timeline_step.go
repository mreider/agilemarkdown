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
	rootDir string
}

func NewSyncTimelineStep(rootDir string) *SyncTimelineStep {
	return &SyncTimelineStep{rootDir: rootDir}
}

func (s *SyncTimelineStep) Execute() error {
	allTags, itemsTags, _, _, err := backlog.ItemsAndIdeasTags(s.rootDir)
	if err != nil {
		return err
	}

	timelineGenerator := backlog.NewTimelineGenerator(s.rootDir)
	for tag, tagItems := range itemsTags {
		hasTimeline := false
		for _, item := range tagItems {
			startDate, endDate := item.Timeline(tag)
			if !startDate.IsZero() && !endDate.IsZero() {
				hasTimeline = true
				break
			}
		}
		if hasTimeline {
			timelineGenerator.ExecuteForTag(tag)
		} else {
			timelineGenerator.RemoveTimeline(tag)
		}
	}

	timelineDir := filepath.Join(s.rootDir, backlog.TimelineDirectoryName)
	items, err := ioutil.ReadDir(timelineDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	lines := []string{"# Timelines", ""}
	lines = append(lines,
		fmt.Sprintf(utils.JoinMarkdownLinks(backlog.MakeStandardLinks(s.rootDir, s.rootDir)...)))
	lines = append(lines, "")

	for _, item := range items {
		if strings.HasSuffix(item.Name(), ".png") {
			timelineImagePath := filepath.Join(timelineDir, item.Name())
			timelineTag := strings.TrimSuffix(item.Name(), ".png")
			if _, ok := allTags[timelineTag]; ok {
				lines = append(lines, fmt.Sprintf("## Tag: %s", utils.MakeMarkdownLink(timelineTag, filepath.Join(s.rootDir, backlog.TagsDirectoryName, timelineTag), s.rootDir)))
				lines = append(lines, "")
				lines = append(lines, fmt.Sprintf("%s", utils.MakeMarkdownImageLink(timelineTag, timelineImagePath, s.rootDir)))
				lines = append(lines, "")
			} else {
				os.Remove(timelineImagePath)
			}
		}
	}

	return ioutil.WriteFile(filepath.Join(s.rootDir, backlog.TimelineFileName), []byte(strings.Join(lines, "\n")), 0644)
}
