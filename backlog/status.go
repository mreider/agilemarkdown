package backlog

import (
	"fmt"
	"strings"

	"github.com/mreider/agilemarkdown/utils"
)

// BacklogItemStatus is one cell of the Pivotal-Tracker-derived workflow:
//
//	unstarted -> started -> finished -> delivered -> accepted
//	                                              -> rejected -> started
//
// `accepted` is the only status that contributes to velocity. `rejected`
// is a holding state; the next action sends the item back to `started`.
type BacklogItemStatus struct {
	Code        string
	Name        string
	Description string
}

var (
	UnstartedStatus = &BacklogItemStatus{"u", "unstarted", "unstarted (in backlog)"}
	StartedStatus   = &BacklogItemStatus{"s", "started", "in progress"}
	FinishedStatus  = &BacklogItemStatus{"f", "finished", "dev complete, awaiting delivery"}
	DeliveredStatus = &BacklogItemStatus{"d", "delivered", "deployed, awaiting acceptance"}
	AcceptedStatus  = &BacklogItemStatus{"a", "accepted", "accepted (counts toward velocity)"}
	RejectedStatus  = &BacklogItemStatus{"r", "rejected", "rejected; back to started"}
)

// AllStatuses is the canonical workflow order, used in views and pickers.
var AllStatuses = []*BacklogItemStatus{
	UnstartedStatus,
	StartedStatus,
	FinishedStatus,
	DeliveredStatus,
	AcceptedStatus,
	RejectedStatus,
}

func StatusByCode(code string) *BacklogItemStatus {
	for _, s := range AllStatuses {
		if s.Code == code {
			return s
		}
	}
	return nil
}

func StatusByName(name string) *BacklogItemStatus {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, s := range AllStatuses {
		if s.Name == name {
			return s
		}
	}
	return nil
}

func StatusIndex(status *BacklogItemStatus) int {
	if status == nil {
		return -1
	}
	for i, s := range AllStatuses {
		if s.Name == status.Name {
			return i
		}
	}
	return -1
}

func StatusNameByCode(code string) string {
	if s := StatusByCode(code); s != nil {
		return s.Name
	}
	return "unknown"
}

func IsValidStatusCode(code string) bool {
	return StatusByCode(code) != nil
}

func AllStatusesList() string {
	parts := make([]string, 0, len(AllStatuses))
	for _, s := range AllStatuses {
		parts = append(parts, fmt.Sprintf("%s (%s)", s.Code, s.Name))
	}
	return strings.Join(parts, ", ")
}

func (s *BacklogItemStatus) CapitalizedName() string {
	return utils.TitleFirstLetter(s.Name)
}

func (s *BacklogItemStatus) Hint() string {
	if strings.HasPrefix(s.Name, s.Code) {
		return fmt.Sprintf("(%s)%s", s.Code, strings.TrimPrefix(s.Name, s.Code))
	}
	return fmt.Sprintf("(%s)%s", s.Code, s.Name)
}

// CountsForVelocity reports whether items in this status contribute to
// velocity. Only `accepted` does.
func (s *BacklogItemStatus) CountsForVelocity() bool {
	return s == AcceptedStatus
}
