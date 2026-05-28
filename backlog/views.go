package backlog

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mreider/agilemarkdown/config"
)

// PriorityASCII renders _priority.md split into iteration bands. The
// rolling velocity is used to size each band: walking from the top of
// priority, items accumulate until points cross velocity, at which point
// a divider is drawn. After `maxBands` bands, the remaining items are
// shown as a single "Backlog" pile. If priority is empty or velocity is
// zero, falls back to a single flat list. When hideAccepted is true,
// items already in the accepted state are omitted from the rendering
// (Pivotal's "Hide accepted stories" behavior).
func PriorityASCII(bck *Backlog, backlogDir string, cfg *config.Config, now time.Time, maxBands int, hideAccepted bool) (string, error) {
	if maxBands <= 0 {
		maxBands = 2
	}
	pri, err := LoadPriority(backlogDir)
	if err != nil {
		return "", err
	}
	items := bck.ActiveItems()
	byPath := make(map[string]*BacklogItem, len(items))
	for _, it := range items {
		byPath[filepath.Base(it.Path())] = it
	}

	var accepted []*BacklogItem
	for _, it := range items {
		if CountsForVelocity(it, cfg) {
			accepted = append(accepted, it)
		}
	}
	overrides, _ := LoadIterationOverrides(filepath.Dir(backlogDir))
	velocity, _, _ := ComputeVelocity(now, accepted, cfg, overrides)
	if velocity <= 0 {
		velocity = 1
	}

	iterStart := IterationStartFor(now, cfg)
	weeks := cfg.Iteration.LengthWeeks
	iterNumber := iterationNumberFor(iterStart, cfg)

	var b strings.Builder
	fmt.Fprintf(&b, "Priority (%s)   velocity %.0f / iteration\n\n", filepath.Base(backlogDir), velocity)

	if pri.IsEmpty() {
		b.WriteString("  (priority is empty; everything is in icebox)\n")
		return b.String(), nil
	}

	// Walk priority once and assign each row to a band. Stories that
	// are already "done" (accepted in a prior iteration) are skipped:
	// they belong in the Done view, not in Current/Backlog. Accepted
	// stories from the *current* iteration land in band 0 above the
	// active work so the PM can see them next to in-flight stories.
	type row struct {
		idx    int
		entry  OrderEntry
		item   *BacklogItem
		points float64
		band   int // 0..maxBands-1 for iteration bands; maxBands for untimed
	}
	rows := make([]row, 0, pri.Len())
	for i, e := range pri.Entries() {
		item := byPath[e.Path]
		if item != nil && IsDone(item, cfg, now) {
			continue
		}
		if hideAccepted && item != nil && strings.EqualFold(item.Status(), AcceptedStatus.Name) {
			continue
		}
		points := 0.0
		if item != nil {
			points = parsePoints(item.Estimate())
		}
		rows = append(rows, row{idx: i, entry: e, item: item, points: points})
	}

	// Assign bands. Accepted-in-current-iter, started, finished, and
	// delivered items belong in band 0 regardless of capacity: they
	// are happening now. Unstarted items fill bands by velocity.
	band := 0
	bandPoints := 0.0
	for i := range rows {
		r := &rows[i]
		isCurrentWork := r.item != nil && (AcceptedInCurrentIteration(r.item, cfg, now) ||
			r.item.Status() == StartedStatus.Name ||
			r.item.Status() == FinishedStatus.Name ||
			r.item.Status() == DeliveredStatus.Name)
		if isCurrentWork {
			r.band = 0
			if band == 0 {
				bandPoints += r.points
			}
			continue
		}
		if band < maxBands && bandPoints > 0 && bandPoints+r.points > velocity {
			band++
			bandPoints = 0
		}
		if band >= maxBands {
			r.band = maxBands
			continue
		}
		r.band = band
		bandPoints += r.points
	}

	// Per-band stats: capacity, done points (accepted+delivered), total points.
	type bandStat struct {
		donePts  float64
		totalPts float64
	}
	stats := make([]bandStat, maxBands+1)
	for _, r := range rows {
		if r.band >= len(stats) {
			continue
		}
		stats[r.band].totalPts += r.points
		if r.item != nil {
			s := r.item.Status()
			if s == AcceptedStatus.Name || s == DeliveredStatus.Name || s == FinishedStatus.Name {
				stats[r.band].donePts += r.points
			}
		}
	}

	// Render.
	currentBand := -1
	bandStart := iterStart
	writeDivider := func(n int, start time.Time, capPts, donePts float64) {
		fmt.Fprintf(&b, "── Iteration %d  %s  %.0f / %.0f pts ──\n", n, start.Format("Mon Jan 02"), donePts, capPts)
	}
	for _, r := range rows {
		if r.band != currentBand {
			if currentBand >= 0 {
				fmt.Fprintln(&b)
			}
			if r.band >= maxBands {
				b.WriteString("── Backlog (untimed) ──\n")
			} else {
				bandStart = iterStart.AddDate(0, 0, 7*weeks*r.band)
				writeDivider(iterNumber+r.band, bandStart, velocity, stats[r.band].donePts)
			}
			currentBand = r.band
		}
		writeOrderRow(&b, r.entry, r.item, r.idx)
		if r.item != nil && r.item.Type() == "release" {
			if rd := releaseDateFor(r.item); !rd.IsZero() {
				if !bandStart.Before(rd) {
					fmt.Fprintf(&b, "       LATE: target %s, this iteration starts %s\n",
						rd.Format("2006-01-02"), bandStart.Format("2006-01-02"))
				} else {
					fmt.Fprintf(&b, "       on track: target %s\n", rd.Format("2006-01-02"))
				}
			}
		}
	}

	if iceboxCount, _ := iceboxCountForBacklog(backlogDir); iceboxCount > 0 {
		fmt.Fprintf(&b, "\n── Icebox (%d items) ── see `am show icebox`\n", iceboxCount)
	}

	return b.String(), nil
}

