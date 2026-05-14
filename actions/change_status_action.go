package actions

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
)

var (
	itemStatusRe = regexp.MustCompile(`^(\d+)\s+([usfdarz])$`)
)

type ChangeStatusAction struct {
	backlogDir string
	statusCode string
}

func NewChangeStatusAction(backlogDir, statusCode string) *ChangeStatusAction {
	return &ChangeStatusAction{backlogDir: backlogDir, statusCode: statusCode}
}

func (a *ChangeStatusAction) Execute() error {
	items, err := (backlog.BacklogView{}).ShowBacklogItems(a.backlogDir, a.statusCode)
	if items == nil {
		return err
	}
	reader := bufio.NewReader(os.Stdin)
	for {
		hints := make([]string, 0, len(backlog.AllStatuses)+1)
		for _, s := range backlog.AllStatuses {
			hints = append(hints, s.Hint())
		}
		hints = append(hints, "(z)archive")
		fmt.Printf("Enter story # and status %s, or e to exit (example: 1 a accepts #1)\n", strings.Join(hints, ", "))
		text, _ := reader.ReadString('\n')
		text = strings.ToLower(strings.TrimSpace(text))
		if text == "e" {
			break
		}
		match := itemStatusRe.FindStringSubmatch(text)
		if match == nil {
			continue
		}
		itemNo, _ := strconv.Atoi(match[1])
		code := match[2]
		idx := itemNo - 1
		if idx < 0 || idx >= len(items) {
			fmt.Println("illegal story number")
			continue
		}
		item := items[idx]
		if code == "z" {
			item.SetArchived(true)
		} else {
			ApplyStatusTransition(item, backlog.StatusByCode(code))
		}
		if err := item.Save(); err != nil {
			return err
		}
	}
	return nil
}

// ApplyStatusTransition mutates item to reflect the new status and stamps
// the relevant timestamp. Pure: caller is responsible for Save().
//
// Pivotal-style story-type shortcuts:
//   - chores skip `finished` and `delivered`. Calling finish or deliver on
//     a chore advances straight to accepted.
//   - releases skip started/finished/delivered entirely. Anything other than
//     unstarted, accepted, or rejected jumps to accepted.
//
// Transitions (after the type-specific remap):
//   - status -> finished:  finished_at = now
//   - status -> delivered: delivered_at = now (finished_at filled if missing)
//   - status -> accepted:  accepted_at = now (finished_at + delivered_at filled if missing)
//   - status -> rejected:  no timestamp; the next move clears the chain
//   - status -> started:   completion timestamps cleared if going backward
//   - status -> unstarted: same; full reset
func ApplyStatusTransition(item *backlog.BacklogItem, newStatus *backlog.BacklogItemStatus) {
	if newStatus == nil {
		return
	}
	// Story-type shortcuts.
	switch item.Type() {
	case "chore":
		if newStatus == backlog.FinishedStatus || newStatus == backlog.DeliveredStatus {
			newStatus = backlog.AcceptedStatus
		}
	case "release":
		switch newStatus {
		case backlog.StartedStatus, backlog.FinishedStatus, backlog.DeliveredStatus:
			newStatus = backlog.AcceptedStatus
		}
	}
	now := utils.GetCurrentTimestamp()
	old := backlog.StatusByName(item.Status())
	item.SetStatus(newStatus)
	item.SetModified(now)
	if old == newStatus {
		return
	}
	switch newStatus {
	case backlog.StartedStatus:
		// Re-starting clears completion timestamps and starts the clock.
		// If a started: was already set (e.g. coming back from rejected),
		// keep the original so cycle-time is computed from first entry.
		if item.Started().IsZero() {
			item.SetStarted(now)
		}
		item.SetFinished("")
		item.SetDelivered("")
		item.SetAccepted("")
	case backlog.FinishedStatus:
		if item.Started().IsZero() {
			item.SetStarted(now)
		}
		item.SetFinished(now)
		item.SetDelivered("")
		item.SetAccepted("")
	case backlog.DeliveredStatus:
		if item.Started().IsZero() {
			item.SetStarted(now)
		}
		if item.Finished().IsZero() {
			item.SetFinished(now)
		}
		item.SetDelivered(now)
		item.SetAccepted("")
	case backlog.AcceptedStatus:
		if item.Started().IsZero() {
			item.SetStarted(now)
		}
		if item.Finished().IsZero() {
			item.SetFinished(now)
		}
		if item.Delivered().IsZero() {
			item.SetDelivered(now)
		}
		item.SetAccepted(now)
	case backlog.RejectedStatus:
		item.SetAccepted("")
	case backlog.UnstartedStatus:
		// Full reset back to the queue: clear started + completions.
		item.SetStarted("")
		item.SetFinished("")
		item.SetDelivered("")
		item.SetAccepted("")
	}
}
