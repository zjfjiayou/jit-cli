package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"jit-cli/internal/appinfo"
	"jit-cli/internal/config"
	"jit-cli/internal/profile"
)

func TestModelListRequiresAppInfoCache(t *testing.T) {
	setupCachedAppProfile(t, profile.Config{
		Server:     "http://127.0.0.1:8080",
		DefaultApp: "whwy/mmm",
	}, nil)

	code, _, errOut := runCmdForTest(t, []string{
		"--profile", "demo",
		"model", "list",
	}, "", mockRuntime{})
	if code != ExitCLIError {
		t.Fatalf("expected exit %d, got %d", ExitCLIError, code)
	}
	if !strings.Contains(errOut, `"error":"missing_appinfo_cache"`) {
		t.Fatalf("expected missing_appinfo_cache, stderr=%s", errOut)
	}
}

func TestModelListReadsCacheAndFiltersPrivateElements(t *testing.T) {
	setupCachedAppProfile(t, profile.Config{
		Server:     "http://127.0.0.1:8080",
		DefaultApp: "whwy/mmm",
	}, &appinfo.AppInfo{
		AppID: "whwy/mmm",
		Elements: map[string]appinfo.ElementDefine{
			"models.PublicModel": {
				FullName: "models.PublicModel",
				Name:     "PublicModel",
				Title:    "Public Model",
				Type:     "models.NormalType",
				FieldList: []map[string]any{{
					"name": "id",
				}},
				Meta: map[string]any{"database": "demo"},
			},
			"models.PrivateModel": {
				FullName:       "models.PrivateModel",
				Name:           "PrivateModel",
				Title:          "Private Model",
				Type:           "models.NormalType",
				AccessModifier: "private",
				FieldList: []map[string]any{{
					"name": "id",
				}},
			},
			"models.PublicModel.defaultForm": {
				FullName: "models.PublicModel.defaultForm",
				Name:     "defaultForm",
				Title:    "Default Form",
				Type:     "components.Form",
			},
			"services.PublicSvc": {
				FullName: "services.PublicSvc",
				Type:     "services.Meta",
			},
		},
		ExtendApps: []appinfo.AppInfo{{
			AppID: "whwy/base",
			Elements: map[string]appinfo.ElementDefine{
				"pays.models.OrderModel": {
					FullName: "pays.models.OrderModel",
					Name:     "OrderModel",
					Title:    "Order",
					Type:     "models.NormalType",
				},
			},
		}},
	})

	code, stdout, errOut := runCmdForTest(t, []string{
		"--profile", "demo",
		"model", "list",
	}, "", mockRuntime{})
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}

	var payload struct {
		Data []elementSummary `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, stdout=%s", err, stdout)
	}
	if len(payload.Data) != 1 {
		t.Fatalf("len(data) = %d, want 1; stdout=%s", len(payload.Data), stdout)
	}
	if payload.Data[0].FullName != "models.PublicModel" {
		t.Fatalf("unexpected model list: %#v", payload.Data)
	}
}

func TestModelListAllIncludesExtendedModels(t *testing.T) {
	setupCachedAppProfile(t, profile.Config{
		Server:     "http://127.0.0.1:8080",
		DefaultApp: "whwy/mmm",
	}, &appinfo.AppInfo{
		AppID: "whwy/mmm",
		Elements: map[string]appinfo.ElementDefine{
			"models.PublicModel": {
				FullName: "models.PublicModel",
				Name:     "PublicModel",
				Title:    "Public Model",
				Type:     "models.NormalType",
			},
		},
		ExtendApps: []appinfo.AppInfo{{
			AppID: "whwy/base",
			Elements: map[string]appinfo.ElementDefine{
				"pays.models.OrderModel": {
					FullName: "pays.models.OrderModel",
					Name:     "OrderModel",
					Title:    "Order",
					Type:     "models.NormalType",
				},
			},
		}},
	})

	code, stdout, errOut := runCmdForTest(t, []string{
		"--profile", "demo",
		"model", "list",
		"--all",
	}, "", mockRuntime{})
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}

	var payload struct {
		Data []elementSummary `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, stdout=%s", err, stdout)
	}
	if len(payload.Data) != 2 {
		t.Fatalf("len(data) = %d, want 2; stdout=%s", len(payload.Data), stdout)
	}
	if payload.Data[0].FullName != "models.PublicModel" || payload.Data[1].FullName != "pays.models.OrderModel" {
		t.Fatalf("unexpected model list --all: %#v", payload.Data)
	}
}

func setupCachedAppProfile(t *testing.T, cfg profile.Config, info *appinfo.AppInfo) {
	t.Helper()

	home := t.TempDir()
	t.Setenv(config.EnvHome, home)

	configSvc, err := config.NewService(home)
	if err != nil {
		t.Fatalf("config.NewService() error = %v", err)
	}
	if err := configSvc.Save(config.GlobalConfig{CurrentProfile: "demo"}); err != nil {
		t.Fatalf("configSvc.Save() error = %v", err)
	}

	profiles, err := profile.NewManager(home)
	if err != nil {
		t.Fatalf("profile.NewManager() error = %v", err)
	}
	if err := profiles.SaveProfile("demo", cfg); err != nil {
		t.Fatalf("profiles.SaveProfile() error = %v", err)
	}
	if info == nil {
		return
	}
	if err := appinfo.Save(profiles.AppInfoPath("demo"), info); err != nil {
		t.Fatalf("appinfo.Save() error = %v", err)
	}
}
