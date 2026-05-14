package utils

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	timestampLayout = "2006-01-02T15:04:05Z07:00"
)

var (
	separators  = regexp.MustCompile(`[ \\/&_=+:]`)
	dashes      = regexp.MustCompile(`[\-]+`)
	illegalName = regexp.MustCompile(`[^[:alnum:]-]`)
	spacesRe    = regexp.MustCompile(`\s+`)
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

func PadStringLeft(value string, width int) string {
	result := value
	if len(result) < width {
		result = strings.Repeat(" ", width-len(result)) + result
	}
	return result
}

// WeekStart returns the Monday 00:00 of the ISO-week containing value.
// Day arithmetic uses AddDate so DST transitions don't shift the result.
func WeekStart(value time.Time) time.Time {
	weekday := int(value.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	d := time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
	return d.AddDate(0, 0, -(weekday - 1))
}

// WeekEnd returns the Sunday 00:00 closing the ISO-week of value.
func WeekEnd(value time.Time) time.Time {
	return WeekStart(value).AddDate(0, 0, 6)
}

// WeekDelta returns the number of whole calendar weeks separating the
// week of `value` from the week of `baseValue`. Negative when value is
// earlier than base. Rounds to the nearest week so DST drift doesn't
// produce off-by-one errors.
func WeekDelta(baseValue, value time.Time) int {
	a := WeekStart(baseValue)
	b := WeekStart(value)
	weeks := b.Sub(a).Hours() / (24 * 7)
	return int(math.Round(weeks))
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

func GetCurrentTimestamp() string {
	return GetTimestamp(time.Now().UTC())
}

func GetTimestamp(moment time.Time) string {
	return moment.UTC().Format(timestampLayout)
}

func ParseTimestamp(timestamp string) (time.Time, error) {
	timestamp = strings.TrimSpace(timestamp)
	if timestamp == "" {
		return time.Time{}, nil
	}
	for _, layout := range []string{
		timestampLayout,
		"2006-01-02 03:04 PM",
		"2006-01-02 15:04",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, timestamp); err == nil {
			return t, nil
		}
	}
	return time.Time{}, &time.ParseError{Layout: timestampLayout, Value: timestamp}
}

func MakeMarkdownLink(linkTitle, linkPath, baseDir string) string {
	return fmt.Sprintf("[%s](%s)", linkTitle, strings.Replace(GetMarkdownLinkPath(linkPath, baseDir), " ", "%20", -1))
}

func GetMarkdownLinkPath(linkPath, baseDir string) string {
	linkPath, _ = filepath.Abs(linkPath)
	baseDir, _ = filepath.Abs(baseDir)

	upCount := 0
	for !strings.HasPrefix(linkPath, baseDir+string(os.PathSeparator)) {
		upCount++
		baseDir = filepath.Dir(baseDir)
	}

	linkPath = strings.TrimPrefix(linkPath, baseDir)
	linkPath = strings.TrimPrefix(linkPath, string(os.PathSeparator))

	if upCount > 0 {
		linkPath = strings.Repeat(fmt.Sprintf("..%c", os.PathSeparator), upCount) + linkPath
	}

	return linkPath
}

func MakeMarkdownImageLink(linkTitle, imagePath, baseDir string) string {
	return fmt.Sprintf("!%s", MakeMarkdownLink(linkTitle, imagePath, baseDir))
}

func JoinMarkdownLinks(links ...string) string {
	return strings.Join(links, " • ")
}

func GetValidFileName(name string) string {
	fileName := strings.TrimSpace(name)
	fileName = separators.ReplaceAllString(fileName, "-")
	fileName = illegalName.ReplaceAllString(fileName, "")
	fileName = dashes.ReplaceAllString(fileName, "-")
	return fileName
}

func CollapseWhiteSpaces(value string) string {
	return strings.TrimSpace(spacesRe.ReplaceAllString(value, " "))
}

func RemoveItemIgnoreCase(items []string, item string) []string {
	if len(items) == 0 {
		return items
	}

	item = strings.ToLower(item)
	result := make([]string, 0, len(items))
	for _, it := range items {
		if strings.ToLower(it) != item {
			result = append(result, it)
		}
	}
	return result
}

func RenameItemIgnoreCase(items []string, oldItem, newItem string) []string {
	if len(items) == 0 {
		return items
	}

	if strings.ToLower(oldItem) == strings.ToLower(newItem) {
		return items
	}

	newItemExists := false
	for _, it := range items {
		if strings.ToLower(it) == strings.ToLower(newItem) {
			newItemExists = true
			break
		}
	}

	oldItem = strings.ToLower(oldItem)
	result := make([]string, 0, len(items))
	for _, it := range items {
		if strings.ToLower(it) != oldItem {
			result = append(result, it)
		} else if !newItemExists {
			result = append(result, newItem)
			newItemExists = true
		}
	}
	return result
}

func SplitByRegexp(value string, re *regexp.Regexp) []string {
	parts := re.Split(value, -1)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
