package config

import (
	"path/filepath"
	"testing"
)

func TestResolveHomeDirUsesEnv(t *testing.T) {
	t.Setenv(EnvHome, "/tmp/custom-jit-home")
	got, err := ResolveHomeDir()
	if err != nil {
		t.Fatalf("ResolveHomeDir() error = %v", err)
	}
	if got != filepath.Clean("/tmp/custom-jit-home") {
		t.Fatalf("ResolveHomeDir() = %q, want %q", got, "/tmp/custom-jit-home")
	}
}

func TestGlobalConfigRoundTrip(t *testing.T) {
	home := t.TempDir()
	svc, err := NewService(home)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	in := GlobalConfig{
		CurrentProfile: "demo",
		DefaultFormat:  "json",
	}
	if err := svc.Save(in); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := svc.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got != in {
		t.Fatalf("Load() = %#v, want %#v", got, in)
	}
}

func TestLoadMissingFileReturnsDefault(t *testing.T) {
	svc, err := NewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	cfg, err := svc.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DefaultFormat != "json" {
		t.Fatalf("default format = %q, want json", cfg.DefaultFormat)
	}
}
