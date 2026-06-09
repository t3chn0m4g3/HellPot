package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/t3chn0m4g3/hellpot/internal/config"
)

func TestRunHelpDoesNotCreateConfig(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	var stdout, stderr bytes.Buffer
	code := run([]string{"--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run(--help) code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("stdout = %q, want usage output", stdout.String())
	}

	configPath := filepath.Join(configHome, "HellPot", "config.toml")
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("config path %q exists after --help or stat error = %v", configPath, err)
	}
}

func TestRunVersionNormalizesReleaseRefs(t *testing.T) {
	oldVersion := version
	oldConfigVersion := config.Version
	version = "refs/tags/v1.2.3"
	t.Cleanup(func() {
		version = oldVersion
		config.Version = oldConfigVersion
	})

	var stdout, stderr bytes.Buffer
	code := run([]string{"--version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run(--version) code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "1.2.3" {
		t.Fatalf("version output = %q, want 1.2.3", got)
	}
}

func TestRunVersionUsesProjectDefault(t *testing.T) {
	oldVersion := version
	oldConfigVersion := config.Version
	version = ""
	config.Version = "0.60"
	t.Cleanup(func() {
		version = oldVersion
		config.Version = oldConfigVersion
	})

	var stdout, stderr bytes.Buffer
	code := run([]string{"--version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run(--version) code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "0.60" {
		t.Fatalf("version output = %q, want 0.60", got)
	}
}

func TestRunGenConfigWritesCurrentDirectory(t *testing.T) {
	cwd := t.TempDir()
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Chdir(cwd)

	var stdout, stderr bytes.Buffer
	code := run([]string{"--genconfig"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run(--genconfig) code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(cwd, "config.toml")); err != nil {
		t.Fatalf("generated cwd config missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(configHome, "HellPot", "config.toml")); !os.IsNotExist(err) {
		t.Fatalf("config home file exists after --genconfig or stat error = %v", err)
	}
}
