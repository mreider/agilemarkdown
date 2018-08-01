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

	headers map[string]int
}

func NewCsvImporter(csvPath string, backlogDir string) *CsvImporter {
	return &CsvImporter{csvPath: csvPath, backlogDir: backlogDir}
}

func (imp *CsvImporter) Import() error {
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
			err := imp.createItemIfNotExists(line)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (imp *CsvImporter) parseHeaders(line []string) {
	imp.headers = make(map[string]int)
	for i, header := range line {
		imp.headers[strings.ToLower(header)] = i
	}
}

func (imp *CsvImporter) cellValue(line []string, header string) string {
	if headerIndex, ok := imp.headers[header]; !ok || headerIndex >= len(line) {
		return ""
	} else {
		return strings.TrimSpace(line[headerIndex])
	}
}

func (imp *CsvImporter) stateToStatus(state string) *BacklogItemStatus {
	switch strings.ToLower(state) {
	case "accepted":
		return FinishedStatus
	case "delivered", "finished", "started":
		return DoingStatus
	case "unstarted":
		return PlannedStatus
	case "unscheduled":
		return UnplannedStatus
	}
	return UnplannedStatus
}

func (imp *CsvImporter) getItemName(title string) string {
	itemName := utils.GetValidFileName(title)
	if startsFromCapitalLetter.MatchString(itemName) {
		itemName = strings.ToLower(itemName[0:1]) + itemName[1:]
	}

	return itemName
}

func (imp *CsvImporter) createItemIfNotExists(line []string) error {
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
	created := imp.cellValue(line, "created at")
	author := imp.cellValue(line, "requested by")
	assigned := imp.cellValue(line, "owned by")
	description := imp.cellValue(line, "description")

	description += `

## Possible solution

## Comments

## Attachments
`

	if createdDate, err := time.Parse("Jan 2, 2006", created); err == nil {
		createdDate = createdDate.Add(time.Hour * 12)
		created = utils.GetTimestamp(createdDate)
	}

	item.SetTitle(title)
	item.SetCreated(created)
	item.SetModified(created)
	item.SetAuthor(author)
	item.SetStatus(status)
	item.SetAssigned(assigned)
	item.SetEstimate(estimate)
	item.SetTags(tags)
	item.SetDescription(description)
	return item.Save()
}
