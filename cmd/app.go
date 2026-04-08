package cmd

import (
	"fmt"
	"strings"

	"jit-cli/internal/appinfo"
	"jit-cli/internal/profile"

	"github.com/spf13/cobra"
)

type elementSummary struct {
	FullName string         `json:"fullName"`
	Name     string         `json:"name,omitempty"`
	Title    string         `json:"title,omitempty"`
	Type     string         `json:"type,omitempty"`
	Meta     map[string]any `json:"meta,omitempty"`
}

func newAppCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app",
		Short: "appInfo 缓存管理与查看",
	}

	cmd.AddCommand(newAppRefreshCmd(f, gf))
	cmd.AddCommand(newAppInfoCmd(f, gf))
	cmd.AddCommand(newAppElementsCmd(f, gf))
	return cmd
}

func newAppRefreshCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "拉取 appInfo.js、解密并刷新本地缓存",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, err := loadProfileContext(gf.Profile, true)
			if err != nil {
				return err
			}

			appRef, err := resolveAppRef(gf.App, ctx.Config.DefaultApp)
			if err != nil {
				return err
			}
			org, appID, err := profile.ParseApp(appRef)
			if err != nil {
				return NewCLIError("invalid_app", err.Error())
			}

			if gf.DryRun {
				preview, err := newAppRefreshDryRun(ctx.Config.Server, org, appID, ctx.Token)
				if err != nil {
					return NewCLIError("output_failed", err.Error())
				}
				return writeValue(f.IO.Out, preview, gf.JQ)
			}

			info, err := appinfo.Fetch(cmd.Context(), ctx.Config.Server, org, appID, ctx.Token)
			if err != nil {
				return NewCLIError("fetch_appinfo_failed", err.Error())
			}
			if err := appinfo.Save(ctx.Profiles.AppInfoPath(ctx.Name), info); err != nil {
				return NewCLIError("save_appinfo_failed", err.Error())
			}

			elements := appinfo.Elements(info)
			payload := map[string]any{
				"ok":       true,
				"appId":    info.AppID,
				"elements": len(elements),
			}
			return writeValue(f.IO.Out, payload, gf.JQ)
		},
	}
}

func newAppInfoCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "显示缓存中的应用基础信息",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, cached, err := loadCachedAppInfo(gf.Profile, gf.App)
			if err != nil {
				return err
			}

			payload := map[string]any{
				"fetchedAt": cached.FetchedAt,
				"app": map[string]any{
					"name":    cached.App.Name,
					"title":   cached.App.Title,
					"appId":   cached.App.AppID,
					"version": cached.App.Version,
				},
			}
			return writeValue(f.IO.Out, payload, gf.JQ)
		},
	}
}

func newAppElementsCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "elements",
		Short: "列出缓存中所有非 private 元素",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cached, elements, err := loadCachedElements(gf.Profile, gf.App, true)
			if err != nil {
				return err
			}

			items := make([]elementSummary, 0, len(elements))
			for _, element := range elements {
				items = append(items, summarizeElement(element))
			}

			return writeValue(f.IO.Out, map[string]any{
				"appId":    cached.App.AppID,
				"elements": items,
			}, gf.JQ)
		},
	}
}

func newAppRefreshDryRun(server, org, appID, token string) (map[string]any, error) {
	normalizedServer, err := profile.NormalizeServer(server)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"method": "GET",
		"url":    fmt.Sprintf("%s/%s/%s/appInfo.js", normalizedServer, org, appID),
		"headers": map[string]string{
			"Authorization": "Bearer " + strings.TrimSpace(token),
		},
	}, nil
}

func summarizeElement(element appinfo.ElementDefine) elementSummary {
	return elementSummary{
		FullName: element.FullName,
		Name:     element.Name,
		Title:    element.Title,
		Type:     element.Type,
		Meta:     element.Meta,
	}
}
