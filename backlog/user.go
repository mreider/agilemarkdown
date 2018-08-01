package backlog

import (
	"fmt"
	"github.com/mreider/agilemarkdown/markdown"
	"github.com/mreider/agilemarkdown/utils"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	UserEmailMetadataKey = "Email"
)

var (
	emailSeparators = regexp.MustCompile(`[,; ]+`)
)

type User struct {
	content *markdown.Content
}

func LoadUser(userPath string) (*User, error) {
	content, err := markdown.LoadMarkdown(userPath,
		[]string{UserEmailMetadataKey},
		nil, "", nil)
	if err != nil {
		return nil, err
	}
	return &User{content}, nil
}

func NewUser(markdownData, contentPath string) (*User, error) {
	content := markdown.NewMarkdown(markdownData, contentPath,
		[]string{UserEmailMetadataKey},
		nil, "", nil)
	return &User{content}, nil
}

func (u *User) Name() string {
	return utils.CollapseWhiteSpaces(u.content.Title())
}

func (u *User) SetName(name string) {
	u.content.SetTitle(utils.CollapseWhiteSpaces(name))
}

func (u *User) Emails() []string {
	emailStr := u.content.MetadataValue(UserEmailMetadataKey)
	parts := utils.SplitByRegexp(emailStr, emailSeparators)
	emails := make([]string, 0, len(parts))
	for _, email := range parts {
		email = utils.CollapseWhiteSpaces(email)
		if email != "" {
			emails = append(emails, email)
		}
	}
	return emails
}

func (u *User) PrimaryEmail() string {
	emails := u.Emails()
	if len(emails) > 0 {
		return emails[0]
	}
	return ""
}

func (u *User) Nickname() string {
	email := u.PrimaryEmail()
	if email == "" {
		return strings.Replace(u.Name(), " ", ".", -1)
	}

	parts := strings.SplitN(email, "@", 2)
	return parts[0]
}

func (u *User) HasEmail(email string) bool {
	emails := u.Emails()
	for _, m := range emails {
		if m == email {
			return true
		}
	}
	return false
}

func (u *User) HasName(name string) bool {
	if strings.ToLower(u.Name()) == utils.CollapseWhiteSpaces(strings.ToLower(name)) {
		return true
	}
	return false
}

func (u *User) AddEmailIfNotExist(email string) bool {
	email = utils.CollapseWhiteSpaces(email)
	if u.HasEmail(email) {
		return false
	}
	emails := u.Emails()
	emails = append(emails, email)
	u.content.SetMetadataValue(UserEmailMetadataKey, strings.Join(emails, ", "))
	u.content.SetHeader("")
	return true
}

func (u *User) Save() error {
	return u.content.Save()
}

func (u *User) UpdateItems(rootDir, tagsDir string, items []*BacklogItem, overviews map[*BacklogItem]*BacklogOverview) (string, error) {
	itemsByProjectAndStatus := make(map[*BacklogOverview]map[string][]*BacklogItem)
	for _, item := range items {
		itemStatus := strings.ToLower(item.Status())
		overview := overviews[item]
		if _, ok := itemsByProjectAndStatus[overview]; !ok {
			itemsByProjectAndStatus[overview] = make(map[string][]*BacklogItem)
		}
		itemsByProjectAndStatus[overview][itemStatus] = append(itemsByProjectAndStatus[overview][itemStatus], item)
	}
	var lines []string
	for overview, itemsByStatus := range itemsByProjectAndStatus {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("## %s", overview.Title()))
		lines = append(lines, "")
		for _, status := range AllStatuses {
			statusItems := itemsByStatus[strings.ToLower(status.Name)]
			if len(statusItems) == 0 {
				continue
			}
			sorter := NewBacklogItemsSorter(overview)
			sorter.SortItemsByStatus(status, statusItems)
			lines = append(lines, fmt.Sprintf("### %s", status.CapitalizedName()))
			itemsLines := BacklogView{}.WriteMarkdownItemsWithoutAssigned(statusItems, status, filepath.Dir(u.content.ContentPath()), tagsDir)
			lines = append(lines, itemsLines...)
			lines = append(lines, "")
		}
	}

	links := MakeStandardLinks(rootDir, filepath.Dir(u.content.ContentPath()))
	u.content.SetLinks(utils.JoinMarkdownLinks(links...))
	u.SetFreeText(lines)
	err := u.Save()
	return u.content.ContentPath(), err
}

func (u *User) SetFreeText(freeText []string) {
	u.content.SetFreeText(freeText)
}

func (u *User) Path() string {
	return u.content.ContentPath()
}
