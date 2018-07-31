package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"
	"github.com/mreider/agilemarkdown/git"
	"github.com/mreider/agilemarkdown/utils"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type SyncIdeasStep struct {
	root     *backlog.BacklogsStructure
	cfg      *config.Config
	userList *backlog.UserList
	author   string
}

func NewSyncIdeasStep(root *backlog.BacklogsStructure, cfg *config.Config, userList *backlog.UserList, author string) *SyncIdeasStep {
	return &SyncIdeasStep{root: root, cfg: cfg, userList: userList, author: author}
}

func (s *SyncIdeasStep) Execute() error {
	_, itemsTags, ideasTags, overviews, err := backlog.ItemsAndIdeasTags(s.root)
	if err != nil {
		return err
	}

	ideasDir := s.root.IdeasDirectory()
	ideas, err := backlog.LoadIdeas(ideasDir)
	if err != nil {
		return err
	}

	s.sendNewCommentsForIdeas(ideas)

	ideasByRank := make(map[string][]*backlog.BacklogIdea)
	var ranks []string
	tagsDir := s.root.TagsDirectory()
	for _, idea := range ideas {
		err := s.updateIdea(tagsDir, idea, ideasTags, itemsTags, overviews)
		if err != nil {
			fmt.Printf("can't update idea '%s'\n", err)
		}
		rank := idea.Rank()
		if _, ok := ideasByRank[strings.TrimSpace(rank)]; !ok {
			ranks = append(ranks, rank)
		}
		ideasByRank[strings.TrimSpace(rank)] = append(ideasByRank[strings.TrimSpace(rank)], idea)
	}

	sort.Strings(ranks)
	if len(ranks) > 0 && strings.TrimSpace(ranks[0]) == "" {
		ranks = append(ranks, "")
		ranks = ranks[1:]
	}

	lines := []string{"# Ideas", ""}
	lines = append(lines,
		fmt.Sprintf(utils.JoinMarkdownLinks(backlog.MakeStandardLinks(s.root.Root(), s.root.Root())...)))
	lines = append(lines, "")
	for _, rank := range ranks {
		if rank != "" {
			lines = append(lines, fmt.Sprintf("## Rank: %s", strings.TrimSpace(rank)))
		} else {
			lines = append(lines, "## Rank: unassigned")
		}
		lines = append(lines, "")
		lines = append(lines, backlog.BacklogView{}.WriteMarkdownIdeas(ideasByRank[strings.TrimSpace(rank)], s.root.Root(), s.root.TagsDirectory())...)
		lines = append(lines, "")
	}
	return ioutil.WriteFile(s.root.IdeasFile(), []byte(strings.Join(lines, "\n")), 0644)
}

func (s *SyncIdeasStep) updateIdea(tagsDir string, idea *backlog.BacklogIdea, ideasTags map[string][]*backlog.BacklogIdea, itemsTags map[string][]*backlog.BacklogItem, overviews map[*backlog.BacklogItem]*backlog.BacklogOverview) error {
	if !idea.HasMetadata() {
		author, created, err := git.InitCommitInfo(idea.Path())
		if err != nil {
			return err
		}
		if author == "" {
			author, _, _ = git.CurrentUser()
			created = time.Now()
		}

		ideaName := filepath.Base(idea.Path())
		ideaName = strings.TrimSuffix(ideaName, filepath.Ext(ideaName))
		ideaTitle := strings.Replace(ideaName, "-", " ", -1)
		ideaTitle = strings.Replace(ideaTitle, "_", " ", -1)
		ideaTitle = utils.TitleFirstLetter(ideaTitle)
		idea.SetTitle(ideaTitle)
		idea.SetCreated(utils.GetTimestamp(created))
		idea.SetModified(utils.GetTimestamp(created))
		idea.SetAuthor(author)
		idea.SetTags(nil)
		idea.SetText(idea.Text())
		idea.Save()
	}
	idea.UpdateLinks(s.root.Root())

	ideaTags := make(map[string]struct{})
NextTag:
	for tag, tagIdeas := range ideasTags {
		for _, tagIdea := range tagIdeas {
			if tagIdea.Path() == idea.Path() {
				ideaTags[tag] = struct{}{}
				continue NextTag
			}
		}
	}

	itemsByStatus := make(map[string][]*backlog.BacklogItem)
	for tag := range ideaTags {
		for _, item := range itemsTags[tag] {
			itemStatus := strings.ToLower(item.Status())
			itemsByStatus[itemStatus] = append(itemsByStatus[itemStatus], item)
		}
	}
	for _, statusItems := range itemsByStatus {
		sorter := backlog.NewBacklogItemsSorter()
		sorter.SortItemsByModifiedDesc(statusItems)
	}

	var items []*backlog.BacklogItem
	for _, status := range backlog.AllStatuses {
		items = append(items, itemsByStatus[strings.ToLower(status.Name)]...)
	}

	itemsLines := []string{"## Stories", ""}
	if len(items) > 0 {
		itemsLines = append(itemsLines, backlog.BacklogView{}.WriteMarkdownItemsWithProjectAndStatus(overviews, items, filepath.Dir(idea.Path()), tagsDir, s.userList)...)
	}
	idea.SetFooter(itemsLines)
	idea.Save()

	return nil
}

func (s *SyncIdeasStep) sendNewCommentsForIdeas(ideas []*backlog.BacklogIdea) {
	userList := backlog.NewUserList(s.root.UsersDirectory())
	var mailSender *utils.MailSender
	if s.cfg.SmtpServer != "" {
		mailSender = utils.NewMailSender(s.cfg.SmtpServer, s.cfg.SmtpUser, s.cfg.SmtpPassword, s.cfg.EmailFrom)
	}

	commented := make([]backlog.Commented, len(ideas))
	for i := range ideas {
		commented[i] = ideas[i]
	}
	sendNewComments(commented, func(idea backlog.Commented, to []string, comment []string) (me string, err error) {
		return sendComment(userList, comment, idea.Title(), s.author, to, mailSender, s.cfg, s.root.Root(), idea.Path())
	})
}
