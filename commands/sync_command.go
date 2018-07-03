package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"
	"github.com/mreider/agilemarkdown/git"
	"github.com/mreider/agilemarkdown/users"
	"github.com/mreider/agilemarkdown/utils"
	"github.com/pkg/errors"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func NewSyncCommand() cli.Command {
	return cli.Command{
		Name:      "sync",
		Usage:     "Sync state",
		ArgsUsage: " ",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:   "test",
				Hidden: true,
			},
			cli.StringFlag{
				Name:   "author",
				Hidden: true,
			},
		},
		Action: func(c *cli.Context) error {
			action := &SyncAction{testMode: c.Bool("test"), author: c.String("author")}
			return action.Execute()
		},
	}
}

type SyncAction struct {
	testMode bool
	author   string
}

func (a *SyncAction) Execute() error {
	rootDir, _ := filepath.Abs(".")
	if err := checkIsBacklogDirectory(); err == nil {
		rootDir = filepath.Dir(rootDir)
	} else if err := checkIsRootDirectory("."); err != nil {
		return err
	}

	cfgPath := filepath.Join(rootDir, configName)
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("Can't load the config file %s: %v\n", cfgPath, err)
	}

	attempts := 10
	for attempts > 0 {
		attempts--

		err := a.updateOverviewsAndIndex(rootDir, cfg)
		if err != nil {
			return err
		}

		err = a.updateVelocity(rootDir, cfg)
		if err != nil {
			return err
		}

		err = a.updateIdeas(rootDir)
		if err != nil {
			return err
		}

		err = a.updateTags(rootDir)
		if err != nil {
			return err
		}

		if a.testMode {
			return nil
		}

		ok, err := a.syncToGit()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}
	return errors.New("can't sync: too many failed attempts")
}

func (a *SyncAction) updateOverviewsAndIndex(rootDir string, cfg *config.Config) error {
	backlogDirs, err := a.backlogDirs(rootDir)
	if err != nil {
		return err
	}
	indexPath := filepath.Join(rootDir, backlog.IndexFileName)
	index, err := backlog.LoadGlobalIndex(indexPath)
	if err != nil {
		return err
	}
	if len(index.FreeText()) == 0 {
		index.SetFreeText([]string{
			"# Agile Markdown",
			"",
			"Welcome to Agilemarkdown, an open source backlog manager that uses Markdown and Git. To read more about the project visit [agilemarkdown.com](http://agilemarkdown.com)",
			"",
		})
	}
	overviews := make([]*backlog.BacklogOverview, 0, len(backlogDirs))
	archives := make([]*backlog.BacklogOverview, 0, len(backlogDirs))
	for _, backlogDir := range backlogDirs {
		overviewPath, ok := findOverviewFileInRootDirectory(backlogDir)
		if !ok {
			return fmt.Errorf("the overview file isn't found for %s", backlogDir)
		}

		err = a.moveItemsToActiveAndArchiveDirectory(backlogDir)
		if err != nil {
			return err
		}

		overview, err := backlog.LoadBacklogOverview(overviewPath)
		if err != nil {
			return err
		}
		bck, err := backlog.LoadBacklog(backlogDir)
		if err != nil {
			return err
		}

		archivePath, _ := findArchiveFileInDirectory(backlogDir)
		archive, err := backlog.LoadBacklogOverview(archivePath)
		if err != nil {
			return err
		}
		archive.SetHideEmptyGroups(true)

		overviews = append(overviews, overview)
		archives = append(archives, archive)

		sorter := backlog.NewBacklogItemsSorter(overview, archive)

		activeItems := bck.ActiveItems()
		overview.UpdateLinks("archive", archivePath, rootDir, rootDir)
		overview.Update(activeItems, sorter)
		a.sendNewComments(cfg, rootDir, overview, activeItems)
		overview.Save()

		archivedItems := bck.ArchivedItems()
		archive.SetTitle(fmt.Sprintf("Archive: %s", overview.Title()))
		archive.UpdateLinks("project page", overviewPath, rootDir, backlogDir)
		archive.Update(archivedItems, sorter)
		archive.Save()

		overview.RemoveVelocity(bck)

		for _, item := range bck.AllItems() {
			item.SetHeader(fmt.Sprintf("Project: %s", overview.Title()))
			item.UpdateLinks(rootDir, overviewPath, archivePath)
		}
	}
	index.UpdateBacklogs(overviews, archives, rootDir)
	index.UpdateLinks(rootDir)

	return nil
}