// DoneASCII renders accepted stories from prior iterations, newest
// first. Stories accepted in the current iteration appear in Current
// (via PriorityASCII), not here.
func DoneASCII(bck *Backlog, backlogDir string, cfg *config.Config, now time.Time) (string, error) {
	items := bck.ActiveItems()
	type doneRow struct {
		item *BacklogItem
		when time.Time
	}
	var rows []doneRow
	for _, it := range items {
		if IsDone(it, cfg, now) {
			rows = append(rows, doneRow{item: it, when: it.Accepted()})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].when.After(rows[j].when) })

	var b strings.Builder
	fmt.Fprintf(&b, "Done (%s)   %d stories\n\n", filepath.Base(backlogDir), len(rows))
	if len(rows) == 0 {
		b.WriteString("  (no accepted stories from prior iterations yet)\n")
		return b.String(), nil
	}

	currentIter := -1
	for i, r := range rows {
		iter := IterationNumberFor(IterationStartFor(r.when, cfg), cfg)
		if iter != currentIter {
			if currentIter >= 0 {
				fmt.Fprintln(&b)
			}
			start := IterationStartFor(r.when, cfg)
			fmt.Fprintf(&b, "── Iteration %d  %s ──\n", iter, start.Format("Mon Jan 02"))
			currentIter = iter
		}
		entry := OrderEntry{Title: r.item.Title(), Path: filepath.Base(r.item.Path())}
		writeOrderRow(&b, entry, r.item, i)
	}
	return b.String(), nil
}

