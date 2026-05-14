package markdown

import (
	"strings"
	"testing"
)

func TestParseRoundtrip(t *testing.T) {
	src := `---
title: Build login flow
status: started
tags: [q2, auth]
estimate: 5
timeline:
  start: 2026-04-15
  end: 2026-05-10
---

## Problem statement

We need login.

## Comments

@alice why so?
`
	f, err := ParseFrontmatter(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if f.GetString("title") != "Build login flow" {
		t.Errorf("title = %q", f.GetString("title"))
	}
	if f.GetString("status") != "started" {
		t.Errorf("status = %q", f.GetString("status"))
	}
	tags := f.GetStringSlice("tags")
	if len(tags) != 2 || tags[0] != "q2" || tags[1] != "auth" {
		t.Errorf("tags = %v", tags)
	}
	if f.GetString("estimate") != "5" {
		t.Errorf("estimate = %q", f.GetString("estimate"))
	}
	tl := f.GetMap("timeline")
	if len(tl) != 2 || tl[0].Key != "start" || tl[0].Value != "2026-04-15" {
		t.Errorf("timeline = %v", tl)
	}
	if !strings.Contains(f.Body(), "Problem statement") {
		t.Errorf("body lost: %q", f.Body())
	}

	out := string(f.Bytes())
	if !strings.HasPrefix(out, "---\n") {
		t.Errorf("missing opener")
	}
	if !strings.Contains(out, "title: Build login flow") {
		t.Errorf("missing title in roundtrip:\n%s", out)
	}
	if !strings.Contains(out, "Problem statement") {
		t.Errorf("missing body in roundtrip:\n%s", out)
	}
}

func TestSetGetRemove(t *testing.T) {
	f, _ := ParseFrontmatter("---\nstatus: unstarted\n---\n\nhi\n")
	f.SetString("status", "started")
	if f.GetString("status") != "started" {
		t.Errorf("status not updated")
	}
	f.SetStringSlice("tags", []string{"a", "b"})
	if got := f.GetStringSlice("tags"); len(got) != 2 {
		t.Errorf("tags not set: %v", got)
	}
	f.SetString("status", "")
	if f.HasKey("status") {
		t.Errorf("status not removed by empty")
	}
	f.SetBool("archive", true)
	if !f.GetBool("archive") {
		t.Errorf("archive not true")
	}
	f.SetBool("archive", false)
	if f.HasKey("archive") {
		t.Errorf("archive not removed by false")
	}
}

func TestNoFrontmatter(t *testing.T) {
	src := "# Just markdown\n\nbody.\n"
	f, err := ParseFrontmatter(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if f.GetString("title") != "" {
		t.Errorf("expected empty fm, got %q", f.GetString("title"))
	}
	if !strings.Contains(f.Body(), "Just markdown") {
		t.Errorf("body lost: %q", f.Body())
	}
}

func TestKeyOrderPreserved(t *testing.T) {
	f, _ := ParseFrontmatter("---\nstatus: started\nassigned: alice\nestimate: 5\n---\n")
	f.SetString("estimate", "8")
	keys := f.Keys()
	want := []string{"status", "assigned", "estimate"}
	if len(keys) != len(want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
	for i, k := range want {
		if keys[i] != k {
			t.Errorf("keys[%d] = %q want %q", i, keys[i], k)
		}
	}
}
