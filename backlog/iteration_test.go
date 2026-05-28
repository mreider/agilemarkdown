package backlog

import (
	"strconv"
	"testing"
	"time"

	"github.com/mreider/agilemarkdown/config"
)

func TestIterationStartMonday1Week(t *testing.T) {
	c := config.Defaults() // monday, 1 week, UTC
	cases := []struct {
		now  string
		want string
	}{
		{"2026-05-07T15:00:00Z", "2026-05-04T00:00:00Z"}, // Thu -> previous Mon
		{"2026-05-04T00:00:00Z", "2026-05-04T00:00:00Z"}, // Mon -> same Mon
		{"2026-05-10T23:59:59Z", "2026-05-04T00:00:00Z"}, // Sun -> previous Mon
		{"2026-05-11T00:00:00Z", "2026-05-11T00:00:00Z"}, // Mon -> same Mon
	}
	for _, tc := range cases {
		now, _ := time.Parse(time.RFC3339, tc.now)
		got := IterationStartFor(now, c)
		want, _ := time.Parse(time.RFC3339, tc.want)
		if !got.Equal(want) {
			t.Errorf("now=%s start=%s want=%s", tc.now, got, want)
		}
	}
}

func TestIterationStart2WeekStable(t *testing.T) {
	c := config.Defaults()
	c.Iteration.LengthWeeks = 2
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	// Two adjacent weeks must hash to the same iteration start.
	now1, _ := time.Parse(time.RFC3339, "2026-05-07T10:00:00Z")
	now2 := now1.AddDate(0, 0, 7)
	if !IterationStartFor(now1, c).Equal(IterationStartFor(now2, c)) {
		t.Errorf("two weeks within 2-week iteration must share a start: %s vs %s",
			IterationStartFor(now1, c), IterationStartFor(now2, c))
	}
	now3 := now1.AddDate(0, 0, 14)
	if IterationStartFor(now1, c).Equal(IterationStartFor(now3, c)) {
		t.Errorf("two weeks two weeks apart must NOT share a start")
	}
}

func TestVelocityManualStrategy(t *testing.T) {
	c := config.Defaults()
	c.Velocity.Strategy = "manual"
	c.Velocity.InitialVelocity = 17
	v, _, _ := ComputeVelocity(time.Now(), nil, c, nil)
	if v != 17 {
		t.Errorf("manual velocity = %v want 17", v)
	}
}

func TestVelocityRollingAverages3Iterations(t *testing.T) {
	c := config.Defaults()
	c.Velocity.Strategy = "rolling"
	c.Velocity.Lookback = 3

	now, _ := time.Parse(time.RFC3339, "2026-05-08T10:00:00Z") // Friday
	// 3 prior iterations ending the Mon before "now"
	// week of now starts 2026-05-04. previous starts 2026-04-27, 2026-04-20, 2026-04-13.
	items := []*BacklogItem{
		acceptedItem(t, "a", 5, "2026-04-15T12:00:00Z"),
		acceptedItem(t, "b", 3, "2026-04-22T12:00:00Z"),
		acceptedItem(t, "c", 2, "2026-04-29T12:00:00Z"),
		acceptedItem(t, "d", 8, "2026-04-29T18:00:00Z"),
		acceptedItem(t, "e", 1, "2026-04-12T12:00:00Z"), // outside window (older)
	}
	v, iters, boot := ComputeVelocity(now, items, c, nil)
	if boot {
		t.Errorf("expected non-bootstrap; iters=%+v", iters)
	}
	// canonical formula: SUM(points/strength) / SUM(length_weeks) * default_length, floored
	// strength=1.0 each, length=1 week each
	// numerator = 5 + 3 + 10 = 18; denominator = 3 weeks
	// per_week = 6.0; * 1 week default = 6.0; floor = 6
	if v != 6.0 {
		t.Errorf("rolling velocity = %v want 6.0; iters=%+v", v, iters)
	}
}

