package backlog

const (
	BacklogItemTitleHeader    = "title"
	BacklogItemAuthorHeader   = "author"
	BacklogItemStatusHeader   = "status"
	BacklogItemAssignedHeader = "assigned"
	BacklogItemEstimateHeader = "estimate"
)

type BacklogItem struct {
	markdown *MarkdownContent
}

func CreateBacklogItem(itemPath string) (*BacklogItem, error) {
	markdown, err := CreateMarkdown(itemPath, map[string]struct{}{
		BacklogItemTitleHeader: {}, CreatedHeader: {}, ModifiedHeader: {}, BacklogItemAuthorHeader: {},
		BacklogItemStatusHeader: {}, BacklogItemAssignedHeader: {}, BacklogItemEstimateHeader: {}})
	if err != nil {
		return nil, err
	}
	return &BacklogItem{markdown}, nil
}

func (item *BacklogItem) Save() error {
	return item.markdown.Save()
}

func (item *BacklogItem) Title() string {
	return item.markdown.Value(BacklogItemTitleHeader)
}

func (item *BacklogItem) SetTitle(title string) {
	item.markdown.SetValue(BacklogItemTitleHeader, title)
}

func (item *BacklogItem) SetCreated() {
	item.markdown.SetValue(CreatedHeader, "")
}

func (item *BacklogItem) SetModified() {
	item.markdown.SetValue(ModifiedHeader, "")
}

func (item *BacklogItem) Author() string {
	return item.markdown.Value(BacklogItemAuthorHeader)
}

func (item *BacklogItem) SetAuthor(author string) {
	item.markdown.SetValue(BacklogItemAuthorHeader, author)
}

func (item *BacklogItem) Status() string {
	return item.markdown.Value(BacklogItemStatusHeader)
}

func (item *BacklogItem) SetStatus(status string) {
	item.markdown.SetValue(BacklogItemStatusHeader, status)
}

func (item *BacklogItem) Assigned() string {
	return item.markdown.Value(BacklogItemAssignedHeader)
}

func (item *BacklogItem) SetAssigned(assigned string) {
	item.markdown.SetValue(BacklogItemAssignedHeader, assigned)
}

func (item *BacklogItem) Estimate() string {
	return item.markdown.Value(BacklogItemEstimateHeader)
}

func (item *BacklogItem) SetEstimate(estimate string) {
	item.markdown.SetValue(BacklogItemEstimateHeader, estimate)
}
