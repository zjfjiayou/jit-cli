package cmd

import (
	"fmt"
	"strings"

	"jit-cli/internal/appinfo"

	"github.com/spf13/cobra"
)

type serviceListItem struct {
	FullName  string   `json:"fullName"`
	Name      string   `json:"name,omitempty"`
	Title     string   `json:"title,omitempty"`
	Type      string   `json:"type,omitempty"`
	Functions []string `json:"functions"`
}

func newServiceCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "列出可调用服务并调用服务函数",
		Long: helpSections(
			helpSection(
				"这组命令是什么",
				"`service` 用来处理可调用的服务元素。这里的服务是指在 appInfo 里声明了 `functionList` 的公开元素。",
				"`service ls` 用来先看目录，`service call` 用来真正执行某个服务函数。",
			),
			helpSection(
				"什么时候使用",
				"当你要调用的是服务函数，而不是模型查询时，优先进入这组命令。",
			),
			helpSection(
				"默认上下文",
				"默认使用当前 profile 的 default_app。",
				"`service ls` 依赖本地 appInfo 缓存；`service call` 会实际请求后端。",
			),
			helpSection(
				"如果它不适合",
				"如果你已经知道完整 endpoint 并且不需要缓存校验，改用 `jit api`。",
				"如果你要处理的是模型定义或模型数据，改用 `jit model ...`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "先看当前 app 暴露了哪些可调用服务",
				Command:     "jit service ls",
			},
			helpExample{
				Description: "直接调用某个服务函数",
				Command:     "jit service call corps.services.MemberSvc getCurrUserInfo",
			},
		),
	}

	cmd.AddCommand(newServiceLsCmd(f, gf))
	cmd.AddCommand(newServiceCallCmd(f, gf))
	return cmd
}

func newServiceLsCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var filter string
	var all bool

	cmd := &cobra.Command{
		Use:   "ls",
		Short: "列出缓存中暴露可调用函数的元素",
		Long: helpSections(
			helpSection(
				"这是什么",
				"从本地 appInfo 缓存中筛出带有 `functionList` 的公开元素，作为可调用服务目录。",
			),
			helpSection(
				"什么时候使用",
				"当你还不知道服务 fullName 或函数名，想先浏览当前 app 暴露了哪些服务能力时使用。",
			),
			helpSection(
				"数据来源",
				"只读取本地 appInfo 缓存。",
				"默认只列当前 app 自身元素；传 `--all` 时会把 `extendApps` 中集成进来的服务也带上。",
				"这个列表不保证覆盖后端运行时全部继承链，只代表缓存里能看到的公开服务。",
			),
			helpSection(
				"输出说明",
				"输出 `services` 数组，每项包含服务 `fullName`、标题、类型，以及可见的函数名列表。",
			),
			helpSection(
				"如果它不适合",
				"如果你已经明确知道要调用的服务和函数，直接用 `jit service call <fullName> <functionName>`。",
				"如果缓存还不存在，先执行 `jit app refresh`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "列出当前 app 自身公开的服务",
				Command:     "jit service ls",
			},
			helpExample{
				Description: "包含 extendApps 并按关键字过滤服务",
				Command:     `jit service ls --all --filter member`,
			},
		),
		RunE: func(cmd *cobra.Command, _ []string) error {
			cached, elements, err := loadCachedElements(gf.Profile, gf.App, all)
			if err != nil {
				return err
			}

			keyword := strings.ToLower(strings.TrimSpace(filter))
			items := make([]serviceListItem, 0)
			for _, element := range elements {
				if !isServiceElement(element) || !matchesFilter(element, keyword) {
					continue
				}
				items = append(items, serviceListItem{
					FullName:  element.FullName,
					Name:      element.Name,
					Title:     element.Title,
					Type:      element.Type,
					Functions: functionNames(element),
				})
			}

			return writeValue(f.IO.Out, map[string]any{
				"appId":    cached.App.AppID,
				"services": items,
			}, gf.JQ)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "包含 extendApps 中集成的服务")
	cmd.Flags().StringVar(&filter, "filter", "", "按 fullName/title 做不区分大小写的关键字过滤")
	return cmd
}

func newServiceCallCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var dataArg string

	cmd := &cobra.Command{
		Use:   "call <fullName> <functionName>",
		Short: "通过标准 API 路径调用服务函数",
		Long: helpSections(
			helpSection(
				"这是什么",
				"按标准 API 路径调用某个服务元素上的函数，等价于把 `fullName` 和 `functionName` 组合成 endpoint 再发起请求。",
			),
			helpSection(
				"什么时候使用",
				"当你已经知道服务 fullName 和函数名，并且想直接执行它时使用。",
			),
			helpSection(
				"默认上下文",
				"默认使用当前 profile 的 server、PAT 和 default_app。",
				"传 `--app` 时会把请求发到指定 app。",
			),
			helpSection(
				"输入说明",
				"第一个位置参数是服务元素 fullName，例如 `corps.services.MemberSvc`。",
				"第二个位置参数是函数名，例如 `getCurrUserInfo`。",
				"`--data` 传入 JSON 请求体；传 `@-` 时从 stdin 读取 JSON。",
			),
			helpSection(
				"校验行为",
				"如果本地缓存存在并命中了该元素，会先校验函数名是否出现在缓存的 `functionList` 中。",
				"如果缓存不存在，或元素不在缓存里，则跳过本地校验，直接把请求交给后端决定。",
			),
			helpSection(
				"输出说明",
				"输出后端返回的原始 JSON，不做二次包装。",
			),
			helpSection(
				"如果它不适合",
				"如果你还不知道可用服务，先用 `jit service ls`。",
				"如果你已经知道完整 endpoint，且不想受缓存校验影响，改用 `jit api`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "调用当前 app 下的当前用户服务函数",
				Command:     "jit service call corps.services.MemberSvc getCurrUserInfo",
			},
			helpExample{
				Description: "带 JSON 请求体调用服务函数",
				Command:     `jit service call corps.services.MemberSvc searchMember --data '{"keyword":"Alice"}'`,
			},
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := parseJSONArg(dataArg, f.IO.In)
			if err != nil {
				return NewCLIError("invalid_data", err.Error())
			}

			if err := validateServiceExec(gf.Profile, gf.App, args[0], args[1]); err != nil {
				return err
			}

			return runAPIRequest(cmd, f, APIRequest{
				Profile:  gf.Profile,
				App:      gf.App,
				Endpoint: elementFunctionEndpoint(args[0], args[1]),
				Method:   "POST",
				Body:     body,
				Format:   gf.Format,
				JQ:       gf.JQ,
				DryRun:   gf.DryRun,
			})
		},
	}

	cmd.Flags().StringVar(&dataArg, "data", "{}", "请求体 JSON；传 @- 表示从 stdin 读取")
	return cmd
}

func validateServiceExec(profileName, appOverride, fullName, functionName string) error {
	_, cached, err := loadCachedAppInfo(profileName, appOverride)
	if err != nil {
		// no cache → skip validation, let backend decide
		return nil
	}

	element, ok := findCachedElement(&cached.App, fullName)
	if !ok {
		// element not in cache (may be inherited server-side) → skip validation
		return nil
	}
	if !hasFunction(element, functionName) {
		return NewCLIError(
			"function_not_found",
			fmt.Sprintf("function %q not found on element %q", functionName, fullName),
		)
	}
	return nil
}

func isServiceElement(element appinfo.ElementDefine) bool {
	return len(element.FunctionList) > 0
}

func functionNames(element appinfo.ElementDefine) []string {
	names := make([]string, 0, len(element.FunctionList))
	for _, item := range element.FunctionList {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	return names
}

func hasFunction(element appinfo.ElementDefine, functionName string) bool {
	target := strings.TrimSpace(functionName)
	for _, item := range element.FunctionList {
		if strings.TrimSpace(item.Name) == target {
			return true
		}
	}
	return false
}

func matchesFilter(element appinfo.ElementDefine, keyword string) bool {
	if keyword == "" {
		return true
	}
	return strings.Contains(strings.ToLower(element.FullName), keyword) ||
		strings.Contains(strings.ToLower(element.Title), keyword)
}

func elementFunctionEndpoint(fullName string, functionName string) string {
	return fmt.Sprintf("%s/%s", modelPath(fullName), strings.TrimSpace(functionName))
}

func findCachedElement(info *appinfo.AppInfo, fullName string) (appinfo.ElementDefine, bool) {
	target := strings.TrimSpace(fullName)
	var walk func(*appinfo.AppInfo) (appinfo.ElementDefine, bool)
	walk = func(app *appinfo.AppInfo) (appinfo.ElementDefine, bool) {
		if app == nil {
			return appinfo.ElementDefine{}, false
		}
		for key, el := range app.Elements {
			name := el.FullName
			if name == "" {
				name = key
			}
			if name == target && !appinfo.IsPrivate(el.AccessModifier) {
				return el, true
			}
		}
		for i := range app.ExtendApps {
			if el, ok := walk(&app.ExtendApps[i]); ok {
				return el, true
			}
		}
		return appinfo.ElementDefine{}, false
	}
	return walk(info)
}
