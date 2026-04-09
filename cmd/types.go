package cmd

import (
	"context"
	"encoding/json"
	"io"
	"os"
)

const (
	ExitOK           = 0
	ExitBackendError = 1
	ExitCLIError     = 2
)

const (
	unwiredRuntimeErrorKey     = "runtime_unwired"
	unwiredRuntimeErrorMessage = "runtime is not wired, internal implementation is missing"
)

const (
	memberSvcGetCurrUserInfo = "corps/services/MemberSvc/getCurrUserInfo"
	modelSvcGetModelInfo     = "models/services/ModelSvc/getModelInfo"
	modelSvcAISelect         = "models/services/ModelSvc/aiSelect"
)

type GlobalFlags struct {
	Profile string
	App     string
	JQ      string
	Format  string
	DryRun  bool
}

type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

type Factory struct {
	IO      IOStreams
	Runtime Runtime
}

func NewDefaultFactory() *Factory {
	runtime, err := NewAppRuntime()
	if err != nil {
		runtime = NewUnwiredRuntime()
	}
	return &Factory{
		IO: IOStreams{
			In:     os.Stdin,
			Out:    os.Stdout,
			ErrOut: os.Stderr,
		},
		Runtime: runtime,
	}
}

type Runtime interface {
	AuthLogin(ctx context.Context, input AuthLoginInput) (map[string]any, error)
	GetCurrUserInfo(ctx context.Context, input UserInfoInput) (map[string]any, error)
	AuthLogout(ctx context.Context, profile string) error
	AuthRemove(ctx context.Context, profile string) error
	AuthList(ctx context.Context) ([]ProfileSummary, error)
	AuthUse(ctx context.Context, profile string) (string, error)
	ResolveApp(ctx context.Context, profile string, appOverride string) (string, error)
	CallAPI(ctx context.Context, req APIRequest) (APIResponse, error)
}

type AuthLoginInput struct {
	Server  string
	App     string
	Profile string
	Token   string
	DryRun  bool
}

type UserInfoInput struct {
	Profile string
	App     string
	JQ      string
	DryRun  bool
}

type ProfileSummary struct {
	Name       string `json:"name"`
	Server     string `json:"server"`
	DefaultApp string `json:"default_app"`
	Index      int    `json:"index"`
	Current    bool   `json:"current"`
	HasToken   bool   `json:"has_token"`
}

type APIRequest struct {
	Profile  string
	App      string
	Endpoint string
	Method   string
	Body     json.RawMessage
	Format   string
	JQ       string
	DryRun   bool
}

type APIResponse struct {
	Raw        json.RawMessage
	ErrCode    int
	HasErrCode bool
}

type unwiredRuntime struct{}

func NewUnwiredRuntime() Runtime {
	return unwiredRuntime{}
}

func (u unwiredRuntime) AuthLogin(context.Context, AuthLoginInput) (map[string]any, error) {
	return nil, newUnwiredRuntimeError()
}

func (u unwiredRuntime) GetCurrUserInfo(context.Context, UserInfoInput) (map[string]any, error) {
	return nil, newUnwiredRuntimeError()
}

func (u unwiredRuntime) AuthLogout(context.Context, string) error {
	return newUnwiredRuntimeError()
}

func (u unwiredRuntime) AuthRemove(context.Context, string) error {
	return newUnwiredRuntimeError()
}

func (u unwiredRuntime) AuthList(context.Context) ([]ProfileSummary, error) {
	return nil, newUnwiredRuntimeError()
}

func (u unwiredRuntime) AuthUse(context.Context, string) (string, error) {
	return "", newUnwiredRuntimeError()
}

func (u unwiredRuntime) ResolveApp(context.Context, string, string) (string, error) {
	return "", newUnwiredRuntimeError()
}

func (u unwiredRuntime) CallAPI(context.Context, APIRequest) (APIResponse, error) {
	return APIResponse{}, newUnwiredRuntimeError()
}

func newUnwiredRuntimeError() *CLIError {
	return NewCLIError(unwiredRuntimeErrorKey, unwiredRuntimeErrorMessage)
}
