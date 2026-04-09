package cmd

import (
	"encoding/json"
	"testing"

	"jit-cli/internal/appinfo"
	"jit-cli/internal/profile"
)

func TestAppGetReadsCachedAppInfo(t *testing.T) {
	setupCachedAppProfile(t, profile.Config{
		Server:     "http://127.0.0.1:8080",
		DefaultApp: "whwy/mmm",
	}, &appinfo.AppInfo{
		AppID:   "whwy/mmm",
		Name:    "mmm",
		Title:   "默认元素测试",
		Version: "1.0.0",
	})

	code, stdout, errOut := runCmdForTest(t, []string{
		"--profile", "demo",
		"app", "get",
	}, "", mockRuntime{})
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}

	var payload struct {
		App struct {
			Name    string `json:"name"`
			Title   string `json:"title"`
			AppID   string `json:"appId"`
			Version string `json:"version"`
		} `json:"app"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, stdout=%s", err, stdout)
	}
	if payload.App.AppID != "whwy/mmm" || payload.App.Name != "mmm" {
		t.Fatalf("unexpected app payload: %#v", payload.App)
	}
}

func TestAppLsReadsCachedElements(t *testing.T) {
	setupCachedAppProfile(t, profile.Config{
		Server:     "http://127.0.0.1:8080",
		DefaultApp: "whwy/mmm",
	}, &appinfo.AppInfo{
		AppID: "whwy/mmm",
		Elements: map[string]appinfo.ElementDefine{
			"models.Customer": {
				FullName: "models.Customer",
				Name:     "Customer",
				Title:    "客户",
				Type:     "models.NormalType",
			},
		},
	})

	code, stdout, errOut := runCmdForTest(t, []string{
		"--profile", "demo",
		"app", "ls",
	}, "", mockRuntime{})
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}

	var payload struct {
		AppID    string           `json:"appId"`
		Elements []elementSummary `json:"elements"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, stdout=%s", err, stdout)
	}
	if payload.AppID != "whwy/mmm" || len(payload.Elements) != 1 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if payload.Elements[0].FullName != "models.Customer" {
		t.Fatalf("unexpected elements: %#v", payload.Elements)
	}
}
