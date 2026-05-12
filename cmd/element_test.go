package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestElementGetBuildsResourceRequest(t *testing.T) {
	var got APIRequest
	rt := mockRuntime{
		callAPIFn: func(_ context.Context, req APIRequest) (APIResponse, error) {
			got = req
			return APIResponse{Raw: json.RawMessage(`{"errcode":0,"data":{"service.py":"code"}}`)}, nil
		},
	}

	code, _, errOut := runCmdForTest(t, []string{
		"--profile", "demo",
		"element", "get", "services.ElementSvc",
		"--resources", `["service.py","e.json"]`,
		"--ignore", `["dist/*"]`,
	}, "", rt)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}
	if got.Endpoint != elementSvcGetResource {
		t.Fatalf("endpoint = %s, want %s", got.Endpoint, elementSvcGetResource)
	}

	body := requireJSONMap(t, got.Body)
	if body["fullName"] != "services.ElementSvc" {
		t.Fatalf("fullName = %#v, want services.ElementSvc", body["fullName"])
	}
	resources := body["resources"].([]any)
	if len(resources) != 2 || resources[0] != "service.py" || resources[1] != "e.json" {
		t.Fatalf("resources = %#v", body["resources"])
	}
	ignore := body["ignore"].([]any)
	if len(ignore) != 1 || ignore[0] != "dist/*" {
		t.Fatalf("ignore = %#v", body["ignore"])
	}
}

func TestElementGetRejectsInvalidResourceList(t *testing.T) {
	assertValidationFailure(t, []string{
		"element", "get", "services.ElementSvc",
		"--resources", `{"bad":true}`,
	}, `"error":"invalid_resources"`)
}

func TestElementSaveBuildsSaveResourceRequest(t *testing.T) {
	var got APIRequest
	rt := mockRuntime{
		callAPIFn: func(_ context.Context, req APIRequest) (APIResponse, error) {
			got = req
			return APIResponse{Raw: json.RawMessage(`{"errcode":0}`)}, nil
		},
	}

	code, _, errOut := runCmdForTest(t, []string{
		"--profile", "demo",
		"element", "save", "services.DemoSvc",
		"--resources", `{"service.py":"print(1)"}`,
	}, "", rt)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}
	if got.Endpoint != elementSvcSaveResource {
		t.Fatalf("endpoint = %s, want %s", got.Endpoint, elementSvcSaveResource)
	}

	body := requireJSONMap(t, got.Body)
	if body["fullName"] != "services.DemoSvc" {
		t.Fatalf("fullName = %#v, want services.DemoSvc", body["fullName"])
	}
	resources, ok := body["resources"].(map[string]any)
	if !ok || resources["service.py"] != "print(1)" {
		t.Fatalf("resources = %#v", body["resources"])
	}
}

func TestElementSaveReadsResourcesFromStdin(t *testing.T) {
	var got APIRequest
	rt := mockRuntime{
		callAPIFn: func(_ context.Context, req APIRequest) (APIResponse, error) {
			got = req
			return APIResponse{Raw: json.RawMessage(`{"errcode":0}`)}, nil
		},
	}

	code, _, errOut := runCmdForTest(t, []string{
		"element", "save", "services.DemoSvc",
		"--resources", "@-",
	}, `{"service.py":"print(2)"}`, rt)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}

	body := requireJSONMap(t, got.Body)
	resources, ok := body["resources"].(map[string]any)
	if !ok || resources["service.py"] != "print(2)" {
		t.Fatalf("resources = %#v", body["resources"])
	}
}

func TestElementSaveRequiresObjectResources(t *testing.T) {
	assertValidationFailure(t, []string{
		"element", "save", "services.DemoSvc",
		"--resources", `["service.py"]`,
	}, `"error":"invalid_data"`)
}