// IceboxASCII renders the icebox as a flat stack-rank list.
func IceboxASCII(bck *Backlog, backlogDir string) (string, error) {
	ice, err := LoadIcebox(backlogDir)
	if err != nil {
		return "", err
	}
	items := bck.ActiveItems()
	byPath := make(map[string]*BacklogItem, len(items))
	for _, it := range items {
		byPath[filepath.Base(it.Path())] = it
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Icebox (%s)   %d items\n\n", filepath.Base(backlogDir), ice.Len())
	if ice.IsEmpty() {
		b.WriteString("  (empty)\n")
		return b.String(), nil
	}
	for i, e := range ice.Entries() {
		writeOrderRow(&b, e, byPath[e.Path], i)
	}
	return b.String(), nil
}

// IterationASCII renders a single iteration window (0=current, 1=next).
func IterationASCII(bck *Backlog, backlogDir string, cfg *config.Config, now time.Time, offset int) (string, error) {
	if offset < 0 {
		offset = 0
	}
	pri, err := LoadPriority(backlogDir)
	if err != nil {
		return "", err
	}
	items := bck.ActiveItems()
	byPath := make(map[string]*BacklogItem, len(items))
	for _, it := range items {
		byPath[filepath.Base(it.Path())] = it
	}
	var accepted []*BacklogItem
	for _, it := range items {
		if CountsForVelocity(it, cfg) {
			accepted = append(accepted, it)
		}
	}
	overrides, _ := LoadIterationOverrides(filepath.Dir(backlogDir))
	velocity, _, _ := ComputeVelocity(now, accepted, cfg, overrides)
	if velocity <= 0 {
		velocity = 1
	}
	iterStart := IterationStartFor(now, cfg).AddDate(0, 0, 7*cfg.Iteration.LengthWeeks*offset)
	iterNumber := iterationNumberFor(IterationStartFor(now, cfg), cfg) + offset

	band := 0
	bandPoints := 0.0
	var rows []OrderEntry
	for _, e := range pri.Entries() {
		item := byPath[e.Path]
		pts := 0.0
		if item != nil {
			pts = parsePoints(item.Estimate())
		}
		if bandPoints > 0 && bandPoints+pts > velocity {
			band++
			bandPoints = 0
			if band > offset {
				break
			}
		}
		bandPoints += pts
		if band == offset {
			rows = append(rows, e)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Iteration %d  %s  cap %.0f pts  (%d items)\n\n", iterNumber, iterStart.Format("Mon Jan 02"), velocity, len(rows))
	if len(rows) == 0 {
		b.WriteString("  (nothing planned for this window)\n")
		return b.String(), nil
	}
	for i, e := range rows {
		writeOrderRow(&b, e, byPath[e.Path], i)
	}
	return b.String(), nil
}

// EpicASCII renders an ASCII burnup for items whose `epic` frontmatter
// field equals the given slug. Walks every backlog under `root`.
func EpicASCII(rootDir, slug string) (string, error) {
	rs := NewBacklogsStructure(rootDir)
	dirs, err := rs.BacklogDirs()
	if err != nil {
		return "", err
	}
	type epicItem struct {
		item    *BacklogItem
		path    string
		points  float64
		accepted bool
	}
	var rows []epicItem
	for _, d := range dirs {
		bck, err := LoadBacklog(d)
		if err != nil {
			return "", err
		}
		for _, it := range bck.ActiveItems() {
			if !strings.EqualFold(it.Epic(), slug) {
				continue
			}
			rel, _ := filepath.Rel(rootDir, it.Path())
			rows = append(rows, epicItem{
				item:     it,
				path:     rel,
				points:   parsePoints(it.Estimate()),
				accepted: strings.EqualFold(it.Status(), AcceptedStatus.Name),
			})
		}
	}

	var b strings.Builder
	if len(rows) == 0 {
		fmt.Fprintf(&b, "Epic %q: no stories carry this epic slug.\n", slug)
		return b.String(), nil
	}

	totalPts, accPts := 0.0, 0.0
	totalCount, accCount := 0, 0
	for _, r := range rows {
		totalPts += r.points
		totalCount++
		if r.accepted {
			accPts += r.points
			accCount++
		}
	}
	pct := 0.0
	if totalPts > 0 {
		pct = accPts / totalPts * 100
	}
	const barW = 24
	filled := 0
	if totalPts > 0 {
		filled = int(float64(barW) * accPts / totalPts)
	}
	if filled < 0 {
		filled = 0
	}
	if filled > barW {
		filled = barW
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)

	fmt.Fprintf(&b, "Epic: %s   %d/%d stories  %.0f/%.0f pts  %.0f%%\n", slug, accCount, totalCount, accPts, totalPts, pct)
	fmt.Fprintf(&b, "[%s]\n\n", bar)

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].accepted != rows[j].accepted {
			return rows[i].accepted // accepted first
		}
		return rows[i].item.Title() < rows[j].item.Title()
	})
	b.WriteString("Accepted:\n")
	any := false
	for _, r := range rows {
		if !r.accepted {
			continue
		}
		fmt.Fprintf(&b, "  %s  %s  (%s)\n", typeMark(r.item.Type()), r.item.Title(), r.path)
		any = true
	}
	if !any {
		b.WriteString("  (none yet)\n")
	}
	b.WriteString("\nOpen:\n")
	any = false
	for _, r := range rows {
		if r.accepted {
			continue
		}
		fmt.Fprintf(&b, "  %s  %s  [%s]  (%s)\n", typeMark(r.item.Type()), r.item.Title(), r.item.Status(), r.path)
		any = true
	}
	if !any {
		b.WriteString("  (none open)\n")
	}
	return b.String(), nil
}

