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
		Short: "列出可调用服务并执行缓存函数",
	}

	cmd.AddCommand(newServiceListCmd(f, gf))
	cmd.AddCommand(newServiceExecCmd(f, gf))
	return cmd
}

func newServiceListCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var filter string
	var all bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出缓存中暴露可调用函数的元素",
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

func newServiceExecCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var dataArg string

	cmd := &cobra.Command{
		Use:   "exec <fullName> <functionName>",
		Short: "通过标准 API 路径执行服务函数",
		Args:  cobra.ExactArgs(2),
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
