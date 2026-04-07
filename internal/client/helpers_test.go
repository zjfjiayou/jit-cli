package client

import "testing"

func TestBuildURL(t *testing.T) {
	urlValue, err := BuildURL(
		"https://demo.jit.cn/",
		"wanyun/JitAi",
		"services/JitAISvc/sendMessage",
	)
	if err != nil {
		t.Fatalf("BuildURL() error = %v", err)
	}

	expected := "https://demo.jit.cn/api/wanyun/JitAi/services/JitAISvc/sendMessage"
	if urlValue != expected {
		t.Fatalf("BuildURL() = %q, want %q", urlValue, expected)
	}
}

func TestParseAppRefInvalid(t *testing.T) {
	if _, err := ParseAppRef("wanyun"); err == nil {
		t.Fatalf("ParseAppRef() expected error, got nil")
	}
}

func TestParseEndpointRefInvalid(t *testing.T) {
	if _, err := ParseEndpointRef("onlyFunction"); err == nil {
		t.Fatalf("ParseEndpointRef() expected error, got nil")
	}
}
