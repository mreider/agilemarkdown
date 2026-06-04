package backlog

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// The activity log is an append-only, semantic audit trail of every
// state-changing backlog operation: who did what to which story, when.
// It is distinct from git history (`am history`) in three ways:
//
//  1. It records the *intent* (the verb: started, delivered, created,
//     bug, set-type, estimate, ...) rather than a file diff.
//  2. It captures changes the moment they happen, before any commit,
//     so an agent working autonomously leaves a trail even between
//     syncs.
//  3. It is a single chronological feed across all stories and
//     backlogs, which is what you want when answering "what has this
//     agent been doing for the last four hours?".
//
// Storage is a tab-separated file under `.am/` so it stays diffable and
// greppable. Fields never contain raw tabs or newlines (we sanitize on
// write), so the format round-trips losslessly.
const activityFileName = ".am/activity.log"

// ActivityEntry is one line of the audit trail.
type ActivityEntry struct {
	Time   time.Time
	Actor  string // git user.name, or "unknown"
	Verb   string // started, finished, delivered, accepted, rejected, created, bug, chore, set-type, estimate, ...
	Item   string // path relative to the repo root, or "" for repo-wide events
	Detail string // free-form context (estimate value, rejection reason, target column, ...)
}

// ActivityFile returns the path to the audit-trail log under .am/.
func (s *BacklogsStructure) ActivityFile() string {
	return filepath.Join(s.root, activityFileName)
}

// sanitizeField strips the field separators so a value can never break
// the line format. Newlines and tabs collapse to a single space.
func sanitizeField(s string) string {
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

// AppendActivity adds one line to the audit trail at <root>/.am/activity.log,
// creating the .am directory if needed. The timestamp is stamped here in
// UTC. Audit logging is best-effort context for humans and agents; callers
// should log-and-ignore the error rather than fail the underlying command
// because the trail could not be written.
func AppendActivity(root, actor, verb, item, detail string) error {
	if root == "" {
		return nil
	}
	path := filepath.Join(root, activityFileName)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if actor == "" {
		actor = "unknown"
	}
	line := strings.Join([]string{
		time.Now().UTC().Format(time.RFC3339),
		sanitizeField(actor),
		sanitizeField(verb),
		sanitizeField(item),
		sanitizeField(detail),
	}, "\t") + "\n"

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line)
	return err
}

// LoadActivity reads the audit trail in chronological (file) order.
// A missing log is not an error: it returns an empty slice. Malformed
// lines are skipped rather than aborting the read, so a hand-edit can
// never make `am log` unusable.
func LoadActivity(root string) ([]ActivityEntry, error) {
	path := filepath.Join(root, activityFileName)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []ActivityEntry
	scanner := bufio.NewScanner(f)
	// Allow long lines (rejection reasons, long titles).
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 5)
		if len(parts) < 3 {
			continue
		}
		ts, err := time.Parse(time.RFC3339, parts[0])
		if err != nil {
			continue
		}
		e := ActivityEntry{Time: ts, Actor: parts[1], Verb: parts[2]}
		if len(parts) > 3 {
			e.Item = parts[3]
		}
		if len(parts) > 4 {
			e.Detail = parts[4]
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return entries, err
	}
	return entries, nil
}
