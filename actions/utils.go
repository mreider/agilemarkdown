package actions

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"
	"github.com/mreider/agilemarkdown/git"
	"github.com/mreider/agilemarkdown/utils"
	"os"
	"strings"
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

func sendNewComments(items []backlog.Commented, onSend func(item backlog.Commented, to []string, comment []string) (me string, err error)) {
	for _, item := range items {
		comments := item.Comments()
		hasChanges := false
		for _, comment := range comments {
			if comment.Closed || comment.Unsent {
				continue
			}
			me, err := onSend(item, comment.Users, comment.Text)
			now := utils.GetCurrentTimestamp()
			hasChanges = true
			if err != nil {
				comment.AddLine(fmt.Sprintf("can't send by @%s at %s: %v", me, now, err))
			} else {
				comment.AddLine(fmt.Sprintf("sent by @%s at %s", me, now))
			}
		}
		if hasChanges {
			item.UpdateComments(comments)
		}
	}
}

func sendComment(userList *backlog.UserList, comment []string, title, from string, to []string, mailSender *utils.MailSender, cfg *config.Config, rootDir, contentPath string) (me string, err error) {
	sepIndex := strings.LastIndexByte(from, ' ')
	if sepIndex >= 0 {
		from = from[sepIndex+1:]
		from = strings.Trim(from, "<>")
	}
	if from == "" {
		from, _, _ = git.CurrentUser()
	}

	meUser := userList.User(from)
	if meUser == nil {
		return "", fmt.Errorf("unknown user %s", from)
	}
	toUsers := make([]*backlog.User, 0, len(to))
	for _, user := range to {
		toUser := userList.User(user)
		if toUser == nil {
			return meUser.Nickname(), fmt.Errorf("unknown user %s", to)
		}
		toUsers = append(toUsers, toUser)
	}
	if mailSender == nil {
		return meUser.Nickname(), errors.New("SMTP server isn't configured")
	}

	msgText := strings.Join(comment, "\n")
	remoteOriginUrl, _ := git.RemoteOriginUrl()
	remoteOriginUrl = strings.TrimSuffix(remoteOriginUrl, ".git")
	if remoteOriginUrl != "" {
		var itemGitUrl string
		itemPath := strings.TrimPrefix(contentPath, rootDir)
		itemPath = strings.TrimPrefix(itemPath, string(os.PathSeparator))
		itemPath = strings.Replace(itemPath, string(os.PathSeparator), "/", -1)
		if cfg.RemoteGitUrlFormat != "" {
			parts := strings.Split(cfg.RemoteGitUrlFormat, "%s")
			if len(parts) >= 3 {
				itemGitUrl = fmt.Sprintf(cfg.RemoteGitUrlFormat, remoteOriginUrl, itemPath)
			} else if len(parts) == 2 {
				itemGitUrl = fmt.Sprintf(cfg.RemoteGitUrlFormat, itemPath)
			} else {
				itemGitUrl = cfg.RemoteGitUrlFormat
			}
		} else {
			itemGitUrl = fmt.Sprintf("%s/%s", remoteOriginUrl, itemPath)
		}
		msgText += fmt.Sprintf("\n\nView on Git: %s\n", itemGitUrl)
		if cfg.RemoteWebUrlFormat != "" {
			itemWebUrl := fmt.Sprintf(cfg.RemoteWebUrlFormat, itemPath)
			msgText += fmt.Sprintf("View on the web: %s\n", itemWebUrl)
		}
	}

	fromSubject := meUser.Nickname()
	if meUser.Name() != meUser.Nickname() {
		fromSubject += fmt.Sprintf(" (%s)", meUser.Name())
	}

	toEmails := make([]string, 0, len(toUsers))
	for _, user := range toUsers {
		toEmails = append(toEmails, user.PrimaryEmail())
	}

	err = mailSender.SendEmail(
		toEmails,
		fmt.Sprintf("%s. New comment from %s", title, fromSubject),
		msgText)
	return meUser.Nickname(), err
}
