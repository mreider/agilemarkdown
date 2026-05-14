package backlog

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	timelineDateRe = regexp.MustCompile(`^[0-9]{4}-[0-9]{1,2}-[0-9]{1,2}$`)
)

// ItemValidationError points at a specific frontmatter field on an item.
type ItemValidationError struct {
	Path    string
	Key     string
	Message string
}

func (e ItemValidationError) Error() string {
	return fmt.Sprintf("%s: %s: %s", e.Path, e.Key, e.Message)
}

// ValidateItem checks an item's frontmatter against schema/item.schema.json.
// Returns nil for valid items.
func ValidateItem(item *BacklogItem) []ItemValidationError {
	var errs []ItemValidationError

	status := strings.ToLower(strings.TrimSpace(item.Status()))
	if status == "" {
		errs = append(errs, ItemValidationError{Path: item.Path(), Key: "status", Message: "required"})
	} else if !isValidStatusName(status) {
		errs = append(errs, ItemValidationError{
			Path:    item.Path(),
			Key:     "status",
			Message: fmt.Sprintf("must be one of %s, got %q", AllStatusesList(), status),
		})
	}

	if t := item.Type(); t != "" {
		switch t {
		case "feature", "bug", "chore", "release":
		default:
			errs = append(errs, ItemValidationError{Path: item.Path(), Key: "type", Message: fmt.Sprintf("must be feature|bug|chore|release, got %q", t)})
		}
		if t == "release" && item.ReleaseDate() == "" {
			errs = append(errs, ItemValidationError{Path: item.Path(), Key: "release_date", Message: "required when type=release"})
		}
	}

	if est := strings.TrimSpace(item.Estimate()); est != "" {
		if _, err := strconv.ParseFloat(est, 64); err != nil {
			errs = append(errs, ItemValidationError{Path: item.Path(), Key: "estimate", Message: fmt.Sprintf("must be numeric, got %q", est)})
		}
	}

	if start, end := item.TimelineStr(); start != "" || end != "" {
		if !timelineDateRe.MatchString(start) {
			errs = append(errs, ItemValidationError{Path: item.Path(), Key: "timeline.start", Message: fmt.Sprintf("must be 'YYYY-MM-DD', got %q", start)})
		}
		if !timelineDateRe.MatchString(end) {
			errs = append(errs, ItemValidationError{Path: item.Path(), Key: "timeline.end", Message: fmt.Sprintf("must be 'YYYY-MM-DD', got %q", end)})
		}
	}

	return errs
}

func isValidStatusName(name string) bool {
	for _, s := range AllStatuses {
		if strings.EqualFold(s.Name, name) {
			return true
		}
	}
	return false
}