func (a *SyncAction) updateVelocity(rootDir string, cfg *config.Config) error {
	backlogDirs, err := a.backlogDirs(rootDir)
	if err != nil {
		return err
	}
	velocityPath := filepath.Join(rootDir, backlog.VelocityFileName)
	velocity, err := backlog.LoadGlobalVelocity(velocityPath)
	if err != nil {
		return err
	}
	if velocity.Title() == "" {
		velocity.SetTitle("Velocity")
	}
	overviews := make([]*backlog.BacklogOverview, 0, len(backlogDirs))
	backlogs := make([]*backlog.Backlog, 0, len(backlogDirs))
	for _, backlogDir := range backlogDirs {
		overviewPath, ok := findOverviewFileInRootDirectory(backlogDir)
		if !ok {
			return fmt.Errorf("the overview file isn't found for %s", backlogDir)
		}

		overview, err := backlog.LoadBacklogOverview(overviewPath)
		if err != nil {
			return err
		}
		bck, err := backlog.LoadBacklog(backlogDir)
		if err != nil {
			return err
		}

		overviews = append(overviews, overview)
		backlogs = append(backlogs, bck)
	}
	velocity.Update(backlogs, overviews, backlogDirs, rootDir)
	velocity.UpdateLinks(rootDir)

	return nil
}

func (a *SyncAction) syncToGit() (bool, error) {
	err := git.AddAll()
	if err != nil {
		return false, err
	}
	git.Commit("sync", a.author) // TODO commit message
	err = git.Fetch()
	if err != nil {
		return false, fmt.Errorf("can't fetch: %v", err)
	}
	mergeOutput, mergeErr := git.Merge()
	if mergeErr != nil {
		status, _ := git.Status()
		if !strings.Contains(status, "Your branch is based on 'origin/master', but the upstream is gone.") {
			conflictFiles, conflictErr := git.ConflictFiles()
			hasConflictItems := false
			for _, fileName := range conflictFiles {
				if fileName == backlog.TagsFileName || strings.HasPrefix(fileName, backlog.TagsDirectoryName+string(os.PathSeparator)) {
					continue
				}

				fileName = strings.TrimSuffix(fileName, string(os.PathSeparator)+ArchiveFileName)
				if strings.Contains(fileName, "/") {
					hasConflictItems = true
					break
				}
			}
			if conflictErr != nil || hasConflictItems {
				fmt.Println(mergeOutput)
				git.AbortMerge()
				return false, fmt.Errorf("can't merge: %v", mergeErr)
			}
			for _, conflictFile := range conflictFiles {
				git.CheckoutOurVersion(conflictFile)
				git.Add(conflictFile)
				fmt.Printf("Remote changes to %s are ignored\n", conflictFile)
			}
			git.CommitNoEdit(a.author)
			return false, nil
		}
	}
	err = git.Push()
	if err != nil {
		return false, fmt.Errorf("can't push: %v", err)
	}
	return true, nil
}

func (a *SyncAction) backlogDirs(rootDir string) ([]string, error) {
	infos, err := ioutil.ReadDir(rootDir)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(infos))
	for _, info := range infos {
		if !info.IsDir() || strings.HasPrefix(info.Name(), ".") || backlog.IsForbiddenBacklogName(info.Name()) {
			continue
		}
		result = append(result, filepath.Join(rootDir, info.Name()))
	}
	sort.Strings(result)
	return result, nil
}

func (a *SyncAction) updateIdeas(rootDir string) error {
	ideasDir := filepath.Join(rootDir, backlog.IdeasDirectoryName)
	ideas, err := backlog.LoadIdeas(ideasDir)
	if err != nil {
		return err
	}

	ideasByRank := make(map[string][]*backlog.BacklogIdea)
	var ranks []string
	for _, idea := range ideas {
		err := a.updateIdea(rootDir, idea)
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
		fmt.Sprintf(utils.JoinMarkdownLinks(backlog.MakeStandardLinks(rootDir, rootDir)...)))
	lines = append(lines, "")
	for _, rank := range ranks {
		if rank != "" {
			lines = append(lines, fmt.Sprintf("## Rank: %s", strings.TrimSpace(rank)))
		} else {
			lines = append(lines, "## Rank: unassigned")
		}
		lines = append(lines, "")
		lines = append(lines, backlog.BacklogView{}.WriteMarkdownIdeas(ideasByRank[strings.TrimSpace(rank)], rootDir, filepath.Join(rootDir, backlog.TagsDirectoryName))...)
		lines = append(lines, "")
	}
	return ioutil.WriteFile(filepath.Join(rootDir, backlog.IdeasFileName), []byte(strings.Join(lines, "\n")), 0644)
}

