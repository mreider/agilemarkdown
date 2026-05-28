package backlog

import (
	"time"

	"github.com/mreider/agilemarkdown/config"
)

// IsDone reports whether an item should be considered "done", i.e.
// already accepted in a prior iteration. Stories accepted within the
// current iteration are NOT done: Pivotal shows them at the top of
// Current with a green chip and lets the PM see them next to the
// in-flight work. They graduate to Done only at iteration roll-over.
func IsDone(item *BacklogItem, cfg *config.Config, now time.Time) bool {
	if item == nil || cfg == nil {
		return false
	}
	if item.Status() != AcceptedStatus.Name {
		return false
	}
	accepted := item.Accepted()
	if accepted.IsZero() {
		// Accepted but no timestamp on the story. Leave it in the
		// active view rather than hiding it: the PM will want to see
		// the missing-date case to fix it.
		return false
	}
	currentIter := IterationNumberFor(IterationStartFor(now, cfg), cfg)
	acceptedIter := IterationNumberFor(IterationStartFor(accepted, cfg), cfg)
	return acceptedIter < currentIter
}

// AcceptedInCurrentIteration reports whether an item was accepted in
// the current iteration. These stay above the line in Current with a
// green chip and no action buttons.
func AcceptedInCurrentIteration(item *BacklogItem, cfg *config.Config, now time.Time) bool {
	if item == nil || cfg == nil {
		return false
	}
	if item.Status() != AcceptedStatus.Name {
		return false
	}
	accepted := item.Accepted()
	if accepted.IsZero() {
		return false
	}
	currentIter := IterationNumberFor(IterationStartFor(now, cfg), cfg)
	acceptedIter := IterationNumberFor(IterationStartFor(accepted, cfg), cfg)
	return acceptedIter == currentIter
}
