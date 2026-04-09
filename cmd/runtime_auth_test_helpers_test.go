package cmd

import (
	"testing"

	"jit-cli/internal/config"
	"jit-cli/internal/profile"
)

func newAppRuntimeForTest(t *testing.T) *appRuntime {
	t.Helper()

	home := t.TempDir()
	t.Setenv(config.EnvHome, home)

	rtIface, err := NewAppRuntime()
	if err != nil {
		t.Fatalf("NewAppRuntime() error = %v", err)
	}
	rt, ok := rtIface.(*appRuntime)
	if !ok {
		t.Fatalf("runtime type = %T, want *appRuntime", rtIface)
	}
	return rt
}

func saveRuntimeProfileForTest(t *testing.T, rt *appRuntime, name string, server string) {
	t.Helper()

	if err := rt.profiles.SaveProfile(name, profile.Config{
		Server:     server,
		DefaultApp: "whwy/mmm",
	}); err != nil {
		t.Fatalf("SaveProfile(%s) error = %v", name, err)
	}
}

func assertCLIErrorKey(t *testing.T, err error, wantKey string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error")
	}
	cliErr, ok := err.(*CLIError)
	if !ok {
		t.Fatalf("error type = %T, want *CLIError", err)
	}
	if cliErr.Key != wantKey {
		t.Fatalf("error key = %q, want %q", cliErr.Key, wantKey)
	}
}
