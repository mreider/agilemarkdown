package actions

import (
	"errors"
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"
	"github.com/mreider/agilemarkdown/git"
	"github.com/mreider/agilemarkdown/utils"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type SyncAction struct {
	rootDir    string
	configName string
	author     string
	testMode   bool
}

func NewSyncAction(rootDir, configName, author string, testMode bool) *SyncAction {
	return &SyncAction{rootDir: rootDir, configName: configName, author: author, testMode: testMode}
}

func (a *SyncAction) Execute() error {
	cfgPath := filepath.Join(a.rootDir, a.configName)
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("Can't load the config file %s: %v\n", cfgPath, err)
	}

	userList := backlog.NewUserList(filepath.Join(a.rootDir, backlog.UsersDirectoryName))

	attempts := 10
	for attempts > 0 {
		attempts--

		err := a.updateItemsModifiedDate(a.rootDir)
		if err != nil {
			return err
		}

		err = a.updateItemsFileNames(a.rootDir)
		if err != nil {
			return err
		}

		err = a.updateOverviewsAndIndex(a.rootDir, cfg, userList)
		if err != nil {
			return err
		}

		err = a.updateVelocity(a.rootDir, cfg)
		if err != nil {
			return err
		}

		err = a.updateIdeas(a.rootDir, cfg, userList)
		if err != nil {
			return err
		}

		err = a.updateTags(a.rootDir, userList)
		if err != nil {
			return err
		}

		err = a.updateUsers(a.rootDir)
		if err != nil {
			return err
		}

		err = a.updateTimeline(a.rootDir)
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

func (a *SyncAction) updateOverviewsAndIndex(rootDir string, cfg *config.Config, userList *backlog.UserList) error {
	backlogDirs, err := backlog.BacklogDirs(rootDir)
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
			"Welcome aboard",
			"",
		})
	}
	overviews := make([]*backlog.BacklogOverview, 0, len(backlogDirs))
	archives := make([]*backlog.BacklogOverview, 0, len(backlogDirs))
	for _, backlogDir := range backlogDirs {
		overviewPath, ok := backlog.FindOverviewFileInRootDirectory(backlogDir)
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

		archivePath, _ := backlog.FindArchiveFileInDirectory(backlogDir)
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
		overview.Update(activeItems, sorter, userList)
		a.sendNewCommentsForItems(cfg, rootDir, overview, activeItems)
		overview.Save()

		archivedItems := bck.ArchivedItems()
		archive.SetTitle(fmt.Sprintf("Archive: %s", overview.Title()))
		archive.UpdateLinks("project page", overviewPath, rootDir, backlogDir)
		archive.Update(archivedItems, sorter, userList)
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
	backlogDirs, err := backlog.BacklogDirs(rootDir)
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
		overviewPath, ok := backlog.FindOverviewFileInRootDirectory(backlogDir)
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

				fileName = strings.TrimSuffix(fileName, string(os.PathSeparator)+backlog.ArchiveFileName)
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

func (a *SyncAction) updateIdeas(rootDir string, cfg *config.Config, userList *backlog.UserList) error {
	_, itemsTags, ideasTags, overviews, err := backlog.ItemsAndIdeasTags(rootDir)
	if err != nil {
		return err
	}

	ideasDir := filepath.Join(rootDir, backlog.IdeasDirectoryName)
	ideas, err := backlog.LoadIdeas(ideasDir)
	if err != nil {
		return err
	}

	a.sendNewCommentsForIdeas(cfg, rootDir, ideas)

	ideasByRank := make(map[string][]*backlog.BacklogIdea)
	var ranks []string
	tagsDir := filepath.Join(rootDir, backlog.TagsDirectoryName)
	for _, idea := range ideas {
		err := a.updateIdea(rootDir, tagsDir, idea, ideasTags, itemsTags, overviews, userList)
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

func (a *SyncAction) updateIdea(rootDir, tagsDir string, idea *backlog.BacklogIdea, ideasTags map[string][]*backlog.BacklogIdea, itemsTags map[string][]*backlog.BacklogItem, overviews map[*backlog.BacklogItem]*backlog.BacklogOverview, userList *backlog.UserList) error {
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
		itemsLines = append(itemsLines, backlog.BacklogView{}.WriteMarkdownItemsWithProjectAndStatus(overviews, items, filepath.Dir(idea.Path()), tagsDir, userList)...)
	}
	idea.SetFooter(itemsLines)
	idea.Save()

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

func (a *SyncAction) updateTags(rootDir string, userList *backlog.UserList) error {
	tagsDir := filepath.Join(rootDir, backlog.TagsDirectoryName)
	os.MkdirAll(tagsDir, 0777)

	allTags, itemsTags, ideasTags, overviews, err := backlog.ItemsAndIdeasTags(rootDir)
	if err != nil {
		return err
	}

	tagsFileNames := make(map[string]bool)
	for tag := range allTags {
		tagItems := itemsTags[tag]
		tagIdeas := ideasTags[tag]
		tagFileName, err := a.updateTagPage(rootDir, tagsDir, tag, tagItems, overviews, tagIdeas, userList)
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

func (a *SyncAction) updateTagPage(rootDir, tagsDir, tag string, items []*backlog.BacklogItem, overviews map[*backlog.BacklogItem]*backlog.BacklogOverview, ideas []*backlog.BacklogIdea, userList *backlog.UserList) (string, error) {
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
		itemsLines := backlog.BacklogView{}.WriteMarkdownItemsWithProject(overviews, statusItems, status, tagsDir, tagsDir, userList)
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

func (a *SyncAction) sendNewCommentsForItems(cfg *config.Config, rootDir string, overview *backlog.BacklogOverview, items []*backlog.BacklogItem) {
	userList := backlog.NewUserList(filepath.Join(rootDir, backlog.UsersDirectoryName))
	var mailSender *utils.MailSender
	if cfg.SmtpServer != "" {
		mailSender = utils.NewMailSender(cfg.SmtpServer, cfg.SmtpUser, cfg.SmtpPassword, cfg.EmailFrom)
	}

	commented := make([]backlog.Commented, len(items))
	for i := range items {
		commented[i] = items[i]
	}
	a.sendNewComments(commented, func(item backlog.Commented, to []string, comment []string) (me string, err error) {
		return a.sendComment(userList, comment, overview.Title(), to, mailSender, cfg, rootDir, item.Path())
	})
}

func (a *SyncAction) sendNewCommentsForIdeas(cfg *config.Config, rootDir string, ideas []*backlog.BacklogIdea) {
	userList := backlog.NewUserList(filepath.Join(rootDir, backlog.UsersDirectoryName))
	var mailSender *utils.MailSender
	if cfg.SmtpServer != "" {
		mailSender = utils.NewMailSender(cfg.SmtpServer, cfg.SmtpUser, cfg.SmtpPassword, cfg.EmailFrom)
	}

	commented := make([]backlog.Commented, len(ideas))
	for i := range ideas {
		commented[i] = ideas[i]
	}
	a.sendNewComments(commented, func(idea backlog.Commented, to []string, comment []string) (me string, err error) {
		return a.sendComment(userList, comment, idea.Title(), to, mailSender, cfg, rootDir, idea.Path())
	})
}

func (a *SyncAction) updateItemsModifiedDate(rootDir string) error {
	backlogDirs, err := backlog.BacklogDirs(rootDir)
	if err != nil {
		return err
	}
	for _, backlogDir := range backlogDirs {
		modifiedFiles, err := git.ModifiedFiles(backlogDir)
		if len(modifiedFiles) == 0 {
			continue
		}

		modifiedFilesSet := make(map[string]bool)
		for _, file := range modifiedFiles {
			modifiedFilesSet[file] = true
		}

		bck, err := backlog.LoadBacklog(backlogDir)
		if err != nil {
			return err
		}
		for _, item := range bck.AllItems() {
			if modifiedFilesSet[filepath.Base(item.Path())] {
				itemPath, _ := filepath.Rel(rootDir, item.Path())
				itemPath = fmt.Sprintf("./%s", itemPath)
				repoItemContent, err := git.RepoVersion(rootDir, itemPath)
				if err != nil {
					return err
				}
				repoItem := backlog.NewBacklogItem(filepath.Base(itemPath), repoItemContent)
				currentTimestamp := utils.GetCurrentTimestamp()
				if item.Assigned() != repoItem.Assigned() || item.Status() != repoItem.Status() || item.Estimate() != repoItem.Estimate() {
					if item.Modified() == repoItem.Modified() {
						item.SetModified(currentTimestamp)
						item.Save()
					}
				}

				oldStatus := backlog.StatusByName(repoItem.Status())
				newStatus := backlog.StatusByName(item.Status())
				if oldStatus != newStatus {
					if newStatus == backlog.FinishedStatus {
						item.SetFinished(currentTimestamp)
						item.Save()
					} else if oldStatus == backlog.FinishedStatus {
						item.SetFinished("")
						item.Save()
					} else {
						if !item.Finished().IsZero() {
							item.SetFinished("")
							item.Save()
						}
					}
				} else if oldStatus == backlog.FinishedStatus && newStatus == backlog.FinishedStatus {
					if item.Finished().IsZero() && !repoItem.Finished().IsZero() {
						item.SetFinished(utils.GetTimestamp(repoItem.Finished()))
						item.Save()
					}
				} else if oldStatus == newStatus {
					if !item.Finished().IsZero() {
						item.SetFinished("")
						item.Save()
					}
				}
			}
		}
	}
	return nil
}

func (a *SyncAction) updateItemsFileNames(rootDir string) error {
	backlogDirs, err := backlog.BacklogDirs(rootDir)
	if err != nil {
		return err
	}
	for _, backlogDir := range backlogDirs {
		overviewPath, ok := backlog.FindOverviewFileInRootDirectory(backlogDir)
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
		for _, item := range bck.AllItems() {
			currentItemName := strings.ToLower(filepath.Base(item.Path()))
			expectedItemName := strings.ToLower(utils.GetValidFileName(item.Title()) + ".md")
			if currentItemName != expectedItemName {
				newItemPath := filepath.Join(filepath.Dir(item.Path()), expectedItemName)
				if _, err := os.Stat(newItemPath); os.IsNotExist(err) {
					err := os.Rename(item.Path(), newItemPath)
					if err == nil {
						git.Add(newItemPath)
						overview.UpdateItemLinkInOverviewFile(item.Path(), newItemPath)
					}
				}
			}
		}
	}
	return nil
}

func (a *SyncAction) updateTimeline(rootDir string) error {
	allTags, itemsTags, _, _, err := backlog.ItemsAndIdeasTags(rootDir)
	if err != nil {
		return err
	}

	timelineGenerator := backlog.NewTimelineGenerator(rootDir)
	for tag, tagItems := range itemsTags {
		hasTimeline := false
		for _, item := range tagItems {
			startDate, endDate := item.Timeline(tag)
			if !startDate.IsZero() && !endDate.IsZero() {
				hasTimeline = true
				break
			}
		}
		if hasTimeline {
			timelineGenerator.ExecuteForTag(tag)
		} else {
			timelineGenerator.RemoveTimeline(tag)
		}
	}

	timelineDir := filepath.Join(rootDir, backlog.TimelineDirectoryName)
	items, err := ioutil.ReadDir(timelineDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	lines := []string{"# Timelines", ""}
	lines = append(lines,
		fmt.Sprintf(utils.JoinMarkdownLinks(backlog.MakeStandardLinks(rootDir, rootDir)...)))
	lines = append(lines, "")

	for _, item := range items {
		if strings.HasSuffix(item.Name(), ".png") {
			timelineImagePath := filepath.Join(timelineDir, item.Name())
			timelineTag := strings.TrimSuffix(item.Name(), ".png")
			if _, ok := allTags[timelineTag]; ok {
				lines = append(lines, fmt.Sprintf("## Tag: %s", utils.MakeMarkdownLink(timelineTag, filepath.Join(rootDir, backlog.TagsDirectoryName, timelineTag), rootDir)))
				lines = append(lines, "")
				lines = append(lines, fmt.Sprintf("%s", utils.MakeMarkdownImageLink(timelineTag, timelineImagePath, rootDir)))
				lines = append(lines, "")
			} else {
				os.Remove(timelineImagePath)
			}
		}
	}

	return ioutil.WriteFile(filepath.Join(rootDir, backlog.TimelineFileName), []byte(strings.Join(lines, "\n")), 0644)
}

func (a *SyncAction) RemoteOriginUrl() string {
	remoteOriginUrl, _ := git.RemoteOriginUrl()
	return strings.TrimSuffix(remoteOriginUrl, ".git")
}

func (a *SyncAction) FromUser() string {
	from := a.author
	sepIndex := strings.LastIndexByte(from, ' ')
	if sepIndex >= 0 {
		from = from[sepIndex+1:]
		from = strings.Trim(from, "<>")
	}
	if from == "" {
		from, _, _ = git.CurrentUser()
	}
	return from
}

func (a *SyncAction) sendComment(userList *backlog.UserList, comment []string, title string, to []string, mailSender *utils.MailSender, cfg *config.Config, rootDir, contentPath string) (me string, err error) {
	from := a.author
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

func (a *SyncAction) sendNewComments(items []backlog.Commented, onSend func(item backlog.Commented, to []string, comment []string) (me string, err error)) {
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

func (a *SyncAction) updateUsers(rootDir string) error {
	userList := backlog.NewUserList(filepath.Join(rootDir, backlog.UsersDirectoryName))
	tagsDir := filepath.Join(rootDir, backlog.TagsDirectoryName)

	items, overviews, err := backlog.ActiveBacklogItems(rootDir)
	if err != nil {
		return err
	}

	for _, user := range userList.Users() {
		userName, userNick := strings.ToLower(user.Name()), strings.ToLower(user.Nickname())
		var userItems []*backlog.BacklogItem
		for _, item := range items {
			assigned := strings.ToLower(utils.CollapseWhiteSpaces(item.Assigned()))
			if assigned == userNick || assigned == userName {
				userItems = append(userItems, item)
			}
		}

		_, err := user.UpdateItems(rootDir, tagsDir, userItems, overviews)
		if err != nil {
			return err
		}
	}
	return a.updateUsersPage(userList, rootDir)
}

func (a *SyncAction) updateUsersPage(userList *backlog.UserList, rootDir string) error {
	lines := []string{"# Users", ""}
	lines = append(lines, fmt.Sprintf(utils.JoinMarkdownLinks(backlog.MakeStandardLinks(rootDir, rootDir)...)))
	lines = append(lines, "", "---", "")
	lines = append(lines, fmt.Sprintf("| Name | Nickname | Email |"))
	lines = append(lines, "|---|---|---|")
	for _, user := range userList.Users() {
		lines = append(lines, fmt.Sprintf("| %s | %s | %s |", backlog.MakeUserLink(user, user.Name(), rootDir), user.Nickname(), strings.Join(user.Emails(), ", ")))
	}
	return ioutil.WriteFile(filepath.Join(rootDir, backlog.UsersFileName), []byte(strings.Join(lines, "  \n")), 0644)
}
