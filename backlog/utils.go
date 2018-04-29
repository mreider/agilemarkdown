package backlog

import "time"

const (
	timestampLayout = "2006-01-02 03:04 PM"
)

func getCurrentTimestamp() string {
	return time.Now().Format(timestampLayout)
}

func parseTimestamp(timestamp string) (time.Time, error) {
	return time.Parse(timestampLayout, timestamp)
}

func areEqual(items1, items2 []string) bool {
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