func (a *SyncAction) updateIdea(rootDir string, idea *backlog.BacklogIdea) error {
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
	idea.UpdateLinks(rootDir)
	return nil
}

func (a *SyncAction) moveItemsToActiveAndArchiveDirectory(backlogDir string) error {
	bck, err := backlog.LoadBacklog(backlogDir)
	if err != nil {
		return err
	}

	for _, item := range bck.ActiveItems() {
		err := item.MoveToBacklogDirectory()
		if err != nil {
			return err
		}
	}

	for _, item := range bck.ArchivedItems() {
		err := item.MoveToBacklogArchiveDirectory()
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *SyncAction) updateTags(rootDir string) error {
	backlogDirs, err := a.backlogDirs(rootDir)
	if err != nil {
		return err
	}
	tagsDir := filepath.Join(rootDir, backlog.TagsDirectoryName)
	os.MkdirAll(tagsDir, 0777)

	ideasDir := filepath.Join(rootDir, backlog.IdeasDirectoryName)
	ideas, err := backlog.LoadIdeas(ideasDir)
	if err != nil {
		return err
	}

	allTags := make(map[string]struct{})
	itemsTags := make(map[string][]*backlog.BacklogItem)
	ideasTags := make(map[string][]*backlog.BacklogIdea)

	overviews := make(map[*backlog.BacklogItem]*backlog.BacklogOverview)
	for _, backlogDir := range backlogDirs {
		overviewPath, ok := findOverviewFileInRootDirectory(backlogDir)
		if !ok {
			return fmt.Errorf("the overview file isn't found for %s", backlogDir)
		}
		overview, err := backlog.LoadBacklogOverview(overviewPath)
		if err != nil {
			return err
		}

		bck, err := backlog.LoadBacklog(backlogDir)
		if err != nil {
			return err
		}

		items := bck.ActiveItems()
		for _, item := range items {
			for _, tag := range item.Tags() {
				allTags[tag] = struct{}{}
				itemsTags[tag] = append(itemsTags[tag], item)
				overviews[item] = overview
			}
		}
	}

	for _, idea := range ideas {
		for _, tag := range idea.Tags() {
			allTags[tag] = struct{}{}
			ideasTags[tag] = append(ideasTags[tag], idea)
		}
	}

	tagsFileNames := make(map[string]bool)
	for tag := range allTags {
		tagItems := itemsTags[tag]
		tagIdeas := ideasTags[tag]
		tagFileName, err := a.updateTagPage(rootDir, tagsDir, tag, tagItems, overviews, tagIdeas)
		if err != nil {
			return err
		}
		tagsFileNames[tagFileName] = true
	}

	infos, _ := ioutil.ReadDir(tagsDir)
	for _, info := range infos {
		if _, ok := tagsFileNames[info.Name()]; !ok {
			os.Remove(filepath.Join(tagsDir, info.Name()))
		}
	}

	err = a.updateTagsPage(rootDir, tagsDir, itemsTags, ideasTags)
	if err != nil {
		return err
	}

	return nil
}

func (a *SyncAction) updateTagPage(rootDir, tagsDir, tag string, items []*backlog.BacklogItem, overviews map[*backlog.BacklogItem]*backlog.BacklogOverview, ideas []*backlog.BacklogIdea) (string, error) {
	itemsByStatus := make(map[string][]*backlog.BacklogItem)
	for _, item := range items {
		itemStatus := strings.ToLower(item.Status())
		itemsByStatus[itemStatus] = append(itemsByStatus[itemStatus], item)
	}
	for _, statusItems := range itemsByStatus {
		sorter := backlog.NewBacklogItemsSorter()
		sorter.SortItemsByModifiedDesc(statusItems)
	}

	lines := []string{
		fmt.Sprintf("# Tag: %s", tag),
		"",
		fmt.Sprintf(utils.JoinMarkdownLinks(backlog.MakeStandardLinks(rootDir, tagsDir)...)),
		"",
	}
	for _, status := range backlog.AllStatuses {
		statusItems := itemsByStatus[strings.ToLower(status.Name)]
		if len(statusItems) == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("## %s", status.CapitalizedName()))
		itemsLines := backlog.BacklogView{}.WriteMarkdownItemsWithProject(overviews, statusItems, tagsDir, tagsDir)
		lines = append(lines, itemsLines...)
		lines = append(lines, "")
	}
	if len(ideas) > 0 {
		lines = append(lines, "## Ideas")
		lines = append(lines, "")
		ideasLines := backlog.BacklogView{}.WriteMarkdownIdeas(ideas, tagsDir, tagsDir)
		lines = append(lines, ideasLines...)
		lines = append(lines, "")
	}
	tagFileName := fmt.Sprintf("%s.md", utils.GetValidFileName(tag))
	err := ioutil.WriteFile(filepath.Join(tagsDir, tagFileName), []byte(strings.Join(lines, "\n")), 0644)
	return tagFileName, err
}

