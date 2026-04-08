package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"jit-cli/internal/appinfo"
	"jit-cli/internal/profile"
)

func TestServiceListReadsCacheAndFiltersByKeyword(t *testing.T) {
	setupCachedAppProfile(t, profile.Config{
		Server:     "http://127.0.0.1:8080",
		DefaultApp: "whwy/mmm",
	}, &appinfo.AppInfo{
		AppID: "whwy/mmm",
		Elements: map[string]appinfo.ElementDefine{
			"corps.services.MemberSvc": {
				FullName: "corps.services.MemberSvc",
				Name:     "MemberSvc",
				Title:    "Member Service",
				Type:     "services.NormalType",
				FunctionList: []appinfo.FunctionDef{{
					Name: "getCurrUserInfo",
				}},
			},
			"models.Customer": {
				FullName: "models.Customer",
				Name:     "Customer",
				Title:    "Customer",
				Type:     "models.NormalType",
				FunctionList: []appinfo.FunctionDef{{
					Name: "queryByName",
				}},
			},
			"corps.services.PrivateSvc": {
				FullName:       "corps.services.PrivateSvc",
				Name:           "PrivateSvc",
				Title:          "Private Service",
				Type:           "services.NormalType",
				AccessModifier: "private",
				FunctionList: []appinfo.FunctionDef{{
					Name: "hidden",
				}},
			},
			"services.NoopSvc": {
				FullName: "services.NoopSvc",
				Name:     "NoopSvc",
				Title:    "Noop Service",
				Type:     "services.NormalType",
			},
		},
	})

	code, stdout, errOut := runCmdForTest(t, []string{
		"--profile", "demo",
		"service", "list",
		"--filter", "member",
	}, "", mockRuntime{})
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}

	var payload struct {
		AppID    string            `json:"appId"`
		Services []serviceListItem `json:"services"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, stdout=%s", err, stdout)
	}
	if payload.AppID != "whwy/mmm" {
		t.Fatalf("appId = %q, want whwy/mmm", payload.AppID)
	}
	if len(payload.Services) != 1 {
		t.Fatalf("len(services) = %d, want 1; stdout=%s", len(payload.Services), stdout)
	}
	if payload.Services[0].FullName != "corps.services.MemberSvc" {
		t.Fatalf("unexpected service list: %#v", payload.Services)
	}
	if len(payload.Services[0].Functions) != 1 || payload.Services[0].Functions[0] != "getCurrUserInfo" {
		t.Fatalf("unexpected function names: %#v", payload.Services[0].Functions)
	}
}

func TestServiceListDefaultsToLocalAppOnly(t *testing.T) {
	setupCachedAppProfile(t, profile.Config{
		Server:     "http://127.0.0.1:8080",
		DefaultApp: "whwy/mmm",
	}, &appinfo.AppInfo{
		AppID: "whwy/mmm",
		Elements: map[string]appinfo.ElementDefine{
			"corps.services.MemberSvc": {
				FullName: "corps.services.MemberSvc",
				Name:     "MemberSvc",
				Title:    "Member Service",
				Type:     "services.NormalType",
				FunctionList: []appinfo.FunctionDef{{
					Name: "getCurrUserInfo",
				}},
			},
		},
		ExtendApps: []appinfo.AppInfo{{
			AppID: "whwy/base",
			Elements: map[string]appinfo.ElementDefine{
				"corps.services.ExtendedSvc": {
					FullName: "corps.services.ExtendedSvc",
					Name:     "ExtendedSvc",
					Title:    "Extended Service",
					Type:     "services.NormalType",
					FunctionList: []appinfo.FunctionDef{{
						Name: "ping",
					}},
				},
			},
		}},
	})

	code, stdout, errOut := runCmdForTest(t, []string{
		"--profile", "demo",
		"service", "list",
	}, "", mockRuntime{})
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}

	var payload struct {
		Services []serviceListItem `json:"services"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, stdout=%s", err, stdout)
	}
	if len(payload.Services) != 1 {
		t.Fatalf("len(services) = %d, want 1; stdout=%s", len(payload.Services), stdout)
	}
	if payload.Services[0].FullName != "corps.services.MemberSvc" {
		t.Fatalf("unexpected default service list: %#v", payload.Services)
	}
}

