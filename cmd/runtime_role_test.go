package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"jit-cli/internal/config"
	"jit-cli/internal/profile"
)

func TestCallAPIPassesThroughBackendRoleSelectionError(t *testing.T) {
	var paths []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/whwy/mmm/models/Customer/query":
			writeResponseJSON(t, w, map[string]any{
				"errcode": 20801008,
				"errmsg":  "Multiple application roles are available; please select one explicitly",
				"data": map[string]any{
					"roleList": []map[string]any{
						{"roleName": "roles.admin", "roleTitle": "管理员"},
						{"roleName": "roles.manager", "roleTitle": "经理"},
					},
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	rt := setupRuntimeWithToken(t, server.URL, "whwy/mmm", "jit_pat_tokenid_secret")
	resp, err := rt.CallAPI(context.Background(), APIRequest{
		Profile:  "demo",
		App:      "whwy/mmm",
		Endpoint: "models/Customer/query",
		Method:   http.MethodPost,
		Body: json.RawMessage(
			`{"methodType":"cls","argDict":{"filter":null,"fieldList":null,"orderList":null,"page":1,"size":10}}`,
		),
	})
	if err != nil {
		t.Fatalf("CallAPI() error = %v", err)
	}
	if !apiResponseHasBackendError(resp) {
		t.Fatalf("expected backend error, got: %s", string(resp.Raw))
	}

	expectedPaths := []string{
		"/api/whwy/mmm/models/Customer/query",
	}
	if !reflect.DeepEqual(paths, expectedPaths) {
		t.Fatalf("paths = %#v, want %#v", paths, expectedPaths)
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Raw, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["errcode"] != float64(20801008) {
		t.Fatalf("errcode = %#v, want 20801008", payload["errcode"])
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("data = %#v, want object", payload["data"])
	}
	roleList, ok := data["roleList"].([]any)
	if !ok || len(roleList) != 2 {
		t.Fatalf("roleList = %#v, want 2 roles", data["roleList"])
	}
}

func setupRuntimeWithToken(t *testing.T, serverURL string, appID string, token string) *appRuntime {
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

	if err := rt.profiles.SaveProfile("demo", profile.Config{
		Server:     serverURL,
		DefaultApp: appID,
	}); err != nil {
		t.Fatalf("SaveProfile() error = %v", err)
	}
	if err := rt.profiles.SaveToken("demo", token); err != nil {
		t.Fatalf("SaveToken() error = %v", err)
	}
	return rt
}

func decodeRequestBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(raw) == 0 {
		return map[string]any{}
	}

	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, body=%s", err, string(raw))
	}
	return body
}

func writeResponseJSON(t *testing.T, w http.ResponseWriter, payload map[string]any) {
	t.Helper()

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
}
