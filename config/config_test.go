package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultsValid(t *testing.T) {
	c := Defaults()
	if err := c.Validate(); err != nil {
		t.Fatalf("defaults must validate: %v", err)
	}
	if got, want := c.Estimation.Scale, "fibonacci"; got != want {
		t.Errorf("scale = %q want %q", got, want)
	}
	if got, want := len(c.Estimation.Values), 6; got != want {
		t.Errorf("fibonacci values = %d want %d", got, want)
	}
	if got, want := c.Iteration.LengthWeeks, 1; got != want {
		t.Errorf("length_weeks = %d want %d", got, want)
	}
	if got, want := c.IterationStartWeekday(), time.Monday; got != want {
		t.Errorf("start weekday = %v want %v", got, want)
	}
	if c.IterationLocation().String() != "UTC" {
		t.Errorf("loc = %v", c.IterationLocation())
	}
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	c, err := LoadConfig(filepath.Join(dir, "nope.yaml"))
	if err != nil {
		t.Fatalf("missing file should return defaults: %v", err)
	}
	if c.Estimation.Scale != "fibonacci" {
		t.Errorf("expected fibonacci default, got %q", c.Estimation.Scale)
	}
}

func TestRoundtripYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	in := Defaults()
	in.Iteration.LengthWeeks = 2
	in.Iteration.StartDay = "wednesday"
	in.Velocity.Strategy = "manual"
	in.Velocity.InitialVelocity = 25
	in.StoryTypes.BugEstimable = true

	if err := in.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	out, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if out.Iteration.LengthWeeks != 2 {
		t.Errorf("length lost: %d", out.Iteration.LengthWeeks)
	}
	if out.IterationStartWeekday() != time.Wednesday {
		t.Errorf("start day lost: %v", out.IterationStartWeekday())
	}
	if out.Velocity.Strategy != "manual" || out.Velocity.InitialVelocity != 25 {
		t.Errorf("velocity lost: %+v", out.Velocity)
	}
	if !out.StoryTypes.BugEstimable {
		t.Errorf("bug_estimable lost")
	}
}

func TestValidateRejectsBadInputs(t *testing.T) {
	cases := []struct {
		name  string
		mut   func(*Config)
		match string
	}{
		{"bad scale", func(c *Config) { c.Estimation.Scale = "bogus" }, "estimation.scale"},
		{"custom no values", func(c *Config) { c.Estimation.Scale = "custom"; c.Estimation.Values = nil }, "estimation.values"},
		{"length 5", func(c *Config) { c.Iteration.LengthWeeks = 5 }, "length_weeks"},
		{"bad timezone", func(c *Config) { c.Iteration.Timezone = "Atlantis/Atlantis" }, "timezone"},
		{"bad start day", func(c *Config) { c.Iteration.StartDay = "funday" }, "start_day"},
		{"bad strategy", func(c *Config) { c.Velocity.Strategy = "vibes" }, "strategy"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := Defaults()
			tc.mut(c)
			err := c.Validate()
			if err == nil {
				t.Fatalf("expected error matching %q, got nil", tc.match)
			}
			if !contains(err.Error(), tc.match) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.match)
			}
		})
	}
}

func TestCustomScale(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yaml := `
estimation:
  scale: custom
  values: [0, 0.5, 1, 2, 3]
iteration:
  length_weeks: 2
velocity:
  strategy: rolling
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got, want := len(c.Estimation.Values), 5; got != want {
		t.Errorf("custom values len = %d want %d", got, want)
	}
	if !c.IsValidEstimate(0.5) {
		t.Errorf("0.5 should be valid")
	}
	if c.IsValidEstimate(13) {
		t.Errorf("13 should not be valid")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