func TestServiceListAllIncludesExtendedServices(t *testing.T) {
	setupCachedAppProfile(t, profile.Config{
		Server:     "http://127.0.0.1:8080",
		DefaultApp: "whwy/mmm",
	}, &appinfo.AppInfo{
		AppID: "whwy/mmm",
		Elements: map[string]appinfo.ElementDefine{
			"corps.services.MemberSvc": {
				FullName: "corps.services.MemberSvc",
				Name:     "MemberSvc",
				Title:    "Member Service",
				Type:     "services.NormalType",
				FunctionList: []appinfo.FunctionDef{{
					Name: "getCurrUserInfo",
				}},
			},
		},
		ExtendApps: []appinfo.AppInfo{{
			AppID: "whwy/base",
			Elements: map[string]appinfo.ElementDefine{
				"corps.services.ExtendedSvc": {
					FullName: "corps.services.ExtendedSvc",
					Name:     "ExtendedSvc",
					Title:    "Extended Service",
					Type:     "services.NormalType",
					FunctionList: []appinfo.FunctionDef{{
						Name: "ping",
					}},
				},
			},
		}},
	})

	code, stdout, errOut := runCmdForTest(t, []string{
		"--profile", "demo",
		"service", "list",
		"--all",
	}, "", mockRuntime{})
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}

	var payload struct {
		Services []serviceListItem `json:"services"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, stdout=%s", err, stdout)
	}
	if len(payload.Services) != 2 {
		t.Fatalf("len(services) = %d, want 2; stdout=%s", len(payload.Services), stdout)
	}
	if payload.Services[0].FullName != "corps.services.MemberSvc" || payload.Services[1].FullName != "corps.services.ExtendedSvc" {
		t.Fatalf("unexpected service list --all: %#v", payload.Services)
	}
}

func TestServiceExecValidatesCachedElementAndFunction(t *testing.T) {
	setupCachedAppProfile(t, profile.Config{
		Server:     "http://127.0.0.1:8080",
		DefaultApp: "whwy/mmm",
	}, &appinfo.AppInfo{
		AppID: "whwy/mmm",
		Elements: map[string]appinfo.ElementDefine{
			"corps.services.MemberSvc": {
				FullName: "corps.services.MemberSvc",
				Name:     "MemberSvc",
				Type:     "services.NormalType",
				FunctionList: []appinfo.FunctionDef{{
					Name: "getCurrUserInfo",
				}},
			},
		},
	})

	// element found in cache but function does not exist → function_not_found
	var got APIRequest
	rt := mockRuntime{
		callAPIFn: func(_ context.Context, req APIRequest) (APIResponse, error) {
			got = req
			return APIResponse{Raw: json.RawMessage(`{"errcode":0}`)}, nil
		},
	}

	code, _, errOut := runCmdForTest(t, []string{
		"--profile", "demo",
		"service", "exec", "corps.services.MemberSvc", "badFunc",
	}, "", rt)
	if code != ExitCLIError {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitCLIError, code, errOut)
	}
	if !strings.Contains(errOut, `"error":"function_not_found"`) {
		t.Fatalf("expected function_not_found, stderr=%s", errOut)
	}

	// element not in cache (inherited/private) → passes through to backend
	code, _, errOut = runCmdForTest(t, []string{
		"--profile", "demo",
		"service", "exec", "nonexist.Svc", "someFunc",
	}, "", rt)
	if code != ExitOK {
		t.Fatalf("expected exit %d (passthrough), got %d, stderr=%s", ExitOK, code, errOut)
	}
	if got.Endpoint != "nonexist/Svc/someFunc" {
		t.Fatalf("expected endpoint nonexist/Svc/someFunc, got %s", got.Endpoint)
	}
}

func TestServiceExecWithoutCacheFallsBackToAPIRequest(t *testing.T) {
	setupCachedAppProfile(t, profile.Config{
		Server:     "http://127.0.0.1:8080",
		DefaultApp: "whwy/mmm",
	}, nil)

	var got APIRequest
	rt := mockRuntime{
		callAPIFn: func(_ context.Context, req APIRequest) (APIResponse, error) {
			got = req
			return APIResponse{
				Raw: json.RawMessage(`{"dryRun":true,"url":"http://127.0.0.1:8080/api/whwy/mmm/corps/services/MemberSvc/getCurrUserInfo"}`),
			}, nil
		},
	}

	code, stdout, errOut := runCmdForTest(t, []string{
		"--profile", "demo",
		"--app", "whwy/mmm",
		"--dry-run",
		"service", "exec", "corps.services.MemberSvc", "getCurrUserInfo",
		"--data", `{"userId":"admin123"}`,
	}, "", rt)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}
	if !got.DryRun {
		t.Fatalf("expected dryRun request")
	}
	if got.Endpoint != "corps/services/MemberSvc/getCurrUserInfo" {
		t.Fatalf("unexpected endpoint: %+v", got)
	}
	if !strings.Contains(stdout, `"dryRun":true`) {
		t.Fatalf("expected dry run payload in stdout, got: %s", stdout)
	}
}
