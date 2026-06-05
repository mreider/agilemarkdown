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

func writeInception(t *testing.T, root string) {
	t.Helper()
	body := `# Inception

## The user

A terminal-dwelling DJ who mixes from the shell.

## The goal

Two-deck mix with sync and crossfade.

## Out of scope

GUI, mobile, streaming services.
`
	if err := os.MkdirAll(filepath.Join(root, ".am"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, inceptionFileName), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

// inceptionBrief surfaces the two orienting sections and deliberately
// leaves the rest (out-of-scope, success, constraints) for
// `am inception --show`, so a SessionStart hook stays lean.
func TestInceptionBriefExtractsUserAndGoalOnly(t *testing.T) {
	root := t.TempDir()
	writeInception(t, root)

	got := inceptionBrief(root)
	for _, want := range []string{"The user", "terminal-dwelling DJ", "The goal", "crossfade"} {
		if !strings.Contains(got, want) {
			t.Fatalf("inceptionBrief missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Out of scope") || strings.Contains(got, "streaming services") {
		t.Fatalf("inceptionBrief should not include out-of-scope:\n%s", got)
	}
}

func TestInceptionBriefMissingReturnsEmpty(t *testing.T) {
	if got := inceptionBrief(t.TempDir()); got != "" {
		t.Fatalf("expected empty brief with no inception.md, got %q", got)
	}
}

func TestAgreementsBriefReturnsBulletsCapped(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".am"), 0755); err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	sb.WriteString("# Team agreements\n\nSome prose that is not a bullet.\n\n")
	for i := 0; i < 20; i++ {
		sb.WriteString("- agreement line\n")
	}
	if err := os.WriteFile(filepath.Join(root, agreementsFileName), []byte(sb.String()), 0644); err != nil {
		t.Fatal(err)
	}

	got := agreementsBrief(root)
	if !strings.Contains(got, "agreement line") {
		t.Fatalf("agreementsBrief should include bullets:\n%s", got)
	}
	if strings.Contains(got, "Some prose") {
		t.Fatalf("agreementsBrief should only take bullets, not prose:\n%s", got)
	}
	if !strings.Contains(got, "…") {
		t.Fatalf("agreementsBrief should mark truncation with an ellipsis when capped:\n%s", got)
	}
}

// End-to-end: `am brief` composes every section against a real (if empty)
// backlog, exercising the shared dashboardText/nextText helpers too.
func TestBriefCommandComposesAllSections(t *testing.T) {
	root, _ := canonicalProject(t) // .git + stories/ + stories.md
	if err := os.MkdirAll(filepath.Join(root, ".am"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := config.Defaults().Save(filepath.Join(root, ".am", "config.yaml")); err != nil {
		t.Fatal(err)
	}
	writeInception(t, root)
	if err := os.WriteFile(filepath.Join(root, agreementsFileName), []byte("# Team agreements\n\n- be kind\n"), 0644); err != nil {
		t.Fatal(err)
	}

	chdir(t, root)
	app := &cli.Command{Commands: []*cli.Command{BriefCommand}}
	stdout, stderr := captureStdio(t)
	if err := app.Run(context.Background(), []string{"am", "brief"}); err != nil {
		stdout.restore()
		stderr.restore()
		t.Fatalf("am brief: %v", err)
	}
	stdout.restore()
	stderr.restore()

	out := stdout.buf.String()
	for _, want := range []string{
		"agent brief",
		"## Project",
		"The user",
		"## Dashboard",
		"velocity:",
		"## Next pull",
		"## In progress",
		"## Working agreements",
		"be kind",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("am brief output missing %q in:\n%s", want, out)
		}
	}
}
