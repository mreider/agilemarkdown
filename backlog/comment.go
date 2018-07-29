package backlog

import (
	"github.com/mreider/agilemarkdown/markdown"
	"strings"
	"unicode"
)

type Commented interface {
	Comments() []*Comment
	UpdateComments(comments []*Comment)
	Path() string
	Title() string
}

type Comment struct {
	Users   []string
	Text    []string
	rawText []string
	Closed  bool
	Unsent  bool
}

func (c *Comment) AddLine(line string) {
	c.Text = append(c.Text, line)

	c.rawText = append(c.rawText, line)
	i := len(c.rawText) - 1
	for i > 0 && strings.TrimSpace(c.rawText[i-1]) == "" {
		c.rawText[i-1], c.rawText[i] = c.rawText[i], c.rawText[i-1]
		i--
	}
}

type MarkdownComments struct {
	markdown *markdown.Content
}

func NewMarkdownComments(content *markdown.Content) *MarkdownComments {
	return &MarkdownComments{markdown: content}
}

func (c *MarkdownComments) Comments() []*Comment {
	commentsStartIndex := -1
	for i := len(c.markdown.FreeText()) - 1; i >= 0; i-- {
		if commentsTitleRe.MatchString(c.markdown.FreeText()[i]) {
			commentsStartIndex = i + 1
			break
		}
	}
	if commentsStartIndex == -1 {
		return nil
	}

	comments := make([]*Comment, 0)
	var comment *Comment
	for i := commentsStartIndex; i < len(c.markdown.FreeText()); i++ {
		line := strings.TrimRightFunc(c.markdown.FreeText()[i], unicode.IsSpace)
		if line == "" {
			if comment != nil {
				comment.rawText = append(comment.rawText, c.markdown.FreeText()[i])
			}
			continue
		}
		if strings.HasPrefix(line, "#") {
			break
		}
		matches := commentRe.FindStringSubmatch(line)
		if len(matches) > 0 {
			if comment != nil {
				comments = append(comments, comment)
			}
			rawUsers := commentUserSeparatorRe.Split(matches[2], -1)
			allUsers := make(map[string]bool)
			users := make([]string, 0, len(rawUsers))
			for _, user := range rawUsers {
				user = strings.TrimPrefix(user, "@")
				user = strings.TrimSuffix(user, ".")
				if user == "" {
					continue
				}
				if !allUsers[user] {
					users = append(users, user)
					allUsers[user] = true
				}
			}
			comment = &Comment{Users: users}
			if len(matches[1]) > 0 {
				comment.Closed = true
			}
			text := strings.TrimSpace(matches[4])
			if text != "" {
				comment.Text = append(comment.Text, text)
			}
			comment.rawText = append(comment.rawText, c.markdown.FreeText()[i])
		} else {
			if comment != nil {
				line := strings.TrimSpace(line)
				if strings.HasPrefix(strings.ToLower(line), "sent by ") {
					comment.Closed = true
				} else if strings.HasPrefix(strings.ToLower(line), "can't send by ") {
					comment.Unsent = true
				}
				comment.Text = append(comment.Text, line)
				comment.rawText = append(comment.rawText, c.markdown.FreeText()[i])
			}
		}
	}
	if comment != nil {
		comments = append(comments, comment)
	}
	return comments
}

func (c *MarkdownComments) UpdateComments(comments []*Comment) {
	commentsStartIndex := -1
	for i := len(c.markdown.FreeText()) - 1; i >= 0; i-- {
		if commentsTitleRe.MatchString(c.markdown.FreeText()[i]) {
			commentsStartIndex = i + 1
			break
		}
	}
	if commentsStartIndex == -1 {
		return
	}

	for commentsStartIndex < len(c.markdown.FreeText()) && strings.TrimSpace(c.markdown.FreeText()[commentsStartIndex]) == "" {
		commentsStartIndex++
	}

	commentsFinishIndex := commentsStartIndex
	for i := commentsStartIndex; i < len(c.markdown.FreeText()); i++ {
		line := strings.TrimRightFunc(c.markdown.FreeText()[i], unicode.IsSpace)
		commentsFinishIndex = i
		if strings.HasPrefix(line, "#") {
			break
		}
	}

	newFreeText := make([]string, 0, len(c.markdown.FreeText()))
	newFreeText = append(newFreeText, c.markdown.FreeText()[:commentsStartIndex]...)
	for _, comment := range comments {
		newFreeText = append(newFreeText, comment.rawText...)
	}
	newFreeText = append(newFreeText, c.markdown.FreeText()[commentsFinishIndex:]...)
	c.markdown.SetFreeText(newFreeText)
}
