package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
)

type mockRuntime struct {
	authLoginFn       func(context.Context, AuthLoginInput) (map[string]any, error)
	getCurrUserInfoFn func(context.Context, UserInfoInput) (map[string]any, error)
	authLogoutFn      func(context.Context, string) error
	authListFn        func(context.Context) ([]ProfileSummary, error)
	authUseFn         func(context.Context, string) error
	resolveAppFn      func(context.Context, string, string) (string, error)
	callAPIFn         func(context.Context, APIRequest) (APIResponse, error)
}

func (m mockRuntime) AuthLogin(ctx context.Context, input AuthLoginInput) (map[string]any, error) {
	if m.authLoginFn != nil {
		return m.authLoginFn(ctx, input)
	}
	return map[string]any{}, nil
}

func (m mockRuntime) GetCurrUserInfo(ctx context.Context, input UserInfoInput) (map[string]any, error) {
	if m.getCurrUserInfoFn != nil {
		return m.getCurrUserInfoFn(ctx, input)
	}
	return map[string]any{}, nil
}

func (m mockRuntime) AuthLogout(ctx context.Context, profile string) error {
	if m.authLogoutFn != nil {
		return m.authLogoutFn(ctx, profile)
	}
	return nil
}

func (m mockRuntime) AuthList(ctx context.Context) ([]ProfileSummary, error) {
	if m.authListFn != nil {
		return m.authListFn(ctx)
	}
	return []ProfileSummary{}, nil
}

func (m mockRuntime) AuthUse(ctx context.Context, profile string) error {
	if m.authUseFn != nil {
		return m.authUseFn(ctx, profile)
	}
	return nil
}

func (m mockRuntime) ResolveApp(ctx context.Context, profile string, appOverride string) (string, error) {
	if m.resolveAppFn != nil {
		return m.resolveAppFn(ctx, profile, appOverride)
	}
	return "wanyun/JitAi", nil
}

func (m mockRuntime) CallAPI(ctx context.Context, req APIRequest) (APIResponse, error) {
	if m.callAPIFn != nil {
		return m.callAPIFn(ctx, req)
	}
	return APIResponse{Raw: json.RawMessage(`{"errcode":0}`)}, nil
}

func runCmdForTest(t *testing.T, args []string, stdin string, rt Runtime) (int, string, string) {
	t.Helper()
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	in := bytes.NewBufferString(stdin)

	factory := &Factory{
		IO: IOStreams{
			In:     in,
			Out:    stdout,
			ErrOut: stderr,
		},
		Runtime: rt,
	}
	root := NewRootCmd(factory)
	root.SetArgs(args)
	err := root.Execute()
	code := ExitOK
	if err != nil {
		code = handleRootError(factory, err)
	}
	return code, stdout.String(), stderr.String()
}

func runHelpForTest(t *testing.T, args []string) string {
	t.Helper()

	var out bytes.Buffer
	root := NewRootCmd(NewDefaultFactory())
	root.SetOut(&out)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	return out.String()
}
