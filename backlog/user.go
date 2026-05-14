package backlog

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mreider/agilemarkdown/markdown"
	"github.com/mreider/agilemarkdown/utils"
)

const (
	userKeyName   = "name"
	userKeyEmails = "emails"
)

var (
	emailSeparators = regexp.MustCompile(`[,; ]+`)
)

type User struct {
	file *markdown.FrontmatterFile
}

func LoadUser(userPath string) (*User, error) {
	f, err := markdown.LoadFrontmatter(userPath)
	if err != nil {
		return nil, err
	}
	return &User{file: f}, nil
}

func NewUser(markdownData, contentPath string) (*User, error) {
	f, err := markdown.ParseFrontmatter(markdownData)
	if err != nil {
		return nil, err
	}
	f.SetPath(contentPath)
	return &User{file: f}, nil
}

func (u *User) Name() string {
	return utils.CollapseWhiteSpaces(u.file.GetString(userKeyName))
}

func (u *User) SetName(name string) {
	u.file.SetString(userKeyName, utils.CollapseWhiteSpaces(name))
}

func (u *User) Emails() []string {
	if list := u.file.GetStringSlice(userKeyEmails); len(list) > 0 {
		out := make([]string, 0, len(list))
		for _, e := range list {
			e = utils.CollapseWhiteSpaces(e)
			if e != "" {
				out = append(out, e)
			}
		}
		return out
	}
	return nil
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
	return strings.SplitN(email, "@", 2)[0]
}

func (u *User) HasEmail(email string) bool {
	if email == "" {
		return false
	}
	for _, m := range u.Emails() {
		if strings.EqualFold(m, email) {
			return true
		}
	}
	return false
}

func (u *User) HasName(name string) bool {
	return strings.EqualFold(u.Name(), utils.CollapseWhiteSpaces(name))
}

func (u *User) AddEmailIfNotExist(email string) bool {
	email = utils.CollapseWhiteSpaces(email)
	if email == "" || u.HasEmail(email) {
		return false
	}
	u.file.SetStringSlice(userKeyEmails, append(u.Emails(), email))
	return true
}

func (u *User) Save() error { return u.file.Save() }

func (u *User) Path() string { return u.file.Path() }

// UpdateItems regenerates the body of the user page with a markdown listing
// of the user's items, grouped by project and status. Frontmatter is left
// untouched.
func (u *User) UpdateItems(rootDir, tagsDir string, items []*BacklogItem, overviews map[*BacklogItem]*BacklogOverview) (string, error) {
	byOverview := make(map[*BacklogOverview]map[string][]*BacklogItem)
	for _, item := range items {
		itemStatus := strings.ToLower(item.Status())
		overview := overviews[item]
		if _, ok := byOverview[overview]; !ok {
			byOverview[overview] = make(map[string][]*BacklogItem)
		}
		byOverview[overview][itemStatus] = append(byOverview[overview][itemStatus], item)
	}

	var lines []string
	links := MakeStandardLinks(rootDir, filepath.Dir(u.file.Path()))
	lines = append(lines, utils.JoinMarkdownLinks(links...))
	lines = append(lines, "")

	for overview, byStatus := range byOverview {
		lines = append(lines, fmt.Sprintf("## %s", overview.Title()), "")
		for _, status := range AllStatuses {
			statusItems := byStatus[strings.ToLower(status.Name)]
			if len(statusItems) == 0 {
				continue
			}
			sorter := NewBacklogItemsSorter(overview)
			sorter.SortItemsByStatus(status, statusItems)
			lines = append(lines, fmt.Sprintf("### %s", status.CapitalizedName()))
			lines = append(lines, BacklogView{}.WriteMarkdownItemsWithoutAssigned(statusItems, status, filepath.Dir(u.file.Path()), tagsDir)...)
			lines = append(lines, "")
		}
	}

	u.file.SetBody(strings.Join(lines, "\n"))
	if err := u.Save(); err != nil {
		return "", err
	}
	return u.file.Path(), nil
}
