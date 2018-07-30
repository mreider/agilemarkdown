package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"os"
	"path/filepath"
	"strings"
)

type ImportAction struct {
	backlogDir string
	csvPaths   []string
}

func NewImportAction(backlogDir string, csvPaths []string) *ImportAction {
	return &ImportAction{backlogDir: backlogDir, csvPaths: csvPaths}
}

func (a *ImportAction) Execute() error {
	for _, csvPath := range a.csvPaths {
		csvPath, err := filepath.Abs(csvPath)
		if err != nil {
			fmt.Printf("The csv file '%s' is wrong: %v\n", csvPath, err)
			continue
		}
		_, err = os.Stat(csvPath)
		if err != nil {
			fmt.Printf("The csv file '%s' is wrong: %v\n", csvPath, err)
			continue
		}
		ext := strings.ToLower(filepath.Ext(csvPath))
		if ext != ".csv" {
			fmt.Printf("The file '%s' should be a CSV file\n", csvPath)
			continue
		}

		csvImporter := backlog.NewCsvImporter(csvPath, a.backlogDir)
		err = csvImporter.Import()
		if err != nil {
			fmt.Printf("Import of the csv file '%s' failed: %v\n", csvPath, err)
			continue
		}
	}
	return nil
}
