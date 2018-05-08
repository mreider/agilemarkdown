package backlog

import (
	"fmt"
	"strings"
)

type BacklogItemStatus struct {
	Code        string
	Name        string
	Description string
}

var (
	DoingStatus     = &BacklogItemStatus{"d", "doing", "in doing"}
	PlannedStatus   = &BacklogItemStatus{"p", "planned", "planned"}
	UnplannedStatus = &BacklogItemStatus{"u", "unplanned", "unplanned"}
	FinishedStatus  = &BacklogItemStatus{"f", "finished", "finished"}
)

var AllStatuses = []*BacklogItemStatus{DoingStatus, PlannedStatus, UnplannedStatus, FinishedStatus}

func StatusByCode(statusCode string) *BacklogItemStatus {
	for _, status := range AllStatuses {
		if status.Code == statusCode {
			return status
		}
	}
	return nil
}

func StatusByName(statusName string) *BacklogItemStatus {
	statusName = strings.ToLower(statusName)
	for _, status := range AllStatuses {
		if status.Name == statusName {
			return status
		}
	}
	return nil
}

func StatusIndex(status *BacklogItemStatus) int {
	if status == nil {
		return -1
	}

	for i, st := range AllStatuses {
		if st.Name == status.Name {
			return i
		}
	}
	return -1
}

func StatusNameByCode(statusCode string) string {
	status := StatusByCode(statusCode)
	if status != nil {
		return status.Name
	}
	return "unknown"
}

func IsValidStatusCode(statusCode string) bool {
	for _, status := range AllStatuses {
		if status.Code == statusCode {
			return true
		}
	}
	return false
}

func AllStatusesList() string {
	result := make([]string, 0, len(AllStatuses))
	for _, status := range AllStatuses {
		result = append(result, fmt.Sprintf("%s (%s)", status.Code, status.Name))
	}
	return strings.Join(result, ", ")
}

func (status *BacklogItemStatus) CapitalizedName() string {
	return strings.ToUpper(status.Name[0:1]) + status.Name[1:]
}
