package backlog

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// IterationOverride is a per-iteration record that mirrors Pivotal
// Tracker's `iteration_override` API resource. The number is the 1-based
// iteration number (the same number used by `iterationNumberFor` in
// views.go and the velocity math). Either or both fields can be set.
//
//   team_strength: float, 1.0 = full strength. 0 excludes the iteration
//                  from velocity entirely. Values >1.0 are allowed.
//   length_weeks:  int, overrides the project-default iteration length
//                  for this one iteration only.
type IterationOverride struct {
	Number       int     `yaml:"number"`
	TeamStrength float64 `yaml:"team_strength,omitempty"`
	LengthWeeks  int     `yaml:"length_weeks,omitempty"`
}

// IterationOverrides is the parsed contents of `.am/iterations.yaml`. The
// file is sparse: only iterations that deviate from project defaults
// have a record. Order on disk is by ascending number.
type IterationOverrides struct {
	Overrides []IterationOverride `yaml:"overrides"`
}

const iterationOverridesFileName = ".am/iterations.yaml"

// IterationOverridesFile returns the absolute path to the iteration
// overrides file under a project root.
func IterationOverridesFile(rootDir string) string {
	return filepath.Join(rootDir, iterationOverridesFileName)
}

// LoadIterationOverrides reads `.am/iterations.yaml`. Missing file
// returns an empty struct, not an error.
func LoadIterationOverrides(rootDir string) (*IterationOverrides, error) {
	path := IterationOverridesFile(rootDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &IterationOverrides{}, nil
		}
		return nil, err
	}
	out := &IterationOverrides{}
	if err := yaml.Unmarshal(data, out); err != nil {
		return nil, fmt.Errorf("iterations.yaml: %w", err)
	}
	sort.Slice(out.Overrides, func(i, j int) bool {
		return out.Overrides[i].Number < out.Overrides[j].Number
	})
	return out, nil
}

// Save writes the overrides back to disk. Empty list still writes the
// file so callers can detect intent. Sorted by number on the way out.
func (o *IterationOverrides) Save(rootDir string) error {
	sort.Slice(o.Overrides, func(i, j int) bool {
		return o.Overrides[i].Number < o.Overrides[j].Number
	})
	path := IterationOverridesFile(rootDir)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(o)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Find returns the override for the given iteration number, or nil.
func (o *IterationOverrides) Find(number int) *IterationOverride {
	for i := range o.Overrides {
		if o.Overrides[i].Number == number {
			return &o.Overrides[i]
		}
	}
	return nil
}

// Set applies the given fields to iteration `number`. Pass strength<0 or
// length<0 to leave that field unchanged. To clear a field, see Clear.
func (o *IterationOverrides) Set(number int, strength float64, length int) {
	rec := o.Find(number)
	if rec == nil {
		o.Overrides = append(o.Overrides, IterationOverride{Number: number})
		rec = &o.Overrides[len(o.Overrides)-1]
	}
	if strength >= 0 {
		rec.TeamStrength = strength
	}
	if length > 0 {
		rec.LengthWeeks = length
	}
}

// Clear removes the override record for `number`, if any.
func (o *IterationOverrides) Clear(number int) {
	for i := range o.Overrides {
		if o.Overrides[i].Number == number {
			o.Overrides = append(o.Overrides[:i], o.Overrides[i+1:]...)
			return
		}
	}
}

// StrengthFor returns the team strength for iteration `number`, or 1.0
// (full strength) if no override is set.
func (o *IterationOverrides) StrengthFor(number int) float64 {
	rec := o.Find(number)
	if rec == nil || rec.TeamStrength == 0 {
		// 0 means "excluded entirely" only when the user explicitly set
		// it; an absent record defaults to 1.0.
		if rec == nil {
			return 1.0
		}
	}
	return rec.TeamStrength
}

// LengthWeeksFor returns the length-weeks override for iteration
// `number`, or 0 if no override (meaning fall back to project default).
func (o *IterationOverrides) LengthWeeksFor(number int) int {
	rec := o.Find(number)
	if rec == nil {
		return 0
	}
	return rec.LengthWeeks
}
