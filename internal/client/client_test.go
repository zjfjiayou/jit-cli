package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestCallDryRunDoesNotSendRequest(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errcode":0}`))
	}))
	defer server.Close()

	c := New(&http.Client{Timeout: 2 * time.Second})
	result, err := c.Call(context.Background(), Request{
		Server:   server.URL,
		App:      "wanyun/JitAuth",
		Endpoint: "auths/loginTypes/services/AuthSvc/listCliTokens",
		Token:    "jit_pat_token_secret",
		Body:     map[string]any{},
		DryRun:   true,
	})
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if result == nil || result.DryRun == nil {
		t.Fatalf("Call() dry run result is nil")
	}
	if result.Response != nil {
		t.Fatalf("Call() dry run should not have response")
	}
	if callCount.Load() != 0 {
		t.Fatalf("server call count = %d, want 0", callCount.Load())
	}
}

func TestCallParsesErrCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errcode":40001,"errmsg":"failed","data":null}`))
	}))
	defer server.Close()

	c := New(&http.Client{Timeout: 2 * time.Second})
	result, err := c.Call(context.Background(), Request{
		Server:   server.URL,
		App:      "wanyun/JitORM",
		Endpoint: "models/services/ModelSvc/getModelInfo",
		Token:    "jit_pat_token_secret",
		Body:     `{"fullName":"nonexist.Model"}`,
	})
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if result == nil || result.Response == nil {
		t.Fatalf("Call() response is nil")
	}
	if !result.Response.HasErrCode {
		t.Fatalf("HasErrCode = false, want true")
	}
	if result.Response.ErrCode != 40001 {
		t.Fatalf("ErrCode = %d, want 40001", result.Response.ErrCode)
	}
	if !result.Response.IsBusinessError() {
		t.Fatalf("IsBusinessError() = false, want true")
	}
	if got := string(result.Response.RawBody); got == "" {
		t.Fatalf("RawBody is empty")
	}
}
