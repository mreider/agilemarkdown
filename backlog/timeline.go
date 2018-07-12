package backlog

import (
	"encoding/json"
	"fmt"
	"github.com/gerald1248/timeline"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type TimelineGenerator struct {
	rootDir string
}

type timelineItem struct {
	item      *BacklogItem
	startDate time.Time
	endDate   time.Time
}

func NewTimelineGenerator(rootDir string) *TimelineGenerator {
	return &TimelineGenerator{rootDir: rootDir}
}

func (tg *TimelineGenerator) Execute() error {
	return tg.ExecuteForTag("")
}

func (tg *TimelineGenerator) ExecuteForTag(tag string) error {
	_, itemsTags, _, _, err := ItemsAndIdeasTags(tg.rootDir)
	if err != nil {
		return err
	}
	for itemsTag, items := range itemsTags {
		if tag != "" && itemsTag != tag {
			continue
		}

		timelineItems := make([]*timelineItem, 0, len(items))
		for _, item := range items {
			startDate, endDate := item.Timeline(tag)
			if !startDate.IsZero() && !endDate.IsZero() {
				timelineItems = append(timelineItems, &timelineItem{item: item, startDate: startDate, endDate: endDate})
			}
		}

		tg.sortTimelineItems(timelineItems)

		timelineDirectory := filepath.Join(tg.rootDir, TimelineDirectoryName)
		os.MkdirAll(timelineDirectory, 0777)
		pngPath := filepath.Join(timelineDirectory, fmt.Sprintf("%s.png", tag))

		if len(timelineItems) == 0 {
			os.Remove(pngPath)
			return nil
		}

		tasks := make([]*timeline.Task, 0, len(timelineItems))
		for _, item := range timelineItems {
			tasks = append(tasks, &timeline.Task{Label: item.item.Title(), Start: item.startDate.Format("2006-01-02"), End: item.endDate.AddDate(0, 0, 1).Format("2006-01-02")})
		}
		data, _ := json.Marshal(timeline.Data{
			Tasks:      tasks,
			MySettings: &timeline.Settings{},
			MyTheme: &timeline.Theme{
				ColorScheme:      "gradient",
				BorderColor1:     "#aaffaa",
				FillColor1:       "#bbffbb",
				BorderColor2:     "#ccffcc",
				FillColor2:       "#ddffdd",
				FrameBorderColor: "#ffffff",
				FrameFillColor:   "#aaaaaa",
				StripeColorDark:  "#dddddd",
				StripeColorLight: "#eeeeee",
				GridColor:        "#888888",
			},
		})
		result := timeline.ProcessBytes(data)
		result.Context.SavePNG(pngPath)
	}
	return nil
}

func (tg *TimelineGenerator) sortTimelineItems(items []*timelineItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].startDate != items[j].startDate {
			return items[i].startDate.Before(items[j].startDate)
		}

		if items[i].endDate != items[j].endDate {
			return items[i].endDate.Before(items[j].endDate)
		}

		if items[i].item.Modified() != items[j].item.Modified() {
			return items[i].item.Modified().Before(items[j].item.Modified())
		}
		return items[i].item.Name() < items[j].item.Name()
	})
}
