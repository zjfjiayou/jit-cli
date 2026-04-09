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
		Short: "应用缓存管理与元素查看",
		Long: helpSections(
			helpSection(
				"这组命令是什么",
				"`app` 这组命令围绕 appInfo 缓存工作。appInfo 是前端暴露的应用元素目录，里面会列出模型、服务和其它元素的基础定义。",
				"`app refresh` 负责把这个目录拉到本地，`app get` 和 `app ls` 则负责读取本地缓存。",
			),
			helpSection(
				"什么时候使用",
				"第一次接入一个 app、切换到另一个 app、或者怀疑模型/服务目录已经变化时，先用这组命令建立或刷新本地视图。",
			),
			helpSection(
				"默认上下文",
				"默认读取当前 profile 的 server、PAT 和 default_app。",
				"除非显式传 `--app`，否则所有 `app` 子命令都针对当前 profile 的 default_app。",
			),
			helpSection(
				"如果它不适合",
				"如果你需要的是实时模型字段定义，改用 `jit model get`。",
				"如果你已经知道目标 endpoint，改用 `jit api` 或 `jit service call`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "先刷新当前 app 的本地元素目录缓存",
				Command:     "jit app refresh",
			},
			helpExample{
				Description: "查看当前 app 对外暴露了哪些元素",
				Command:     "jit app ls",
			},
		),
	}

	cmd.AddCommand(newAppRefreshCmd(f, gf))
	cmd.AddCommand(newAppGetCmd(f, gf))
	cmd.AddCommand(newAppLsCmd(f, gf))
	return cmd
}

func newAppRefreshCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "拉取 appInfo.js、解密并刷新本地缓存",
		Long: helpSections(
			helpSection(
				"这是什么",
				"从目标 app 拉取前端暴露的 `appInfo.js`，完成解密后写入当前 profile 对应的 `appinfo.json`。",
			),
			helpSection(
				"什么时候使用",
				"第一次在本机访问某个 app 时使用。",
				"切换 app、切换 profile、或应用元素定义发生变化后，也应重新执行一次。",
			),
			helpSection(
				"默认上下文",
				"默认使用当前 profile 的 server、PAT 和 default_app。",
				"传 `--app` 时会改为刷新指定 app 的缓存。",
			),
			helpSection(
				"副作用",
				"会覆盖本地已有的 appInfo 缓存。",
				"后续 `jit app ls`、`jit model ls`、`jit service ls` 都会读取这份缓存。",
			),
			helpSection(
				"数据来源",
				"数据直接来自 `/{org}/{app}/appInfo.js`，不是 ModelSvc，也不是其它业务接口。",
			),
			helpSection(
				"输出说明",
				"成功时输出 `{ok:true, appId:<...>, elements:<数量>}`。",
				"开启 `--dry-run` 时只输出将要访问的 URL 和请求头，不真正拉取。",
			),
			helpSection(
				"如果它不适合",
				"如果你只是想读取已经存在的缓存，不要重复刷新，改用 `jit app get` 或 `jit app ls`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "刷新当前 profile 默认 app 的缓存",
				Command:     "jit app refresh",
			},
			helpExample{
				Description: "先预览将要请求的 appInfo 地址，不实际发请求",
				Command:     "jit app refresh --dry-run",
			},
		),
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

func newAppGetCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "显示缓存中的应用基础信息",
		Long: helpSections(
			helpSection(
				"这是什么",
				"读取本地 appInfo 缓存的顶层信息，确认当前缓存对应的是哪个 app、哪个版本、是什么时候抓取的。",
			),
			helpSection(
				"什么时候使用",
				"当你不确定当前 profile 缓存的是哪个 app，或者想确认缓存是否已经刷新成功时使用。",
			),
			helpSection(
				"数据来源",
				"只读取本地 `appinfo.json`，不请求后端。",
			),
			helpSection(
				"输出说明",
				"输出 `fetchedAt` 和 `app` 对象，其中 `app` 包含 `name`、`title`、`appId`、`version`。",
			),
			helpSection(
				"如果它不适合",
				"如果你想列出元素目录，改用 `jit app ls`；如果缓存还不存在，先执行 `jit app refresh`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "查看当前 app 缓存的基本信息",
				Command:     "jit app get",
			},
		),
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

func newAppLsCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "列出缓存中所有非 private 元素",
		Long: helpSections(
			helpSection(
				"这是什么",
				"列出当前 appInfo 缓存中所有对外暴露的元素目录。",
				"元素既可能是模型，也可能是服务、页面组件或其它前端登记的元素类型。",
			),
			helpSection(
				"什么时候使用",
				"当你还不知道当前 app 里有哪些元素、想先找 fullName、或准备继续用 `model`/`service` 命令时使用。",
			),
			helpSection(
				"数据来源",
				"只读取本地 appInfo 缓存。",
				"结果会包含当前 app 以及 `extendApps` 展开的非 private 元素。",
			),
			helpSection(
				"输出说明",
				"输出 `elements` 数组，每项包含 `fullName`、`name`、`title`、`type` 和部分 `meta`。",
			),
			helpSection(
				"如果它不适合",
				"如果你只关心模型，改用 `jit model ls`；如果你只关心可调用服务，改用 `jit service ls`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "浏览当前 app 暴露的全部公开元素",
				Command:     "jit app ls",
			},
			helpExample{
				Description: "只提取元素 fullName 供后续命令复用",
				Command:     `jit app ls --jq '.elements[].fullName'`,
			},
		),
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
