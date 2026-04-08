package appinfo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchDecryptsAndParsesAppInfo(t *testing.T) {
	key := "whwy.mmm"
	payload := map[string]any{
		"orgId":   "whwy",
		"appId":   "whwy.mmm",
		"name":    "mmm",
		"title":   "Demo",
		"version": "1.0.0",
		"elements": map[string]any{
			"services.FooSvc": map[string]any{
				"fullName":       "services.FooSvc",
				"name":           "FooSvc",
				"title":          "Foo",
				"type":           "services.Meta",
				"accessModifier": nil,
				"functionList": []map[string]any{{
					"name":       "ping",
					"title":      "Ping",
					"returnType": "None",
					"args":       []map[string]any{{"name": "message", "title": "Message", "dataType": "Stext"}},
				}},
			},
		},
		"extendApps": []map[string]any{{
			"orgId": "whwy",
			"appId": "whwy.base",
			"name":  "base",
			"title": "Base",
			"elements": map[string]any{
				"models.BaseModel": map[string]any{
					"fullName": "models.BaseModel",
					"name":     "BaseModel",
					"title":    "Base Model",
					"type":     "models.NormalType",
					"fieldList": []map[string]any{{
						"name": "id",
					}},
				},
			},
		}},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	encrypted := xorBytes([]byte("var JIT_APP_INFO = "+string(data)+";"), []byte(key))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer demo-token" {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
		_, _ = w.Write(encrypted)
	}))
	defer server.Close()

	info, err := Fetch(context.Background(), server.URL, "whwy", "mmm", "demo-token")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if info.AppID != "whwy/mmm" {
		t.Fatalf("AppID = %q, want whwy/mmm", info.AppID)
	}
	if info.Elements["services.FooSvc"].FullName != "services.FooSvc" {
		t.Fatalf("unexpected fullName: %#v", info.Elements["services.FooSvc"])
	}
	if len(info.ExtendApps) != 1 || info.ExtendApps[0].AppID != "whwy/base" {
		t.Fatalf("unexpected extend apps: %#v", info.ExtendApps)
	}
}

func TestExtractAppInfoJSONRejectsMissingAssignment(t *testing.T) {
	_, err := extractAppInfoJSON("console.log('missing');")
	if err == nil || !strings.Contains(err.Error(), "JIT_APP_INFO") {
		t.Fatalf("extractAppInfoJSON() error = %v, want missing assignment", err)
	}
}

func xorBytes(input []byte, key []byte) []byte {
	out := make([]byte, len(input))
	for i, b := range input {
		out[i] = b ^ key[i%len(key)]
	}
	return out
}
