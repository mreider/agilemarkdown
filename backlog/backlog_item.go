package backlog

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mreider/agilemarkdown/markdown"
)

const (
	itemKeyTitle    = "title"
	itemKeyProject  = "project"
	itemKeyStatus   = "status"
	itemKeyAssigned = "assigned"
	itemKeyEstimate = "estimate"
	itemKeyTags     = "tags"
	itemKeyAuthor   = "author"
	itemKeyCreated  = "created"
	itemKeyModified = "modified"
	itemKeyStarted   = "started"
	itemKeyFinished  = "finished"
	itemKeyDelivered = "delivered"
	itemKeyAccepted  = "accepted"
	itemKeyType       = "type"
	itemKeyArchive    = "archive"
	itemKeyTimeline   = "timeline"
	itemKeyHypothesis  = "hypothesis"
	itemKeyEpic        = "epic"
	itemKeyReleaseDate = "release_date"
	itemKeyBlocked       = "blocked"
	itemKeyBlockedReason = "blocked_reason"

	timelineKeyStart = "start"
	timelineKeyEnd   = "end"
)

const itemDateLayout = "2006-01-02"

var (
	commentsTitleRe        = regexp.MustCompile(`(?i)^#{1,3}\s+Comments\s*$`)
	commentRe              = regexp.MustCompile(`^(\s*)((@[\w.\-_]+[\s,;]+)+)(.*)$`)
	commentUserSeparatorRe = regexp.MustCompile(`[\s,;]+`)
)

type BacklogItem struct {
	name string
	file *markdown.FrontmatterFile
}

func LoadBacklogItem(itemPath string) (*BacklogItem, error) {
	f, err := markdown.LoadFrontmatter(itemPath)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSuffix(filepath.Base(itemPath), filepath.Ext(itemPath))
	return &BacklogItem{name: name, file: f}, nil
}

func NewBacklogItem(name string, markdownData string) *BacklogItem {
	f, _ := markdown.ParseFrontmatter(markdownData)
	return &BacklogItem{name: name, file: f}
}

func (item *BacklogItem) Save() error                { return item.file.Save() }
func (item *BacklogItem) Name() string               { return item.name }
func (item *BacklogItem) Path() string               { return item.file.Path() }
func (item *BacklogItem) Content() []byte            { return item.file.Bytes() }
func (item *BacklogItem) Title() string              { return item.file.GetString(itemKeyTitle) }
func (item *BacklogItem) SetTitle(title string)      { item.file.SetString(itemKeyTitle, title) }
func (item *BacklogItem) Project() string            { return item.file.GetString(itemKeyProject) }
func (item *BacklogItem) SetProject(project string)  { item.file.SetString(itemKeyProject, project) }
func (item *BacklogItem) Status() string             { return item.file.GetString(itemKeyStatus) }
// Assigned returns the first assignee (or comma-joined display string for
// legacy callers). Most call sites are read-only string handlers; for
// multi-owner work use Assignees().
func (item *BacklogItem) Assigned() string {
	xs := item.Assignees()
	if len(xs) == 0 {
		return ""
	}
	return strings.Join(xs, ", ")
}

