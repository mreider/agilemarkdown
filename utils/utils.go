package utils

import (
	"strconv"
	"strings"
	"time"
	"unicode"
)

func PadIntLeft(value, width int) string {
	result := strconv.Itoa(value)
	if len(result) < width {
		result = strings.Repeat(" ", width-len(result)) + result
	}
	return result
}

func PadStringRight(value string, width int) string {
	result := value
	if len(result) < width {
		result += strings.Repeat(" ", width-len(result))
	}
	return result
}

func WeekStart(value time.Time) time.Time {
	weekday := value.Weekday()
	if weekday == 0 {
		weekday = 7
	}
	weekStart := value.Add(time.Duration(-(weekday-1)*24) * time.Hour)
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())
	return weekStart
}

func WeekDelta(baseValue, value time.Time) int {
	return int(WeekStart(value).Sub(WeekStart(baseValue)).Hours()) / 24 / 7
}

func AreEqualStrings(items1, items2 []string) bool {
	if len(items1) != len(items2) {
		return false
	}
	for i := range items1 {
		if items1[i] != items2[i] {
			return false
		}
	}
	return true
}

func WrapLinesToMarkdownCodeBlock(lines []string) []string {
	result := make([]string, 0, len(lines)+2)
	if len(lines) > 0 {
		result = append(result, "```")
		result = append(result, lines...)
		result = append(result, "```")
	}
	return result
}

func TitleFirstLetter(s string) string {
	first := true
	return strings.Map(
		func(r rune) rune {
			if first {
				first = false
				return unicode.ToTitle(r)
			}
			return r
		},
		s)
}

func ContainsStringIgnoreCase(items []string, item string) bool {
	item = strings.ToLower(item)
	for i := range items {
		if strings.ToLower(items[i]) == item {
			return true
		}
	}
	return false
}
