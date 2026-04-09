package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"jit-cli/internal/client"
	"jit-cli/internal/config"
	"jit-cli/internal/output"
	"jit-cli/internal/profile"
)

type appRuntime struct {
	configSvc *config.Service
	profiles  *profile.Manager
	client    *client.Client
}

func NewAppRuntime() (Runtime, error) {
	configSvc, err := config.NewService("")
	if err != nil {
		return nil, err
	}
	profiles, err := profile.NewManager(configSvc.HomeDir())
	if err != nil {
		return nil, err
	}
	return &appRuntime{
		configSvc: configSvc,
		profiles:  profiles,
		client:    client.New(nil),
	}, nil
}

func (r *appRuntime) AuthLogin(ctx context.Context, input AuthLoginInput) (map[string]any, error) {
	server, err := profile.NormalizeServer(input.Server)
	if err != nil {
		return nil, NewCLIError("invalid_server", err.Error())
	}
	if _, _, err := profile.ParseApp(input.App); err != nil {
		return nil, NewCLIError("invalid_app", err.Error())
	}
	if err := profile.ValidatePAT(strings.TrimSpace(input.Token)); err != nil {
		return nil, NewCLIError("invalid_pat", err.Error())
	}

	profileName := strings.TrimSpace(input.Profile)
	if profileName == "" {
		profileName, err = profile.ProfileNameFromServer(server)
		if err != nil {
			return nil, NewCLIError("profile_name_failed", err.Error())
		}
	}

	resp, err := r.callCurrentUser(ctx, server, strings.TrimSpace(input.Token), input.App, input.DryRun)
	if err != nil {
		return nil, err
	}
	if input.DryRun {
		return rawResponseMap(resp, "")
	}
	if resp.Response.IsBusinessError() {
		return nil, NewCLIError("auth_failed", "PAT is invalid or expired")
	}

	if err := r.profiles.SaveProfile(profileName, profile.Config{Server: server, DefaultApp: input.App}); err != nil {
		return nil, NewCLIError("save_profile_failed", err.Error())
	}
	if err := r.profiles.SaveToken(profileName, strings.TrimSpace(input.Token)); err != nil {
		return nil, NewCLIError("save_token_failed", err.Error())
	}
	if err := r.setCurrentProfile(profileName); err != nil {
		return nil, err
	}

	payload, ok := resp.Response.JSON.(map[string]any)
	if !ok {
		return nil, NewCLIError("invalid_response", "expected object response from auth status")
	}
	return payload, nil
}

func (r *appRuntime) GetCurrUserInfo(ctx context.Context, input UserInfoInput) (map[string]any, error) {
	profileName, cfg, token, err := r.resolveProfile(input.Profile)
	if err != nil {
		return nil, err
	}

	baseApp, err := resolveAppRef(input.App, cfg.DefaultApp)
	if err != nil {
		return nil, err
	}

	resp, err := r.callCurrentUser(ctx, cfg.Server, token, baseApp, input.DryRun)
	if err != nil {
		return nil, err
	}

	responseMap, err := rawResponseMap(resp, input.JQ)
	if err != nil {
		return nil, err
	}
	responseMap["profile"] = profileName
	if resp.Response != nil && resp.Response.IsBusinessError() {
		return responseMap, NewCLIError("auth_failed", "PAT is invalid or expired")
	}
	return responseMap, nil
}

func (r *appRuntime) AuthLogout(ctx context.Context, profileName string) error {
	name, err := r.resolveSelectedProfile(profileName, "profile is required")
	if err != nil {
		return err
	}
	if err := r.profiles.RemoveToken(name); err != nil {
		return NewCLIError("logout_failed", err.Error())
	}
	return nil
}

func (r *appRuntime) AuthList(ctx context.Context) ([]ProfileSummary, error) {
	cfg, err := r.loadGlobalConfig()
	if err != nil {
		return nil, err
	}
	items, err := r.profiles.Summaries(cfg.CurrentProfile)
	if err != nil {
		return nil, NewCLIError("list_profiles_failed", err.Error())
	}

	result := make([]ProfileSummary, 0, len(items))
	for _, item := range items {
		result = append(result, ProfileSummary{
			Name:       item.Name,
			Server:     item.Server,
			DefaultApp: item.DefaultApp,
			Current:    item.Current,
			HasToken:   item.HasToken,
		})
	}
	return result, nil
}

func (r *appRuntime) AuthUse(ctx context.Context, profileName string) error {
	if !r.profiles.Exists(profileName) {
		return NewCLIError("profile_not_found", fmt.Sprintf("profile %q does not exist", profileName))
	}
	return r.setCurrentProfile(profileName)
}

func (r *appRuntime) ResolveApp(ctx context.Context, profileName string, appOverride string) (string, error) {
	_, cfg, _, err := r.resolveProfile(profileName)
	if err != nil {
		return "", err
	}
	return resolveAppRef(appOverride, cfg.DefaultApp)
}

