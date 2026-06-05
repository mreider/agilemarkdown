package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mreider/agilemarkdown/config"
	"github.com/urfave/cli/v3"
)

func writeMode(t *testing.T, root, mode string) {
	t.Helper()
	cfg := config.Defaults()
	cfg.Mode = mode
	if err := cfg.Save(filepath.Join(root, ".am", "config.yaml")); err != nil {
		t.Fatal(err)
	}
}

// In coached mode the dev pair cannot accept its own work — the seam is
// a hard gate.
func TestRunCoachCheckAcceptCoachedRefuses(t *testing.T) {
	root, _ := canonicalProject(t)
	writeMode(t, root, config.ModeCoached)
	chdir(t, root)
	t.Setenv("AM_MODE", "")

	v, err := runCoachCheck("set_status", "stories/x.md", "accepted", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if v.Allowed {
		t.Fatalf("coached mode must refuse a dev-pair accept")
	}
}

// Autonomous mode collapses the seam: accept is allowed, surfaced as a
// nudge to record the rationale.
func TestRunCoachCheckAcceptAutonomousAllows(t *testing.T) {
	root, _ := canonicalProject(t)
	writeMode(t, root, config.ModeAutonomous)
	chdir(t, root)
	t.Setenv("AM_MODE", "")

	v, err := runCoachCheck("set_status", "stories/x.md", "accepted", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !v.Allowed {
		t.Fatalf("autonomous mode must allow a dev-pair accept; got refusal: %+v", v)
	}
	if !v.Nudge {
		t.Fatalf("autonomous accept should be a nudge (record rationale): %+v", v)
	}
}

// AM_MODE overrides the configured (coached) mode for a single session.
func TestRunCoachCheckAcceptEnvOverride(t *testing.T) {
	root, _ := canonicalProject(t)
	writeMode(t, root, config.ModeCoached)
	chdir(t, root)
	t.Setenv("AM_MODE", "autonomous")

	v, err := runCoachCheck("set_status", "stories/x.md", "accepted", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !v.Allowed {
		t.Fatalf("AM_MODE=autonomous should lift the accept gate over coached config")
	}
}

func TestJournalCommandGroupsByStory(t *testing.T) {
	root, _ := canonicalProject(t)
	if err := os.MkdirAll(filepath.Join(root, ".am"), 0755); err != nil {
		t.Fatal(err)
	}
	log := "2026-06-05T10:00:00Z\talice\tstarted\tstories/x.md\t\n" +
		"2026-06-05T12:00:00Z\talice\taccepted\tstories/x.md\tAC met: verified\n"
	if err := os.WriteFile(filepath.Join(root, ".am", "activity.log"), []byte(log), 0644); err != nil {
		t.Fatal(err)
	}
	writeMode(t, root, config.ModeCoached)
	chdir(t, root)
	t.Setenv("AM_MODE", "")

	app := &cli.Command{Commands: []*cli.Command{JournalCommand}}
	stdout, stderr := captureStdio(t)
	if err := app.Run(context.Background(), []string{"am", "journal"}); err != nil {
		stdout.restore()
		stderr.restore()
		t.Fatalf("am journal: %v", err)
	}
	stdout.restore()
	stderr.restore()

	out := stdout.buf.String()
	for _, want := range []string{"Journal", "mode: coached", "started", "accepted", "AC met: verified"} {
		if !strings.Contains(out, want) {
			t.Fatalf("am journal output missing %q in:\n%s", want, out)
		}
	}
}
