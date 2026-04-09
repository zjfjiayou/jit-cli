package cmd

import (
	"context"
	"testing"
)

func TestAuthListAddsIndex(t *testing.T) {
	rt := newAppRuntimeForTest(t)
	saveRuntimeProfileForTest(t, rt, "a-demo", "http://127.0.0.1:8080")
	saveRuntimeProfileForTest(t, rt, "b-demo", "http://127.0.0.1:8081")

	items, err := rt.AuthList(context.Background())
	if err != nil {
		t.Fatalf("AuthList() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("AuthList() len = %d, want 2", len(items))
	}
	if items[0].Name != "a-demo" || items[0].Index != 0 {
		t.Fatalf("items[0] = %#v, want a-demo index 0", items[0])
	}
	if items[1].Name != "b-demo" || items[1].Index != 1 {
		t.Fatalf("items[1] = %#v, want b-demo index 1", items[1])
	}
}

func TestAuthUseSupportsIndex(t *testing.T) {
	rt := newAppRuntimeForTest(t)
	saveRuntimeProfileForTest(t, rt, "a-demo", "http://127.0.0.1:8080")
	saveRuntimeProfileForTest(t, rt, "b-demo", "http://127.0.0.1:8081")

	name, err := rt.AuthUse(context.Background(), "1")
	if err != nil {
		t.Fatalf("AuthUse() error = %v", err)
	}
	if name != "b-demo" {
		t.Fatalf("AuthUse() name = %q, want b-demo", name)
	}

	cfg, err := rt.configSvc.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.CurrentProfile != "b-demo" {
		t.Fatalf("CurrentProfile = %q, want b-demo", cfg.CurrentProfile)
	}
}

func TestAuthUsePrefersProfileNameOverIndex(t *testing.T) {
	rt := newAppRuntimeForTest(t)
	saveRuntimeProfileForTest(t, rt, "0", "http://127.0.0.1:8080")
	saveRuntimeProfileForTest(t, rt, "z-demo", "http://127.0.0.1:8081")

	name, err := rt.AuthUse(context.Background(), "0")
	if err != nil {
		t.Fatalf("AuthUse() error = %v", err)
	}
	if name != "0" {
		t.Fatalf("AuthUse() name = %q, want literal profile 0", name)
	}
}

func TestAuthUseRejectsOutOfRangeIndex(t *testing.T) {
	rt := newAppRuntimeForTest(t)
	saveRuntimeProfileForTest(t, rt, "demo", "http://127.0.0.1:8080")
	_, err := rt.AuthUse(context.Background(), "1")
	assertCLIErrorKey(t, err, "profile_index_out_of_range")
}
