package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
)

type SyncValidateStep struct {
	root *backlog.BacklogsStructure
}

func NewSyncValidateStep(root *backlog.BacklogsStructure) *SyncValidateStep {
	return &SyncValidateStep{root: root}
}

func (s *SyncValidateStep) Execute() error {
	backlogDirs, err := s.root.BacklogDirs()
	if err != nil {
		return err
	}

	var allErrs []backlog.ItemValidationError
	for _, dir := range backlogDirs {
		bck, err := backlog.LoadBacklog(dir)
		if err != nil {
			return err
		}
		for _, item := range bck.AllItems() {
			allErrs = append(allErrs, backlog.ValidateItem(item)...)
		}
	}

	if len(allErrs) == 0 {
		return nil
	}

	fmt.Printf("Validation failed for %d field(s):\n", len(allErrs))
	for _, e := range allErrs {
		fmt.Printf("  %s\n", e.Error())
	}
	return fmt.Errorf("schema validation failed; fix items above and re-run sync")
}
