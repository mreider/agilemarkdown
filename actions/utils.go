package actions

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
)

func confirmAction(question string) bool {
	fmt.Println(question)

	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	text = strings.ToLower(strings.TrimSpace(text))
	return text == "y"
}

func existsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// processNewComments closes any unsent comments by stamping a "sent by"
// trailer line so they don't reappear. The legacy SMTP delivery path was
// removed; comments now live entirely in the markdown body and notify
// via git history / GitHub PR notifications.
func processNewComments(items []backlog.Commented, author string) error {
	for _, item := range items {
		comments := item.Comments()
		hasChanges := false
		for _, comment := range comments {
			if comment.Closed || comment.Unsent {
				continue
			}
			now := utils.GetCurrentTimestamp()
			comment.AddLine(fmt.Sprintf("sent by @%s at %s", author, now))
			hasChanges = true
		}
		if hasChanges {
			if err := item.UpdateComments(comments); err != nil {
				return err
			}
		}
	}
	return nil
}