func (a *SyncAction) updateTagsPage(rootDir, tagsDir string, itemsTags map[string][]*backlog.BacklogItem, ideasTags map[string][]*backlog.BacklogIdea) error {
	allTagsSet := make(map[string]bool)
	allTags := make([]string, 0, len(itemsTags)+len(ideasTags))
	for tag := range itemsTags {
		if !allTagsSet[tag] {
			allTagsSet[tag] = true
			allTags = append(allTags, tag)
		}
	}
	for tag := range ideasTags {
		if !allTagsSet[tag] {
			allTagsSet[tag] = true
			allTags = append(allTags, tag)
		}
	}
	sort.Strings(allTags)

	lines := []string{"# Tags", ""}
	lines = append(lines, fmt.Sprintf(utils.JoinMarkdownLinks(backlog.MakeStandardLinks(rootDir, rootDir)...)))
	lines = append(lines, "", "---", "")
	for _, tag := range allTags {
		lines = append(lines, fmt.Sprintf("%s", backlog.MakeTagLink(tag, tagsDir, rootDir)))
	}
	return ioutil.WriteFile(filepath.Join(rootDir, backlog.TagsFileName), []byte(strings.Join(lines, "  \n")), 0644)
}

func (a *SyncAction) sendNewComments(cfg *config.Config, rootDir string, overview *backlog.BacklogOverview, activeItems []*backlog.BacklogItem) {
	userList := users.NewUserList(filepath.Join(rootDir, backlog.UsersDirectoryName))
	var mailSender *utils.MailSender
	if cfg.SmtpServer != "" {
		mailSender = utils.NewMailSender(cfg.SmtpServer, cfg.SmtpUser, cfg.SmtpPassword, cfg.EmailFrom)
	}
	remoteOriginUrl, _ := git.RemoteOriginUrl()
	remoteOriginUrl = strings.TrimSuffix(remoteOriginUrl, ".git")

	from := a.author
	sepIndex := strings.LastIndexByte(from, ' ')
	if sepIndex >= 0 {
		from = from[sepIndex+1:]
		from = strings.Trim(from, "<>")
	}
	if from == "" {
		from, _, _ = git.CurrentUser()
	}
	overview.SendNewComments(activeItems, func(item *backlog.BacklogItem, to []string, comment []string) (me string, err error) {
		meUser := userList.User(from)
		if meUser == nil {
			return "", fmt.Errorf("unknown user %s", from)
		}
		toUsers := make([]*users.User, 0, len(to))
		for _, user := range to {
			toUser := userList.User(user)
			if toUser == nil {
				return meUser.Nick(), fmt.Errorf("unknown user %s", to)
			}
			toUsers = append(toUsers, toUser)
		}
		if mailSender == nil {
			return meUser.Nick(), errors.New("SMTP server isn't configured")
		}

		msgText := strings.Join(comment, "\n")
		if remoteOriginUrl != "" {
			var itemGitUrl string
			itemPath := strings.TrimPrefix(item.Path(), rootDir)
			itemPath = strings.TrimPrefix(itemPath, string(os.PathSeparator))
			itemPath = strings.Replace(itemPath, string(os.PathSeparator), "/", -1)
			if cfg.RemoteGitUrlFormat != "" {
				itemGitUrl = fmt.Sprintf(cfg.RemoteGitUrlFormat, remoteOriginUrl, itemPath)
			} else {
				itemGitUrl = fmt.Sprintf("%s/%s", remoteOriginUrl, itemPath)
			}
			msgText += fmt.Sprintf("\n\nView on Git: %s\n", itemGitUrl)
			if cfg.RemoteWebUrlFormat != "" {
				itemWebUrl := fmt.Sprintf(cfg.RemoteWebUrlFormat, itemPath)
				msgText += fmt.Sprintf("View on the web: %s\n", itemWebUrl)
			}
		}

		fromSubject := meUser.Nick()
		if meUser.Name() != meUser.Nick() {
			fromSubject += fmt.Sprintf(" (%s)", meUser.Name())
		}

		toEmails := make([]string, 0, len(toUsers))
		for _, user := range toUsers {
			toEmails = append(toEmails, user.Email())
		}

		err = mailSender.SendEmail(
			toEmails,
			fmt.Sprintf("%s. New comment from %s", overview.Title(), fromSubject),
			msgText)
		return meUser.Nick(), err
	})
}
