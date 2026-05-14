package backlog

import (
	"strings"
	"testing"
)

func TestParseAcceptance_NoSection(t *testing.T) {
	body := "## Description\n\nSome words.\n"
	if got := ParseAcceptance(body); got != nil {
		t.Fatalf("expected nil for body without acceptance section, got %#v", got)
	}
}

func TestParseAcceptance_LegacyBareBullets(t *testing.T) {
	body := strings.Join([]string{
		"## Acceptance",
		"",
		"- a user can sign in",
		"- a wrong password produces a recoverable error",
		"* the session persists across reload",
		"",
		"## Description",
	}, "\n")
	got := ParseAcceptance(body)
	if len(got) != 3 {
		t.Fatalf("expected 3 bullets, got %d: %#v", len(got), got)
	}
	for i, b := range got {
		if b.State != AcceptanceOpen {
			t.Errorf("bullet %d: state=%s want open", i, b.State)
		}
	}
	if got[0].Text != "a user can sign in" {
		t.Errorf("bullet 0 text=%q", got[0].Text)
	}
	if got[2].Index != 3 {
		t.Errorf("bullet 2 index=%d want 3", got[2].Index)
	}
}

func TestParseAcceptance_AllStates(t *testing.T) {
	body := strings.Join([]string{
		"## Acceptance",
		"",
		"- [ ] open bullet",
		"- [~] claimed bullet",
		"- [x] verified bullet",
		"- [X] verified caps bullet",
		"",
	}, "\n")
	got := ParseAcceptance(body)
	if len(got) != 4 {
		t.Fatalf("expected 4 bullets, got %d", len(got))
	}
	wantStates := []AcceptanceState{AcceptanceOpen, AcceptanceClaimed, AcceptanceVerified, AcceptanceVerified}
	for i, b := range got {
		if b.State != wantStates[i] {
			t.Errorf("bullet %d: state=%s want %s", i, b.State, wantStates[i])
		}
	}
}

func TestParseAcceptance_ClaimNote(t *testing.T) {
	body := strings.Join([]string{
		"## Acceptance",
		"",
		"- [~] typo tolerance works <!-- claim: passes tests/search_test.go -->",
	}, "\n")
	got := ParseAcceptance(body)
	if len(got) != 1 {
		t.Fatalf("expected 1 bullet, got %d", len(got))
	}
	if got[0].Text != "typo tolerance works" {
		t.Errorf("text=%q", got[0].Text)
	}
	if got[0].ClaimNote != "passes tests/search_test.go" {
		t.Errorf("claim=%q", got[0].ClaimNote)
	}
}

func TestParseAcceptance_StopsAtNextHeading(t *testing.T) {
	body := strings.Join([]string{
		"## Acceptance",
		"- [ ] one",
		"## Description",
		"- not a bullet",
	}, "\n")
	got := ParseAcceptance(body)
	if len(got) != 1 {
		t.Fatalf("expected 1 bullet, got %d", len(got))
	}
}

func TestParseAcceptance_FirstSectionWins(t *testing.T) {
	body := strings.Join([]string{
		"## Acceptance",
		"- [ ] first",
		"## Description",
		"some text",
		"## Acceptance",
		"- [ ] second",
	}, "\n")
	got := ParseAcceptance(body)
	if len(got) != 1 {
		t.Fatalf("expected first section to win with 1 bullet, got %d", len(got))
	}
	if got[0].Text != "first" {
		t.Errorf("text=%q want first", got[0].Text)
	}
}

func TestAcceptanceBulletTexts_BackCompat(t *testing.T) {
	body := strings.Join([]string{
		"## Acceptance",
		"- [ ] one",
		"- [~] two",
		"- three",
	}, "\n")
	got := AcceptanceBulletTexts(body)
	want := []string{"one", "two", "three"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want=%d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] got=%q want=%q", i, got[i], want[i])
		}
	}
}

func TestSetAcceptanceState_FlipOpenToClaimed(t *testing.T) {
	body := strings.Join([]string{
		"## Acceptance",
		"- [ ] one",
		"- [ ] two",
	}, "\n")
	out, err := SetAcceptanceState(body, 2, AcceptanceClaimed, "passes tests")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(out, "- [~] two <!-- claim: passes tests -->") {
		t.Errorf("expected bullet 2 to be claimed with note; body=\n%s", out)
	}
	if !strings.Contains(out, "- [ ] one") {
		t.Errorf("bullet 1 should be untouched; body=\n%s", out)
	}
}

func TestSetAcceptanceState_FlipClaimedToVerified(t *testing.T) {
	body := strings.Join([]string{
		"## Acceptance",
		"- [~] one <!-- claim: passes -->",
	}, "\n")
	out, err := SetAcceptanceState(body, 1, AcceptanceVerified, "")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(out, "- [x] one") {
		t.Errorf("expected verified bullet without claim note; body=\n%s", out)
	}
	if strings.Contains(out, "<!-- claim:") {
		t.Errorf("verified bullet should not carry claim note; body=\n%s", out)
	}
}

func TestSetAcceptanceState_LegacyBulletNormalises(t *testing.T) {
	body := strings.Join([]string{
		"## Acceptance",
		"- legacy bullet text",
	}, "\n")
	out, err := SetAcceptanceState(body, 1, AcceptanceVerified, "")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(out, "- [x] legacy bullet text") {
		t.Errorf("expected legacy bullet normalised to checkbox; body=\n%s", out)
	}
}

func TestSetAcceptanceState_OutOfRange(t *testing.T) {
	body := "## Acceptance\n- [ ] only one\n"
	if _, err := SetAcceptanceState(body, 5, AcceptanceVerified, ""); err == nil {
		t.Errorf("expected error for out-of-range index")
	}
	if _, err := SetAcceptanceState(body, 0, AcceptanceVerified, ""); err == nil {
		t.Errorf("expected error for zero index")
	}
}

func TestAppendAcceptanceBullet_AddsToExistingSection(t *testing.T) {
	body := strings.Join([]string{
		"## Acceptance",
		"",
		"- [ ] one",
		"",
		"## Description",
		"text",
	}, "\n")
	out := AppendAcceptanceBullet(body, "two")
	if !strings.Contains(out, "- [ ] two") {
		t.Errorf("expected new bullet; body=\n%s", out)
	}
	if !strings.Contains(out, "## Description") {
		t.Errorf("Description heading should remain; body=\n%s", out)
	}
	// Order matters: bullet 2 should come before Description.
	twoIdx := strings.Index(out, "- [ ] two")
	descIdx := strings.Index(out, "## Description")
	if twoIdx > descIdx {
		t.Errorf("new bullet should appear before Description heading")
	}
}

func TestAppendAcceptanceBullet_CreatesSection(t *testing.T) {
	body := "## Description\ntext\n"
	out := AppendAcceptanceBullet(body, "first")
	if !strings.Contains(out, "## Acceptance\n\n- [ ] first") {
		t.Errorf("expected new section; body=\n%s", out)
	}
}

func TestAppendAcceptanceBullet_EmptyTextNoOp(t *testing.T) {
	body := "## Acceptance\n- [ ] one\n"
	out := AppendAcceptanceBullet(body, "   ")
	if out != body {
		t.Errorf("expected no-op for empty text")
	}
}
