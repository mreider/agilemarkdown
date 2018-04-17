package backlog

const (
	BacklogOverviewTitleHeader = "title"
)

type BacklogOverview struct {
	markdown *MarkdownContent
}

func CreateBacklogOverview(overviewPath string) (*BacklogOverview, error) {
	markdown, err := CreateMarkdown(overviewPath, map[string]struct{}{BacklogOverviewTitleHeader: {}, CreatedHeader: {}, ModifiedHeader: {}})
	if err != nil {
		return nil, err
	}
	return &BacklogOverview{markdown}, nil
}

func (overview *BacklogOverview) Save() error {
	return overview.markdown.Save()
}

func (overview *BacklogOverview) Title() string {
	return overview.markdown.Value(BacklogOverviewTitleHeader)
}

func (overview *BacklogOverview) SetTitle(title string) {
	overview.markdown.SetValue(BacklogOverviewTitleHeader, title)
}

func (overview *BacklogOverview) SetCreated() {
	overview.markdown.SetValue(CreatedHeader, "")
}
