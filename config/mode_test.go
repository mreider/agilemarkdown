package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultModeIsCoached(t *testing.T) {
	if got := Defaults().Mode; got != ModeCoached {
		t.Fatalf("default mode = %q, want %q", got, ModeCoached)
	}
	if Defaults().IsAutonomous() {
		t.Fatal("defaults must not be autonomous")
	}
}

func TestLoadConfigMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	body := "mode: autonomous\nestimation:\n  scale: fibonacci\niteration:\n  length_weeks: 1\nvelocity:\n  strategy: rolling\n"
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !cfg.IsAutonomous() {
		t.Fatalf("expected autonomous, got %q", cfg.EffectiveMode())
	}
}

func TestLoadConfigRejectsBadMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("mode: cowboy\nestimation:\n  scale: fibonacci\niteration:\n  length_weeks: 1\nvelocity:\n  strategy: rolling\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("expected an error for an invalid mode, got nil")
	}
}

func TestEffectiveModeEnvOverride(t *testing.T) {
	cfg := Defaults() // coached

	t.Setenv("AM_MODE", "autonomous")
	if cfg.EffectiveMode() != ModeAutonomous {
		t.Fatalf("AM_MODE=autonomous should override coached config")
	}

	t.Setenv("AM_MODE", "coached")
	autoCfg := Defaults()
	autoCfg.Mode = ModeAutonomous
	if autoCfg.EffectiveMode() != ModeCoached {
		t.Fatalf("AM_MODE=coached should override autonomous config")
	}

	// An invalid env value is ignored; the configured mode wins.
	t.Setenv("AM_MODE", "nonsense")
	if autoCfg.EffectiveMode() != ModeAutonomous {
		t.Fatalf("invalid AM_MODE should fall back to configured mode")
	}
}