func TestVelocityStrictTrimsExtremes(t *testing.T) {
	c := config.Defaults()
	c.Velocity.Strategy = "strict"
	c.Velocity.Lookback = 3
	now, _ := time.Parse(time.RFC3339, "2026-05-08T10:00:00Z")
	items := []*BacklogItem{
		acceptedItem(t, "a", 1, "2026-04-15T12:00:00Z"),
		acceptedItem(t, "b", 9, "2026-04-22T12:00:00Z"),
		acceptedItem(t, "c", 4, "2026-04-29T12:00:00Z"),
	}
	v, _, _ := ComputeVelocity(now, items, c, nil)
	// drop low(1), high(9), middle iteration with 4 points / 1 week / 1.0 strength
	// = 4 per week * 1 default = 4
	if v != 4.0 {
		t.Errorf("strict velocity = %v want 4.0", v)
	}
}

func TestVelocityTeamStrengthNormalizes(t *testing.T) {
	// One iteration at 50% strength accepted 6 points; canonical formula
	// normalizes that to 12 in the numerator.
	c := config.Defaults()
	c.Velocity.Strategy = "rolling"
	c.Velocity.Lookback = 3
	now, _ := time.Parse(time.RFC3339, "2026-05-08T10:00:00Z")
	items := []*BacklogItem{
		acceptedItem(t, "a", 6, "2026-04-15T12:00:00Z"), // half-strength
		acceptedItem(t, "b", 6, "2026-04-22T12:00:00Z"), // full
		acceptedItem(t, "c", 6, "2026-04-29T12:00:00Z"), // full
	}
	overrides := &IterationOverrides{}
	overrides.Set(iterationNumberFor(IterationStartFor(now.AddDate(0, 0, -21), c), c), 0.5, 0)

	v, _, _ := ComputeVelocity(now, items, c, overrides)
	// numerator = 6/0.5 + 6/1 + 6/1 = 24; denominator = 3 weeks; per_week = 8
	if v != 8.0 {
		t.Errorf("strength-normalized velocity = %v want 8.0", v)
	}
}

func TestVelocityTeamStrengthZeroExcludes(t *testing.T) {
	c := config.Defaults()
	c.Velocity.Strategy = "rolling"
	c.Velocity.Lookback = 3
	now, _ := time.Parse(time.RFC3339, "2026-05-08T10:00:00Z")
	items := []*BacklogItem{
		acceptedItem(t, "a", 100, "2026-04-15T12:00:00Z"), // excluded
		acceptedItem(t, "b", 4, "2026-04-22T12:00:00Z"),
		acceptedItem(t, "c", 4, "2026-04-29T12:00:00Z"),
	}
	overrides := &IterationOverrides{}
	overrides.Set(iterationNumberFor(IterationStartFor(now.AddDate(0, 0, -21), c), c), 0, 0)

	v, _, _ := ComputeVelocity(now, items, c, overrides)
	// only iter2 + iter3: numerator = 4 + 4 = 8; denominator = 2 weeks; per_week = 4
	if v != 4.0 {
		t.Errorf("strength=0 should exclude iteration; got %v want 4.0", v)
	}
}

func TestVelocityIgnoresBugAndChoreByDefault(t *testing.T) {
	c := config.Defaults()
	now, _ := time.Parse(time.RFC3339, "2026-05-08T10:00:00Z")
	bug := acceptedItem(t, "x", 5, "2026-04-29T12:00:00Z")
	bug.SetType("bug")
	chore := acceptedItem(t, "y", 5, "2026-04-29T12:00:00Z")
	chore.SetType("chore")
	feat := acceptedItem(t, "z", 3, "2026-04-29T12:00:00Z")

	if CountsForVelocity(bug, c) {
		t.Errorf("bug should not count by default")
	}
	if CountsForVelocity(chore, c) {
		t.Errorf("chore should not count by default")
	}
	if !CountsForVelocity(feat, c) {
		t.Errorf("feature should count")
	}

	c.StoryTypes.BugEstimable = true
	if !CountsForVelocity(bug, c) {
		t.Errorf("bug should count when bug_estimable=true")
	}

	_ = now
}

func acceptedItem(t *testing.T, name string, points int, acceptedAt string) *BacklogItem {
	t.Helper()
	item := NewBacklogItem(name, "")
	item.SetTitle(name)
	item.SetStatus(AcceptedStatus)
	item.SetType("feature")
	item.SetEstimate(strconv.Itoa(points))
	item.SetAccepted(acceptedAt)
	return item
}
