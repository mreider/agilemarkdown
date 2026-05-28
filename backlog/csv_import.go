package backlog

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"github.com/mreider/agilemarkdown/utils"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	startsFromCapitalLetter = regexp.MustCompile(`^[A-Z][a-z].*`)
	spacesRe                = regexp.MustCompile(`\s+`)
	delimiterRe             = regexp.MustCompile(`\s*,\s*`)
)

type CsvImporter struct {
	csvPath    string
	backlogDir string

	headers map[string][]int
}

func NewCsvImporter(csvPath string, backlogDir string) *CsvImporter {
	return &CsvImporter{csvPath: csvPath, backlogDir: backlogDir}
}

func (imp *CsvImporter) Import() error {
	root := NewBacklogsStructure(filepath.Join(imp.backlogDir, ".."))
	userList := NewUserList(root.UsersDirectory())

	csvFile, err := os.Open(imp.csvPath)
	if err != nil {
		return err
	}
	defer csvFile.Close()
	reader := csv.NewReader(bufio.NewReader(csvFile))
	reader.FieldsPerRecord = -1
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		if imp.headers == nil {
			imp.parseHeaders(line)
		} else {
			err := imp.createItemIfNotExists(line, userList)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (imp *CsvImporter) parseHeaders(line []string) {
	imp.headers = make(map[string][]int)
	for i, header := range line {
		header = strings.ToLower(header)
		imp.headers[header] = append(imp.headers[header], i)
	}
}

func (imp *CsvImporter) cellValues(line []string, header string) []string {
	if headerIndexes, ok := imp.headers[header]; !ok {
		return nil
	} else {
		var values []string
		for _, headerIndex := range headerIndexes {
			if headerIndex < len(line) {
				value := strings.TrimSpace(line[headerIndex])
				if value != "" {
					values = append(values, value)
				}
			}
		}

		return values
	}
}

func (imp *CsvImporter) cellValue(line []string, header string) string {
	values := imp.cellValues(line, header)
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

func (imp *CsvImporter) stateToStatus(state string) *BacklogItemStatus {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "accepted":
		return AcceptedStatus
	case "delivered":
		return DeliveredStatus
	case "finished":
		return FinishedStatus
	case "started":
		return StartedStatus
	case "rejected":
		return RejectedStatus
	case "unstarted", "planned":
		return UnstartedStatus
	case "unscheduled":
		return UnstartedStatus
	}
	return UnstartedStatus
}

// storyType maps Pivotal's "Story Type" column to our `type:` field.
// Unknown values default to feature.
func (imp *CsvImporter) storyType(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "feature", "":
		return "feature"
	case "bug":
		return "bug"
	case "chore":
		return "chore"
	case "release":
		return "release"
	}
	return "feature"
}

func (imp *CsvImporter) getItemName(title string) string {
	itemName := utils.GetValidFileName(title)
	if startsFromCapitalLetter.MatchString(itemName) {
		itemName = strings.ToLower(itemName[0:1]) + itemName[1:]
	}

	return itemName
}

func (imp *CsvImporter) createItemIfNotExists(line []string, userList *UserList) error {
	title := imp.cellValue(line, "title")
	labels := delimiterRe.Split(imp.cellValue(line, "labels"), -1)
	itemName := imp.getItemName(title)
	itemPath := filepath.Join(imp.backlogDir, fmt.Sprintf("%s.md", itemName))
	_, err := os.Stat(itemPath)
	if err == nil {
		fmt.Printf("The item '%s' already exists. Skipping.\n", itemName)
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}

	item, err := LoadBacklogItem(itemPath)
	if err != nil {
		return err
	}

	tagSet := make(map[string]bool)
	var tags []string
	for _, label := range labels {
		label = spacesRe.ReplaceAllString(label, "-")
		if label != "" && !tagSet[strings.ToLower(label)] {
			tagSet[label] = true
			tags = append(tags, label)
		}
	}

	estimate := imp.cellValue(line, "estimate")
	status := imp.stateToStatus(imp.cellValue(line, "current state"))
	storyType := imp.storyType(imp.cellValue(line, "story type"))
	created := imp.cellValue(line, "created at")
	author := imp.cellValue(line, "requested by")
	assigned := imp.cellValue(line, "owned by")
	description := imp.cellValue(line, "description")

	// Pivotal's CSV exports use either "Mon 2, 2006" or ISO-ish forms.
	// Normalize to our timestamp.
	parseDate := func(s string) string {
		s = strings.TrimSpace(s)
		if s == "" {
			return ""
		}
		layouts := []string{"Jan 2, 2006", "2006-01-02", "2006-01-02T15:04:05Z", time.RFC3339}
		for _, layout := range layouts {
			if t, err := time.Parse(layout, s); err == nil {
				if layout == "Jan 2, 2006" || layout == "2006-01-02" {
					t = t.Add(12 * time.Hour)
				}
				return utils.GetTimestamp(t)
			}
		}
		return ""
	}
	created = parseDate(created)
	acceptedAt := parseDate(imp.cellValue(line, "accepted at"))
	finishedAt := parseDate(imp.cellValue(line, "started at"))
	deliveredAt := parseDate(imp.cellValue(line, "delivered at"))

	description += `

## Possible solution

## Comments

## Attachments
`

	err = imp.resolveUnknownUsers(line, userList)
	if err != nil {
		return err
	}

	item.SetTitle(title)
	if created != "" {
		item.SetCreated(created)
		item.SetModified(created)
	}
	item.SetAuthor(author)
	item.SetType(storyType)
	item.SetStatus(status)
	item.SetAssigned(assigned)
	if storyType != "bug" && storyType != "chore" {
		item.SetEstimate(estimate)
	}
	item.SetTags(tags)
	if finishedAt != "" {
		item.SetFinished(finishedAt)
	}
	if deliveredAt != "" {
		item.SetDelivered(deliveredAt)
	}
	if acceptedAt != "" {
		item.SetAccepted(acceptedAt)
	} else if status == AcceptedStatus && created != "" {
		// Pivotal CSV often omits accepted_at; fall back to created so
		// velocity has something to bucket against.
		item.SetAccepted(created)
	}
	item.SetDescription(description)
	return item.Save()
}

func (imp *CsvImporter) resolveUnknownUsers(line []string, userList *UserList) error {
	allAssigned := imp.cellValues(line, "owned by")
	for _, assigned := range allAssigned {
		user := userList.User(assigned)
		if user == nil {
			unresolvedUsers, err := userList.ResolveGitUsers([]string{assigned})
			if err != nil {
				return err
			}
			if len(unresolvedUsers) > 0 {
				if userList.AddUser(assigned, "") {
					fmt.Printf("You should specify email for auto-created user '%s'\n", assigned)
					err := userList.Save()
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
