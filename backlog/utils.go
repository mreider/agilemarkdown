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