func TestElementApplyBuildsSaveElementRequest(t *testing.T) {
	var got APIRequest
	rt := mockRuntime{
		callAPIFn: func(_ context.Context, req APIRequest) (APIResponse, error) {
			got = req
			return APIResponse{Raw: json.RawMessage(`{"errcode":0}`)}, nil
		},
	}

	code, _, errOut := runCmdForTest(t, []string{
		"element", "apply",
		"--data", `[{"ePath":"services/DemoSvc/e.json","define":{"name":"DemoSvc"},"resources":{"service.py":"pass"}}]`,
	}, "", rt)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}
	if got.Endpoint != elementSvcSaveElement {
		t.Fatalf("endpoint = %s, want %s", got.Endpoint, elementSvcSaveElement)
	}

	body := requireJSONMap(t, got.Body)
	items, ok := body["elementList"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("elementList = %#v", body["elementList"])
	}
}

func TestElementBuildSearchAndKnowledgeUseExpectedEndpoints(t *testing.T) {
	var requests []APIRequest
	rt := mockRuntime{
		callAPIFn: func(_ context.Context, req APIRequest) (APIResponse, error) {
			requests = append(requests, req)
			return APIResponse{Raw: json.RawMessage(`{"errcode":0}`)}, nil
		},
	}

	cases := [][]string{
		{"element", "build", "services.DemoSvc", "models.Customer"},
		{"element", "search", "ElementSvc"},
		{"element", "knowledge", "services.ElementSvc"},
	}
	for _, args := range cases {
		code, _, errOut := runCmdForTest(t, args, "", rt)
		if code != ExitOK {
			t.Fatalf("args=%v expected exit %d, got %d, stderr=%s", args, ExitOK, code, errOut)
		}
	}

	if got := requests[0].Endpoint; got != elementSvcBuildElement {
		t.Fatalf("build endpoint = %s, want %s", got, elementSvcBuildElement)
	}
	buildBody := requireJSONMap(t, requests[0].Body)
	fullNames := buildBody["fullNames"].([]any)
	if len(fullNames) != 2 || fullNames[0] != "services.DemoSvc" || fullNames[1] != "models.Customer" {
		t.Fatalf("fullNames = %#v", buildBody["fullNames"])
	}

	if got := requests[1].Endpoint; got != elementSvcSearchContent {
		t.Fatalf("search endpoint = %s, want %s", got, elementSvcSearchContent)
	}
	searchBody := requireJSONMap(t, requests[1].Body)
	if searchBody["pattern"] != "ElementSvc" {
		t.Fatalf("pattern = %#v, want ElementSvc", searchBody["pattern"])
	}

	if got := requests[2].Endpoint; got != elementSvcGetKnowledge {
		t.Fatalf("knowledge endpoint = %s, want %s", got, elementSvcGetKnowledge)
	}
	knowledgeBody := requireJSONMap(t, requests[2].Body)
	if knowledgeBody["elementFullName"] != "services.ElementSvc" {
		t.Fatalf("elementFullName = %#v, want services.ElementSvc", knowledgeBody["elementFullName"])
	}
}

func TestElementBuildRequiresAtLeastOneName(t *testing.T) {
	code, _, errOut := runCmdForTest(t, []string{
		"element", "build",
	}, "", mockRuntime{})
	if code != ExitCLIError {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitCLIError, code, errOut)
	}
	if !strings.Contains(errOut, `"error":"missing_element"`) {
		t.Fatalf("expected missing_element, stderr=%s", errOut)
	}
}

func TestAppBuildCallsElementBuildApp(t *testing.T) {
	var got APIRequest
	rt := mockRuntime{
		callAPIFn: func(_ context.Context, req APIRequest) (APIResponse, error) {
			got = req
			return APIResponse{Raw: json.RawMessage(`{"errcode":0}`)}, nil
		},
	}

	code, _, errOut := runCmdForTest(t, []string{
		"--profile", "demo",
		"app", "build",
	}, "", rt)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}
	if got.Endpoint != elementSvcBuildApp {
		t.Fatalf("endpoint = %s, want %s", got.Endpoint, elementSvcBuildApp)
	}
	body := requireJSONMap(t, got.Body)
	if len(body) != 0 {
		t.Fatalf("body = %#v, want empty object", body)
	}
}
