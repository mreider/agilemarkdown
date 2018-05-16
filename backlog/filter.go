package backlog

import (
	"strings"
)

type BacklogItemsFilter interface {
	Match(item *BacklogItem) bool
}

type BacklogItemsOrFilter struct {
	filters []BacklogItemsFilter
}

type BacklogItemsAndFilter struct {
	filters []BacklogItemsFilter
}

type BacklogItemsTrueFilter struct {
}

type BacklogItemsStatusCodeFilter struct {
	statusName string
}

type BacklogItemsAssignedFilter struct {
	user string
}

type BacklogItemsTagsFilter struct {
	filter BacklogItemsFilter
}

type BacklogItemsActiveFilter struct {
}

type BacklogItemsArchivedFilter struct {
}

type tagFilter struct {
	tag string
}

func (f *BacklogItemsOrFilter) Match(item *BacklogItem) bool {
	for _, filter := range f.filters {
		if filter.Match(item) {
			return true
		}
	}
	return false
}

func (f *BacklogItemsOrFilter) Or(filter BacklogItemsFilter) *BacklogItemsOrFilter {
	f.filters = append(f.filters, filter)
	return f
}

func (f *BacklogItemsAndFilter) Match(item *BacklogItem) bool {
	for _, filter := range f.filters {
		if !filter.Match(item) {
			return false
		}
	}
	return true
}

func (f *BacklogItemsAndFilter) And(filter BacklogItemsFilter) *BacklogItemsAndFilter {
	f.filters = append(f.filters, filter)
	return f
}

func (f *BacklogItemsTrueFilter) Match(item *BacklogItem) bool {
	return true
}

func NewBacklogItemsStatusCodeFilter(statusCode string) *BacklogItemsStatusCodeFilter {
	return &BacklogItemsStatusCodeFilter{statusName: strings.ToLower(StatusNameByCode(statusCode))}
}

func NewBacklogItemsAssignedFilter(user string) *BacklogItemsAssignedFilter {
	return &BacklogItemsAssignedFilter{user: strings.ToLower(user)}
}

func (f *BacklogItemsStatusCodeFilter) Match(item *BacklogItem) bool {
	itemStatusName := strings.ToLower(item.Status())
	return itemStatusName == f.statusName
}

func (f *BacklogItemsAssignedFilter) Match(item *BacklogItem) bool {
	if f.user == "" {
		return true
	}

	itemAssigned := strings.ToLower(item.Assigned())
	return itemAssigned == f.user
}

func NewBacklogItemsTagsFilter(filter string) *BacklogItemsTagsFilter {
	f := &BacklogItemsTagsFilter{}
	f.parseFilter(strings.TrimSpace(filter))
	return f
}

func (f *BacklogItemsTagsFilter) parseFilter(filter string) {
	if filter == "" {
		f.filter = &BacklogItemsTrueFilter{}
		return
	}

	orParts := strings.Fields(filter)
	orFilter := &BacklogItemsOrFilter{}
	for _, orPart := range orParts {
		orFilter.Or(&tagFilter{tag: orPart})
	}
	f.filter = orFilter
}

func (f *BacklogItemsTagsFilter) Match(item *BacklogItem) bool {
	return f.filter.Match(item)
}

func (f *tagFilter) Match(item *BacklogItem) bool {
	tag := strings.ToLower(f.tag)
	for _, itemTag := range item.Tags() {
		itemTag = strings.ToLower(itemTag)
		if itemTag == tag {
			return true
		}
	}
	return false
}

func (f *BacklogItemsActiveFilter) Match(item *BacklogItem) bool {
	return !item.Archived()
}

func (f *BacklogItemsArchivedFilter) Match(item *BacklogItem) bool {
	return item.Archived()
}
