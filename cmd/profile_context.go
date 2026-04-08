package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"jit-cli/internal/appinfo"
	"jit-cli/internal/config"
	"jit-cli/internal/profile"
)

type profileContext struct {
	Name     string
	Config   profile.Config
	Token    string
	Profiles *profile.Manager
}

func loadProfileContext(profileName string, requireToken bool) (profileContext, error) {
	var ctx profileContext

	configSvc, err := config.NewService("")
	if err != nil {
		return ctx, NewCLIError("load_config_failed", err.Error())
	}
	profiles, err := profile.NewManager(configSvc.HomeDir())
	if err != nil {
		return ctx, NewCLIError("load_config_failed", err.Error())
	}

	name := strings.TrimSpace(profileName)
	if name == "" {
		globalCfg, err := configSvc.Load()
		if err != nil {
			return ctx, NewCLIError("load_config_failed", err.Error())
		}
		name = strings.TrimSpace(globalCfg.CurrentProfile)
	}
	if name == "" {
		return ctx, NewCLIError("missing_profile", "profile is required, pass --profile or run `jit auth use <profile>`")
	}

	cfg, err := profiles.LoadProfile(name)
	if err != nil {
		return ctx, NewCLIError("profile_not_found", err.Error())
	}

	ctx = profileContext{
		Name:     name,
		Config:   cfg,
		Profiles: profiles,
	}
	if !requireToken {
		return ctx, nil
	}

	token, err := profiles.LoadToken(name)
	if err != nil {
		return ctx, NewCLIError("token_load_failed", err.Error())
	}
	if strings.TrimSpace(token) == "" {
		return ctx, NewCLIError("missing_token", fmt.Sprintf("profile %q does not have a PAT", name))
	}
	ctx.Token = token
	return ctx, nil
}

func loadCachedAppInfo(profileName string, appOverride string) (profileContext, *appinfo.CachedAppInfo, error) {
	ctx, err := loadProfileContext(profileName, false)
	if err != nil {
		return ctx, nil, err
	}

	expectedApp, err := resolveAppRef(appOverride, ctx.Config.DefaultApp)
	if err != nil {
		return ctx, nil, err
	}

	cached, err := appinfo.Load(ctx.Profiles.AppInfoPath(ctx.Name))
	if errors.Is(err, os.ErrNotExist) {
		return ctx, nil, NewCLIError("missing_appinfo_cache", "app info cache not found, run `jit app refresh`")
	}
	if err != nil {
		return ctx, nil, NewCLIError("load_appinfo_failed", err.Error())
	}
	if cached.App.AppID != "" && cached.App.AppID != expectedApp {
		return ctx, nil, NewCLIError(
			"stale_appinfo_cache",
			fmt.Sprintf("app info cache is for %q, expected %q, run `jit app refresh`", cached.App.AppID, expectedApp),
		)
	}

	return ctx, cached, nil
}

func loadCachedElements(profileName string, appOverride string, includeExtended bool) (*appinfo.CachedAppInfo, []appinfo.ElementDefine, error) {
	_, cached, err := loadCachedAppInfo(profileName, appOverride)
	if err != nil {
		return nil, nil, err
	}
	if includeExtended {
		return cached, appinfo.Elements(&cached.App), nil
	}
	return cached, appinfo.ElementsLocal(&cached.App), nil
}
