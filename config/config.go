package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the project-wide settings file (`.am/config.yaml`).
//
// Pivotal Tracker semantics: estimation scale, iteration window math,
// velocity strategy, story-type estimability.
type Config struct {
	Estimation Estimation `yaml:"estimation"`
	Iteration  Iteration  `yaml:"iteration"`
	Velocity   Velocity   `yaml:"velocity"`
	StoryTypes StoryTypes `yaml:"story_types"`
}

type Estimation struct {
	// Scale: linear, fibonacci, powers, or custom.
	Scale string `yaml:"scale"`

	// Values is the explicit ordered list of allowed estimates.
	// When Scale != custom this is regenerated from the scale on Load.
	Values []float64 `yaml:"values,omitempty"`
}

type Iteration struct {
	// LengthWeeks: 1, 2, 3, or 4. Default 1.
	LengthWeeks int `yaml:"length_weeks"`

	// StartDay: monday..sunday. Default monday.
	StartDay string `yaml:"start_day"`

	// Timezone: IANA name (e.g. "UTC", "America/New_York"). Default UTC.
	Timezone string `yaml:"timezone"`
}

type Velocity struct {
	// Strategy: rolling, strict, or manual.
	//   rolling: average of last Lookback completed iterations (Pivotal default)
	//   strict:  drop high+low outliers, then average (non-Pivotal extension)
	//   manual:  InitialVelocity constant
	Strategy string `yaml:"strategy"`

	// Lookback iterations to average. Default 3.
	Lookback int `yaml:"lookback"`

	// InitialVelocity is the bootstrap value used before any iteration has
	// completed (and re-used if the project goes Lookback iterations with no
	// accepted points). Mirrors Pivotal Tracker's "Initial Velocity" project
	// setting. Default 10.
	InitialVelocity int `yaml:"initial_velocity"`

	// Manual is a back-compat alias for InitialVelocity. If both are set,
	// InitialVelocity wins. Older repos that wrote `velocity.manual` keep
	// loading; on next Save the file is rewritten with `initial_velocity`.
	Manual int `yaml:"manual,omitempty"`
}

type StoryTypes struct {
	// BugEstimable: when true, bug-type stories carry estimates and feed velocity.
	// Pivotal default: false.
	BugEstimable bool `yaml:"bug_estimable"`

	// ChoreEstimable: same idea for chores. Pivotal default: false.
	ChoreEstimable bool `yaml:"chore_estimable"`
}

// Defaults returns the standard agilemarkdown configuration: Pivotal-style
// fibonacci 0-8, 1-week iterations starting Monday UTC, rolling-3 velocity,
// bugs and chores not estimable.
func Defaults() *Config {
	return &Config{
		Estimation: Estimation{
			Scale:  "fibonacci",
			Values: scaleValues("fibonacci"),
		},
		Iteration: Iteration{
			LengthWeeks: 1,
			StartDay:    "monday",
			Timezone:    "UTC",
		},
		Velocity: Velocity{
			Strategy:        "rolling",
			Lookback:        3,
			InitialVelocity: 10,
		},
		StoryTypes: StoryTypes{
			BugEstimable:   false,
			ChoreEstimable: false,
		},
	}
}

// LoadConfig reads `path`. Missing file returns Defaults() with no error.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Defaults(), nil
		}
		return nil, err
	}
	cfg := Defaults()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("config %s: %w", path, err)
	}
	cfg.normalize()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save writes the config as YAML to path. Creates parent dirs as needed.
func (c *Config) Save(path string) error {
	if err := os.MkdirAll(parentDir(path), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func parentDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == os.PathSeparator || path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}

func (c *Config) normalize() {
	c.Estimation.Scale = strings.ToLower(strings.TrimSpace(c.Estimation.Scale))
	if c.Estimation.Scale == "" {
		c.Estimation.Scale = "fibonacci"
	}
	if c.Estimation.Scale != "custom" {
		c.Estimation.Values = scaleValues(c.Estimation.Scale)
	}
	if c.Iteration.LengthWeeks <= 0 {
		c.Iteration.LengthWeeks = 1
	}
	c.Iteration.StartDay = strings.ToLower(strings.TrimSpace(c.Iteration.StartDay))
	if c.Iteration.StartDay == "" {
		c.Iteration.StartDay = "monday"
	}
	if strings.TrimSpace(c.Iteration.Timezone) == "" {
		c.Iteration.Timezone = "UTC"
	}
	c.Velocity.Strategy = strings.ToLower(strings.TrimSpace(c.Velocity.Strategy))
	if c.Velocity.Strategy == "" {
		c.Velocity.Strategy = "rolling"
	}
	if c.Velocity.Lookback <= 0 {
		c.Velocity.Lookback = 3
	}
	// initial_velocity is canonical; manual is the legacy alias.
	if c.Velocity.InitialVelocity <= 0 && c.Velocity.Manual > 0 {
		c.Velocity.InitialVelocity = c.Velocity.Manual
	}
	if c.Velocity.InitialVelocity <= 0 {
		c.Velocity.InitialVelocity = 10
	}
	c.Velocity.Manual = 0
}

// Validate checks for impossible values that normalize() can't fix.
func (c *Config) Validate() error {
	switch c.Estimation.Scale {
	case "linear", "fibonacci", "powers", "custom":
	default:
		return fmt.Errorf("estimation.scale must be linear|fibonacci|powers|custom, got %q", c.Estimation.Scale)
	}
	if c.Estimation.Scale == "custom" && len(c.Estimation.Values) == 0 {
		return fmt.Errorf("estimation.values required when scale is custom")
	}
	switch c.Iteration.LengthWeeks {
	case 1, 2, 3, 4:
	default:
		return fmt.Errorf("iteration.length_weeks must be 1, 2, 3, or 4")
	}
	if _, err := time.LoadLocation(c.Iteration.Timezone); err != nil {
		return fmt.Errorf("iteration.timezone %q: %w", c.Iteration.Timezone, err)
	}
	if _, ok := startDayMap[c.Iteration.StartDay]; !ok {
		return fmt.Errorf("iteration.start_day must be monday..sunday")
	}
	switch c.Velocity.Strategy {
	case "rolling", "strict", "manual":
	default:
		return fmt.Errorf("velocity.strategy must be rolling|strict|manual")
	}
	return nil
}

// IterationLocation returns the resolved IANA Location for iteration math.
func (c *Config) IterationLocation() *time.Location {
	loc, err := time.LoadLocation(c.Iteration.Timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

// IterationStartWeekday converts the configured start day to time.Weekday.
func (c *Config) IterationStartWeekday() time.Weekday {
	if w, ok := startDayMap[c.Iteration.StartDay]; ok {
		return w
	}
	return time.Monday
}

// IsValidEstimate reports whether the given numeric points value matches the
// configured estimation scale.
func (c *Config) IsValidEstimate(points float64) bool {
	for _, v := range c.Estimation.Values {
		if v == points {
			return true
		}
	}
	return false
}

var startDayMap = map[string]time.Weekday{
	"sunday":    time.Sunday,
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
}

func scaleValues(scale string) []float64 {
	switch scale {
	case "linear":
		return []float64{0, 1, 2, 3}
	case "powers":
		return []float64{0, 1, 2, 4, 8}
	case "fibonacci":
		return []float64{0, 1, 2, 3, 5, 8}
	}
	return nil
}
