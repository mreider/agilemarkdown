package tests

import (
	"github.com/mreider/agilemarkdown/utils"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestPadIntLeft(t *testing.T) {
	assert.Equal(t, "12", utils.PadIntLeft(12, 0))
	assert.Equal(t, "12", utils.PadIntLeft(12, 1))
	assert.Equal(t, "12", utils.PadIntLeft(12, 2))
	assert.Equal(t, " 12", utils.PadIntLeft(12, 3))
	assert.Equal(t, "  12", utils.PadIntLeft(12, 4))
}

func TestPadStringRight(t *testing.T) {
	assert.Equal(t, "12", utils.PadStringRight("12", 0))
	assert.Equal(t, "12", utils.PadStringRight("12", 1))
	assert.Equal(t, "12", utils.PadStringRight("12", 2))
	assert.Equal(t, "12 ", utils.PadStringRight("12", 3))
	assert.Equal(t, "12  ", utils.PadStringRight("12", 4))
}

func TestPadStringLeft(t *testing.T) {
	assert.Equal(t, "12", utils.PadStringLeft("12", 0))
	assert.Equal(t, "12", utils.PadStringLeft("12", 1))
	assert.Equal(t, "12", utils.PadStringLeft("12", 2))
	assert.Equal(t, " 12", utils.PadStringLeft("12", 3))
	assert.Equal(t, "  12", utils.PadStringLeft("12", 4))
}

func TestWeekStart(t *testing.T) {
	assert.Equal(t, createDate(2018, 4, 23), utils.WeekStart(createDate(2018, 4, 23)))
	assert.Equal(t, createDate(2018, 4, 23), utils.WeekStart(createDate(2018, 4, 25)))
	assert.Equal(t, createDate(2018, 4, 23), utils.WeekStart(createDate(2018, 4, 29)))
	assert.Equal(t, createDate(2018, 4, 16), utils.WeekStart(createDate(2018, 4, 22)))
	assert.Equal(t, createDate(2018, 4, 30), utils.WeekStart(createDate(2018, 4, 30)))
	assert.Equal(t, createDate(2018, 4, 30), utils.WeekStart(createDate(2018, 5, 6)))
}

func TestWeekEnd(t *testing.T) {
	assert.Equal(t, createDate(2018, 7, 8), utils.WeekEnd(createDate(2018, 7, 5)))
	assert.Equal(t, createDate(2018, 7, 8), utils.WeekEnd(createDate(2018, 7, 8)))
	assert.Equal(t, createDate(2018, 7, 8), utils.WeekEnd(createDate(2018, 7, 2)))
	assert.Equal(t, createDate(2018, 7, 15), utils.WeekEnd(createDate(2018, 7, 9)))
	assert.Equal(t, createDate(2018, 7, 1), utils.WeekEnd(createDate(2018, 7, 1)))
}

func TestWeekDelta(t *testing.T) {
	baseValue := createDate(2018, 4, 25)
	assert.Equal(t, 0, utils.WeekDelta(baseValue, baseValue))
	assert.Equal(t, 0, utils.WeekDelta(baseValue, createDate(2018, 4, 23)))
	assert.Equal(t, 0, utils.WeekDelta(baseValue, createDate(2018, 4, 29)))
	assert.Equal(t, -1, utils.WeekDelta(baseValue, createDate(2018, 4, 22)))
	assert.Equal(t, -2, utils.WeekDelta(baseValue, createDate(2018, 4, 11)))
	assert.Equal(t, 1, utils.WeekDelta(baseValue, createDate(2018, 4, 30)))
	assert.Equal(t, 1, utils.WeekDelta(baseValue, createDate(2018, 5, 6)))
	assert.Equal(t, 2, utils.WeekDelta(baseValue, createDate(2018, 5, 7)))
}

func TestTitleFirstLetter(t *testing.T) {
	assert.Equal(t, "A task", utils.TitleFirstLetter("a task"))
	assert.Equal(t, "An apple", utils.TitleFirstLetter("An apple"))
	assert.Equal(t, "2a task", utils.TitleFirstLetter("2a task"))
	assert.Equal(t, "", utils.TitleFirstLetter(""))
	assert.Equal(t, "X", utils.TitleFirstLetter("x"))
}

func TestGetValidFileName(t *testing.T) {
	assert.Equal(t, "apple", utils.GetValidFileName("apple"))
	assert.Equal(t, "apple-red", utils.GetValidFileName("apple/red"))
	assert.Equal(t, "apple-red-green", utils.GetValidFileName("apple:red:green"))
}

func TestCollapseWhiteSpaces(t *testing.T) {
	assert.Equal(t, "", utils.CollapseWhiteSpaces(""))
	assert.Equal(t, "", utils.CollapseWhiteSpaces(" "))
	assert.Equal(t, "", utils.CollapseWhiteSpaces(" \n"))
	assert.Equal(t, "it is a test", utils.CollapseWhiteSpaces(" it \n  is a    \t  test  "))
}

func createDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.Local)
}
