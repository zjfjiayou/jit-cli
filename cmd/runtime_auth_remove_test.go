package cmd

import (
	"context"
	"os"
	"testing"
)

func TestAuthRemoveDeletesProfileAndClearsCurrent(t *testing.T) {
	rt := newAppRuntimeForTest(t)
	saveRuntimeProfileForTest(t, rt, "demo", "http://127.0.0.1:8080")
	if err := os.WriteFile(rt.profiles.AppInfoPath("demo"), []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile(appinfo) error = %v", err)
	}
	if err := rt.setCurrentProfile("demo"); err != nil {
		t.Fatalf("setCurrentProfile() error = %v", err)
	}

	if err := rt.AuthRemove(context.Background(), "demo"); err != nil {
		t.Fatalf("AuthRemove() error = %v", err)
	}
	if rt.profiles.Exists("demo") {
		t.Fatalf("profile should be removed")
	}
	if _, err := os.Stat(rt.profiles.ProfileDir("demo")); !os.IsNotExist(err) {
		t.Fatalf("profile dir should be removed, stat err = %v", err)
	}

	cfg, err := rt.configSvc.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.CurrentProfile != "" {
		t.Fatalf("CurrentProfile = %q, want empty", cfg.CurrentProfile)
	}
}

func TestAuthRemoveRejectsMissingProfile(t *testing.T) {
	rt := newAppRuntimeForTest(t)
	assertCLIErrorKey(t, rt.AuthRemove(context.Background(), "missing"), "profile_not_found")
}
