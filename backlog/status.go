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

var AllStatusCodes = []string{"f", "g", "h", "l"}

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
	for _, status := range AllStatuses {
		if status.Name == statusName {
			return status
		}
	}
	return nil
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
	for _, code := range AllStatusCodes {
		if code == statusCode {
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