// Assignees returns the assignee list. A scalar `assigned: foo` becomes a
// one-element list; a YAML sequence becomes a multi-element list.
func (item *BacklogItem) Assignees() []string {
	xs := item.file.GetStringSlice(itemKeyAssigned)
	out := make([]string, 0, len(xs))
	for _, s := range xs {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// SetAssigned writes a single assignee as a YAML scalar. Empty clears.
func (item *BacklogItem) SetAssigned(s string) { item.file.SetString(itemKeyAssigned, s) }

// SetAssignees writes one or more assignees. A single value writes a YAML
// scalar (back-compat with v4.2 files); two or more values write a flow
// sequence. Empty clears.
func (item *BacklogItem) SetAssignees(xs []string) {
	clean := make([]string, 0, len(xs))
	seen := make(map[string]bool)
	for _, s := range xs {
		s = strings.TrimSpace(s)
		if s == "" || seen[strings.ToLower(s)] {
			continue
		}
		seen[strings.ToLower(s)] = true
		clean = append(clean, s)
	}
	if len(clean) == 0 {
		item.file.SetString(itemKeyAssigned, "")
		return
	}
	if len(clean) == 1 {
		item.file.SetString(itemKeyAssigned, clean[0])
		return
	}
	item.file.SetStringSlice(itemKeyAssigned, clean)
}
func (item *BacklogItem) Estimate() string           { return item.file.GetString(itemKeyEstimate) }
func (item *BacklogItem) SetEstimate(estimate string) {
	item.file.SetString(itemKeyEstimate, estimate)
}
func (item *BacklogItem) Author() string         { return item.file.GetString(itemKeyAuthor) }
func (item *BacklogItem) SetAuthor(s string)     { item.file.SetString(itemKeyAuthor, s) }
func (item *BacklogItem) SetCreated(ts string)   { item.file.SetString(itemKeyCreated, ts) }
func (item *BacklogItem) SetModified(ts string)  { item.file.SetString(itemKeyModified, ts) }
func (item *BacklogItem) SetFinished(ts string)  { item.file.SetString(itemKeyFinished, ts) }
func (item *BacklogItem) Delivered() time.Time   { return parseTimestamp(item.file.GetString(itemKeyDelivered)) }
func (item *BacklogItem) SetDelivered(ts string) { item.file.SetString(itemKeyDelivered, ts) }
func (item *BacklogItem) Accepted() time.Time    { return parseTimestamp(item.file.GetString(itemKeyAccepted)) }
func (item *BacklogItem) SetAccepted(ts string)  { item.file.SetString(itemKeyAccepted, ts) }
func (item *BacklogItem) Hypothesis() string     { return item.file.GetString(itemKeyHypothesis) }
func (item *BacklogItem) SetHypothesis(s string)  { item.file.SetString(itemKeyHypothesis, s) }
func (item *BacklogItem) Epic() string            { return strings.TrimSpace(item.file.GetString(itemKeyEpic)) }
func (item *BacklogItem) SetEpic(s string)        { item.file.SetString(itemKeyEpic, strings.TrimSpace(s)) }
func (item *BacklogItem) ReleaseDate() string     { return strings.TrimSpace(item.file.GetString(itemKeyReleaseDate)) }
func (item *BacklogItem) SetReleaseDate(s string) { item.file.SetString(itemKeyReleaseDate, strings.TrimSpace(s)) }
func (item *BacklogItem) Type() string           { return strings.ToLower(strings.TrimSpace(item.file.GetString(itemKeyType))) }
func (item *BacklogItem) SetType(t string)       { item.file.SetString(itemKeyType, strings.ToLower(strings.TrimSpace(t))) }
func (item *BacklogItem) Created() time.Time     { return parseTimestamp(item.file.GetString(itemKeyCreated)) }
func (item *BacklogItem) Modified() time.Time    { return parseTimestamp(item.file.GetString(itemKeyModified)) }
func (item *BacklogItem) Started() time.Time     { return parseTimestamp(item.file.GetString(itemKeyStarted)) }
func (item *BacklogItem) SetStarted(ts string)   { item.file.SetString(itemKeyStarted, ts) }
func (item *BacklogItem) Finished() time.Time    { return parseTimestamp(item.file.GetString(itemKeyFinished)) }

func (item *BacklogItem) SetStatus(status *BacklogItemStatus) {
	item.file.SetString(itemKeyStatus, status.Name)
}

func (item *BacklogItem) Tags() []string {
	tags := item.file.GetStringSlice(itemKeyTags)
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func (item *BacklogItem) SetTags(tags []string) {
	clean := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t != "" {
			clean = append(clean, t)
		}
	}
	item.file.SetStringSlice(itemKeyTags, clean)
}

func (item *BacklogItem) Archived() bool { return item.file.GetBool(itemKeyArchive) }

func (item *BacklogItem) SetArchived(archived bool) {
	item.file.SetBool(itemKeyArchive, archived)
}

func (item *BacklogItem) TimelineStr() (start, end string) {
	tl := item.file.GetMap(itemKeyTimeline)
	for _, kv := range tl {
		switch kv.Key {
		case timelineKeyStart:
			start = kv.Value
		case timelineKeyEnd:
			end = kv.Value
		}
	}
	return start, end
}

func (item *BacklogItem) Timeline() (start, end time.Time) {
	s, e := item.TimelineStr()
	start, _ = time.Parse(itemDateLayout, s)
	end, _ = time.Parse(itemDateLayout, e)
	return start, end
}

func (item *BacklogItem) SetTimeline(start, end time.Time) {
	item.file.SetMap(itemKeyTimeline, []markdown.KV{
		{Key: timelineKeyStart, Value: start.Format(itemDateLayout)},
		{Key: timelineKeyEnd, Value: end.Format(itemDateLayout)},
	})
}

func (item *BacklogItem) ClearTimeline() { item.file.Remove(itemKeyTimeline) }

func (item *BacklogItem) SetDescription(description string) {
	item.file.SetBody(description)
}

func (item *BacklogItem) Body() string         { return item.file.Body() }
func (item *BacklogItem) SetBody(body string)  { item.file.SetBody(body) }
func (item *BacklogItem) File() *markdown.FrontmatterFile { return item.file }

func (item *BacklogItem) Blocked() bool { return item.file.GetBool(itemKeyBlocked) }

func (item *BacklogItem) BlockedReason() string {
	return strings.TrimSpace(item.file.GetString(itemKeyBlockedReason))
}

func (item *BacklogItem) SetBlocked(blocked bool, reason string) {
	item.file.SetBool(itemKeyBlocked, blocked)
	if blocked {
		item.file.SetString(itemKeyBlockedReason, strings.TrimSpace(reason))
		return
	}
	item.file.SetString(itemKeyBlockedReason, "")
}

func (item *BacklogItem) Comments() []*Comment {
	return parseBodyComments(item.file.Body())
}

func (item *BacklogItem) UpdateComments(comments []*Comment) error {
	body := updateBodyComments(item.file.Body(), comments)
	item.file.SetBody(body)
	return item.Save()
}

func (item *BacklogItem) MoveToBacklogDirectory() error {
	dir := filepath.Dir(item.file.Path())
	if filepath.Base(dir) == archiveDirectoryName {
		newPath := filepath.Join(filepath.Dir(dir), filepath.Base(item.file.Path()))
		if err := os.Rename(item.file.Path(), newPath); err != nil {
			return err
		}
		item.file.SetPath(newPath)
	}
	return nil
}

func (item *BacklogItem) MoveToBacklogArchiveDirectory() error {
	dir := filepath.Dir(item.file.Path())
	if filepath.Base(dir) != archiveDirectoryName {
		newPath := filepath.Join(dir, archiveDirectoryName, filepath.Base(item.file.Path()))
		if err := os.MkdirAll(filepath.Dir(newPath), 0777); err != nil {
			return err
		}
		if err := os.Rename(item.file.Path(), newPath); err != nil {
			return err
		}
		item.file.SetPath(newPath)
	}
	return nil
}

// estimateAsFloat returns the numeric value of the estimate or 0.
func (item *BacklogItem) estimateAsFloat() float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(item.Estimate()), 64)
	return v
}

func parseTimestamp(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 03:04 PM",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
