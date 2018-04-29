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

var AllStatuses = []*BacklogItemStatus{
	{"f", "flying", "in flight"},
	{"g", "gate", "at the gate"},
	{"h", "hangar", "in hangar"},
	{"l", "landed", "landed"},
}

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

func StatusDescriptionByCode(statusCode string) string {
	status := StatusByCode(statusCode)
	if status != nil {
		return status.Description
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
