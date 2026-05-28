package actions

import (
	"fmt"

	"github.com/mreider/agilemarkdown/backlog"
)

// SyncPriorityStep enforces the priority/icebox invariant per backlog:
// every active item appears exactly once across `_priority.md` and
// `_icebox.md`. Orphan order entries (pointing at deleted items) are
// trimmed. Items in both are collapsed to priority (priority wins).
// Items in neither are appended to icebox bottom. Run as part of `am
// sync` after items are loaded.
type SyncPriorityStep struct {
	root *backlog.BacklogsStructure
}

func NewSyncPriorityStep(root *backlog.BacklogsStructure) *SyncPriorityStep {
	return &SyncPriorityStep{root: root}
}

func (s *SyncPriorityStep) Execute() error {
	dirs, err := s.root.BacklogDirs()
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		bck, err := backlog.LoadBacklog(dir)
		if err != nil {
			return err
		}
		pri, ice, changed, err := backlog.EnforceOrderInvariant(bck, dir)
		if err != nil {
			return err
		}
		if !changed {
			continue
		}
		if err := pri.Save(); err != nil {
			return fmt.Errorf("save _priority.md: %w", err)
		}
		if err := ice.Save(); err != nil {
			return fmt.Errorf("save _icebox.md: %w", err)
		}
	}
	return nil
}
