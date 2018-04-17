package backlog

import "time"

func getCurrentTimestamp() string {
	return time.Now().Format("2006-01-02 03:04 PM")
}