func (r *appRuntime) CallAPI(ctx context.Context, req APIRequest) (APIResponse, error) {
	_, cfg, token, err := r.resolveProfile(req.Profile)
	if err != nil {
		return APIResponse{}, err
	}

	appRef, err := resolveAppRef(req.App, cfg.DefaultApp)
	if err != nil {
		return APIResponse{}, err
	}

	resp, err := r.callAppRequest(ctx, cfg.Server, token, appRef, req.Endpoint, req.Method, req.Body, req.DryRun)
	if err != nil {
		return APIResponse{}, err
	}

	raw, err := resultRaw(resp, req.JQ)
	if err != nil {
		return APIResponse{}, NewCLIError("output_failed", err.Error())
	}

	apiResp := APIResponse{Raw: raw}
	if resp.Response != nil {
		apiResp.ErrCode = resp.Response.ErrCode
		apiResp.HasErrCode = resp.Response.HasErrCode
	}
	return apiResp, nil
}

func (r *appRuntime) resolveProfile(profileName string) (string, profile.Config, string, error) {
	var cfg profile.Config
	name, err := r.resolveSelectedProfile(profileName, "profile is required, pass --profile or run `jit auth use <profile>`")
	if err != nil {
		return "", cfg, "", err
	}

	cfg, err = r.profiles.LoadProfile(name)
	if err != nil {
		return "", cfg, "", NewCLIError("profile_not_found", err.Error())
	}
	token, err := r.profiles.LoadToken(name)
	if err != nil {
		return "", cfg, "", NewCLIError("token_load_failed", err.Error())
	}
	if strings.TrimSpace(token) == "" {
		return "", cfg, "", NewCLIError("missing_token", fmt.Sprintf("profile %q does not have a PAT", name))
	}
	return name, cfg, token, nil
}

func (r *appRuntime) loadGlobalConfig() (config.GlobalConfig, error) {
	cfg, err := r.configSvc.Load()
	if err != nil {
		return config.GlobalConfig{}, NewCLIError("load_config_failed", err.Error())
	}
	return cfg, nil
}

func (r *appRuntime) setCurrentProfile(profileName string) error {
	cfg, err := r.loadGlobalConfig()
	if err != nil {
		return err
	}
	cfg.CurrentProfile = profileName
	if err := r.configSvc.Save(cfg); err != nil {
		return NewCLIError("save_config_failed", err.Error())
	}
	return nil
}

func (r *appRuntime) resolveSelectedProfile(profileName string, missingMessage string) (string, error) {
	name := strings.TrimSpace(profileName)
	if name != "" {
		return name, nil
	}
	cfg, err := r.loadGlobalConfig()
	if err != nil {
		return "", err
	}
	name = strings.TrimSpace(cfg.CurrentProfile)
	if name == "" {
		return "", NewCLIError("missing_profile", missingMessage)
	}
	return name, nil
}

func resolveAppRef(appOverride string, defaultApp string) (string, error) {
	appRef := strings.TrimSpace(appOverride)
	if appRef == "" {
		appRef = strings.TrimSpace(defaultApp)
	}
	if appRef == "" {
		return "", NewCLIError("missing_app", "profile does not have default_app, pass --app <org/app>")
	}
	if _, _, err := profile.ParseApp(appRef); err != nil {
		return "", NewCLIError("invalid_app", err.Error())
	}
	return appRef, nil
}

func (r *appRuntime) callCurrentUser(ctx context.Context, server string, token string, baseApp string, dryRun bool) (*client.Result, error) {
	resp, err := r.callAppRequest(ctx, server, token, baseApp, memberSvcGetCurrUserInfo, http.MethodPost, map[string]any{}, dryRun)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (r *appRuntime) callAppRequest(
	ctx context.Context,
	server string,
	token string,
	appRef string,
	endpoint string,
	method string,
	body any,
	dryRun bool,
) (*client.Result, error) {
	resp, err := r.client.Call(ctx, client.Request{
		Server:   server,
		App:      appRef,
		Endpoint: endpoint,
		Token:    token,
		Method:   method,
		Body:     body,
		DryRun:   dryRun,
	})
	if err != nil {
		return nil, NewCLIError("request_failed", err.Error())
	}
	return resp, nil
}

func rawResponseMap(result *client.Result, jq string) (map[string]any, error) {
	raw, err := resultRaw(result, jq)
	if err != nil {
		return nil, NewCLIError("output_failed", err.Error())
	}
	return map[string]any{"raw": raw}, nil
}

func resultRaw(result *client.Result, jq string) (json.RawMessage, error) {
	if result == nil {
		return json.RawMessage(`null`), nil
	}
	if result.DryRun != nil {
		return json.Marshal(result.DryRun)
	}
	if result.Response == nil {
		return json.RawMessage(`null`), nil
	}
	if strings.TrimSpace(jq) == "" {
		return append(json.RawMessage(nil), result.Response.RawBody...), nil
	}
	var buf bytes.Buffer
	if err := output.WriteJQ(&buf, result.Response.JSON, jq, false); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}