func writeOrderRow(b *strings.Builder, e OrderEntry, item *BacklogItem, idx int) {
	mark := "●"
	status := "?"
	pts := ""
	if item != nil {
		mark = typeMark(item.Type())
		status = item.Status()
		if e := strings.TrimSpace(item.Estimate()); e != "" {
			pts = e + "p"
		}
	}
	title := e.Title
	if item != nil && item.Title() != "" {
		title = item.Title()
	}
	fmt.Fprintf(b, "  %2d. %s  %-40s  %-10s  %s\n", idx+1, mark, truncate(title, 40), status, pts)
}

func typeMark(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "feature", "":
		return "★"
	case "bug":
		return "●"
	case "chore":
		return "⚙"
	case "release":
		return "▶"
	default:
		return "·"
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n < 1 {
		return ""
	}
	return s[:n-1] + "…"
}

func parsePoints(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

// releaseDateFor parses an item's release_date frontmatter as a calendar
// date. Returns zero Time if missing or unparsable.
func releaseDateFor(item *BacklogItem) time.Time {
	rd := item.ReleaseDate()
	if rd == "" {
		return time.Time{}
	}
	t, err := time.Parse("2006-01-02", rd)
	if err != nil {
		return time.Time{}
	}
	return t
}

// IterationNumberFor returns a 1-based iteration number derived from
// weeks-since-epoch / length_weeks. Stable across config reloads because
// it shares the same epoch as IterationStartFor.
func IterationNumberFor(start time.Time, c *config.Config) int {
	loc := c.IterationLocation()
	epoch := startOfWeekDay(time.Date(2000, time.January, 3, 0, 0, 0, 0, loc), c.IterationStartWeekday(), loc)
	weeks := c.Iteration.LengthWeeks
	if weeks < 1 {
		weeks = 1
	}
	d := int(start.Sub(epoch).Hours()/24) / 7
	return d/weeks + 1
}

// iterationNumberFor (unexported alias kept so existing call sites in
// this package don't need to change).
func iterationNumberFor(start time.Time, c *config.Config) int {
	return IterationNumberFor(start, c)
}

// ItemIteration returns the iteration number and a human-readable
// label for an item. The number is 1-based and shares the same epoch
// as the velocity / charting code, so it's stable across the rest of
// the surface. Anchor preference:
//   - accepted timestamp wins (definitive)
//   - then delivered / finished (in-flight history)
//   - then started (currently in flight)
// When no timestamp is set the item has no iteration: returns
// (0, "icebox" or "backlog") depending on whether priorityPos >= 0.
//
// priorityPos: 1-based position in _priority.md (or 0 if not in
// priority). When set, an unstarted item is bucketed into the
// current-or-later iteration using rolling velocity, mirroring the
// board's iteration-band rendering.
func ItemIteration(item *BacklogItem, c *config.Config, priorityPos int, velocity float64) (int, string) {
	if c == nil {
		return 0, ""
	}
	if t := item.Accepted(); !t.IsZero() {
		return IterationNumberFor(t, c), ""
	}
	if t := item.Delivered(); !t.IsZero() {
		return IterationNumberFor(t, c), "in flight"
	}
	if t := item.Finished(); !t.IsZero() {
		return IterationNumberFor(t, c), "in flight"
	}
	if t := item.Started(); !t.IsZero() {
		return IterationNumberFor(t, c), "in flight"
	}
	// Not started yet. Bucket by priority position if we have one.
	if priorityPos > 0 && velocity > 0 {
		current := IterationNumberFor(time.Now().In(c.IterationLocation()), c)
		band := int((float64(priorityPos-1)) / velocity)
		return current + band, ""
	}
	if priorityPos > 0 {
		return 0, "backlog"
	}
	return 0, "icebox"
}

// iceboxCountForBacklog returns the number of items in _icebox.md,
// ignoring errors.
func iceboxCountForBacklog(backlogDir string) (int, error) {
	ice, err := LoadIcebox(backlogDir)
	if err != nil {
		return 0, err
	}
	return ice.Len(), nil
}
