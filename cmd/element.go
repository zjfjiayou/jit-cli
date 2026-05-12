package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

func newElementCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "element",
		Short: "元素源码读取、保存、构建与检索",
		Long: helpSections(
			helpSection(
				"这组命令是什么",
				"`element` 用来处理 IDEAppBackend 暴露的开发态元素源码能力。",
				"它面向需要读取元素源码、保存元素资源、触发构建或检索源码内容的 Agent 和脚本。",
			),
			helpSection(
				"什么时候使用",
				"当你要修改 JIT 元素源码，而不是查询业务模型数据或调用普通业务服务时，优先进入这组命令。",
			),
			helpSection(
				"默认上下文",
				"默认使用当前 profile 的 default_app。",
				"通常应把 `--app` 指向部署了 `services.ElementSvc` 的开发态应用，例如 IDEApp。",
			),
			helpSection(
				"安全边界",
				"这组命令只保留元素源码闭环中的稳定入口，不暴露裸删除、重命名和批量替换等高风险文件操作。",
			),
			helpSection(
				"如果它不适合",
				"如果你要查询业务模型数据，改用 `jit model ...`。",
				"如果你已经知道完整 endpoint，改用 `jit api`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "读取一个元素的源码资源",
				Command:     "jit element get services.ElementSvc",
			},
			helpExample{
				Description: "保存一个元素资源文件并触发增量构建",
				Command:     `jit element save services.DemoSvc --resources '{"service.py":"print(\"hi\")"}'`,
			},
		),
	}

	cmd.AddCommand(newElementLsCmd(f, gf))
	cmd.AddCommand(newElementGetCmd(f, gf))
	cmd.AddCommand(newElementSaveCmd(f, gf))
	cmd.AddCommand(newElementApplyCmd(f, gf))
	cmd.AddCommand(newElementBuildCmd(f, gf))
	cmd.AddCommand(newElementSearchCmd(f, gf))
	cmd.AddCommand(newElementKnowledgeCmd(f, gf))
	return cmd
}

func newElementLsCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "列出当前应用的元素树",
		Long: helpSections(
			helpSection(
				"这是什么",
				"调用 `ElementSvc/getElementTree`，返回当前应用及继承应用的元素树。",
			),
			helpSection(
				"什么时候使用",
				"当你还不知道元素 fullName，或者准备继续读取、保存、构建某个元素前使用。",
			),
			helpSection(
				"数据来源",
				"数据来自后端运行时的元素定义和源码目录，不读取本地 appInfo 缓存。",
			),
			helpSection(
				"输出说明",
				"输出后端返回的原始 JSON，通常包含 `appId`、`elements` 和 `extends`。",
			),
			helpSection(
				"如果它不适合",
				"如果你只想浏览 appInfo 缓存中的公开元素，改用 `jit app ls`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "列出当前开发态应用的元素树",
				Command:     "jit element ls",
			},
		),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runElementSvcCall(cmd, f, gf, elementSvcGetTree, map[string]any{})
		},
	}
}

func newElementGetCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var ignoreArg string
	var resourcesArg string

	cmd := &cobra.Command{
		Use:   "get <fullName>",
		Short: "读取元素源码资源",
		Long: helpSections(
			helpSection(
				"这是什么",
				"调用 `ElementSvc/getElementResource`，按元素 fullName 读取源码文件内容。",
			),
			helpSection(
				"什么时候使用",
				"当你准备检查或修改某个元素源码时，先用这个命令拿到当前资源内容。",
			),
			helpSection(
				"输入说明",
				"位置参数是元素 fullName，例如 `services.ElementSvc`。",
				"`--resources` 是 JSON 数组字符串，用来只读取匹配的资源，例如 `[\"service.py\",\"e.json\"]` 或 `[\"*.py\"]`。",
				"`--ignore` 是 JSON 数组字符串，用来排除匹配文件。",
			),
			helpSection(
				"输出说明",
				"输出后端返回的文件名到文件内容映射。",
			),
			helpSection(
				"如果它不适合",
				"如果你只想看元素定义元数据，优先用 `jit app ls` 或 `jit service ls`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "读取元素的全部可读源码资源",
				Command:     "jit element get services.ElementSvc",
			},
			helpExample{
				Description: "只读取 Python 与声明文件",
				Command:     `jit element get services.ElementSvc --resources '["service.py","e.json"]'`,
			},
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{"fullName": args[0]}
			if err := addOptionalStringList(body, "ignore", ignoreArg, "--ignore"); err != nil {
				return err
			}
			if err := addOptionalStringList(body, "resources", resourcesArg, "--resources"); err != nil {
				return err
			}
			return runElementSvcCall(cmd, f, gf, elementSvcGetResource, body)
		},
	}

	cmd.Flags().StringVar(&ignoreArg, "ignore", "", `忽略资源 JSON 数组，例如 ["dist/*"]`)
	cmd.Flags().StringVar(&resourcesArg, "resources", "", `待读取资源 JSON 数组，例如 ["service.py","e.json"]`)
	return cmd
}

func newElementSaveCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var resourcesArg string

	cmd := &cobra.Command{
		Use:   "save <fullName>",
		Short: "保存元素源码资源并构建元素",
		Long: helpSections(
			helpSection(
				"这是什么",
				"调用 `ElementSvc/saveElementResource`，把文件名到内容的映射写回指定元素，并按后端默认行为构建该元素。",
			),
			helpSection(
				"什么时候使用",
				"当你已经读过元素资源、准备保存少量源码文件修改时使用。",
			),
			helpSection(
				"输入说明",
				"位置参数是元素 fullName。",
				"`--resources` 必须是 JSON 对象，键是相对元素目录的文件名，值是文件内容；传 `@-` 时从 stdin 读取。",
			),
			helpSection(
				"副作用",
				"会写入远端应用源码，并触发该元素构建。",
			),
			helpSection(
				"输出说明",
				"输出后端返回的元素构建结果。",
			),
			helpSection(
				"如果它不适合",
				"如果需要同时保存声明和多个元素资源，改用 `jit element apply`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "保存一个资源文件",
				Command:     `jit element save services.DemoSvc --resources '{"service.py":"print(\"hi\")"}'`,
			},
			helpExample{
				Description: "从 stdin 读取资源对象",
				Command:     `printf '%s' '{"service.py":"print(1)"}' | jit element save services.DemoSvc --resources @-`,
			},
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resources, err := parseRequiredJSONObjectArg(resourcesArg, f.IO.In, "--resources")
			if err != nil {
				return err
			}
			return runElementSvcCall(cmd, f, gf, elementSvcSaveResource, map[string]any{
				"fullName":  args[0],
				"resources": resources,
			})
		},
	}

	cmd.Flags().StringVar(&resourcesArg, "resources", "", "资源 JSON 对象；传 @- 表示从 stdin 读取")
	return cmd
}

func newElementApplyCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var dataArg string

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "批量保存元素声明和源码资源",
		Long: helpSections(
			helpSection(
				"这是什么",
				"调用 `ElementSvc/saveElement`，一次保存多个元素的声明和源码资源，然后由后端统一构建。",
			),
			helpSection(
				"什么时候使用",
				"当一次修改涉及 `e.json` 声明和多个资源文件，或者需要批量保存多个元素时使用。",
			),
			helpSection(
				"输入说明",
				"`--data` 必须是 JSON 数组，每项通常包含 `ePath`、`define` 和 `resources`；传 `@-` 时从 stdin 读取。",
			),
			helpSection(
				"副作用",
				"会写入远端应用源码，并触发涉及元素的构建。",
			),
			helpSection(
				"如果它不适合",
				"如果只保存单个元素的少量资源文件，改用 `jit element save <fullName>`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "从 stdin 批量保存元素",
				Command:     "cat element-list.json | jit element apply --data @-",
			},
		),
		RunE: func(cmd *cobra.Command, _ []string) error {
			items, err := parseRequiredJSONArrayArg(dataArg, f.IO.In, "--data")
			if err != nil {
				return err
			}
			return runElementSvcCall(cmd, f, gf, elementSvcSaveElement, map[string]any{
				"elementList": items,
			})
		},
	}

	cmd.Flags().StringVar(&dataArg, "data", "", "元素列表 JSON 数组；传 @- 表示从 stdin 读取")
	return cmd
}

func newElementBuildCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "build <fullName>...",
		Short: "构建一个或多个元素",
		Long: helpSections(
			helpSection(
				"这是什么",
				"调用 `ElementSvc/buildElement`，构建一个或多个指定 fullName 的元素。",
			),
			helpSection(
				"什么时候使用",
				"当源码已经写入远端，但只想重新构建部分元素时使用。",
			),
			helpSection(
				"输入说明",
				"位置参数是一个或多个元素 fullName。",
			),
			helpSection(
				"副作用",
				"会触发后端 Builder，并刷新入口 App 快照。",
			),
			helpSection(
				"如果它不适合",
				"如果需要全量构建当前应用，改用 `jit app build`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "构建单个元素",
				Command:     "jit element build services.DemoSvc",
			},
			helpExample{
				Description: "一次构建多个元素",
				Command:     "jit element build services.DemoSvc models.Customer",
			},
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return NewCLIError("missing_element", "at least one element fullName is required")
			}
			return runElementSvcCall(cmd, f, gf, elementSvcBuildElement, map[string]any{
				"fullNames": args,
			})
		},
	}
}

func newElementSearchCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "search <pattern>",
		Short: "按正则检索当前应用源码",
		Long: helpSections(
			helpSection(
				"这是什么",
				"调用 `ElementSvc/searchContent`，在当前应用源码文件中按正则表达式 pattern 查找内容。",
			),
			helpSection(
				"什么时候使用",
				"当你需要先定位某个函数、字段、组件或配置出现在哪些源码文件中时使用。",
			),
			helpSection(
				"输入说明",
				"位置参数是正则表达式 pattern。",
			),
			helpSection(
				"输出说明",
				"输出匹配到的相对文件路径列表。",
			),
			helpSection(
				"如果它不适合",
				"如果你已经知道元素 fullName，直接用 `jit element get <fullName>` 读取资源。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "查找源码中出现 ElementSvc 的文件",
				Command:     "jit element search ElementSvc",
			},
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runElementSvcCall(cmd, f, gf, elementSvcSearchContent, map[string]any{
				"pattern": args[0],
			})
		},
	}
}

func newElementKnowledgeCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "knowledge <fullName>",
		Short: "读取元素知识描述",
		Long: helpSections(
			helpSection(
				"这是什么",
				"调用 `ElementSvc/getElementKnowledge`，读取元素暴露给 Agent 的知识描述。",
				"后端优先调用元素的 `knowledges()`，否则读取元素定义里的 `description`。",
			),
			helpSection(
				"什么时候使用",
				"当你需要先了解元素职责、能力或使用约束，再决定是否读取源码或调用接口时使用。",
			),
			helpSection(
				"输入说明",
				"位置参数是元素 fullName。",
			),
			helpSection(
				"输出说明",
				"输出后端返回的知识内容，可能是文本，也可能是 JSON 字符串。",
			),
			helpSection(
				"如果它不适合",
				"如果你要读取真实源码文件，改用 `jit element get <fullName>`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "读取元素知识描述",
				Command:     "jit element knowledge services.ElementSvc",
			},
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runElementSvcCall(cmd, f, gf, elementSvcGetKnowledge, map[string]any{
				"elementFullName": args[0],
			})
		},
	}
}

func runElementSvcCall(cmd *cobra.Command, f *Factory, gf *GlobalFlags, endpoint string, payload map[string]any) error {
	return runAPIRequest(cmd, f, APIRequest{
		Profile:  gf.Profile,
		App:      gf.App,
		Endpoint: endpoint,
		Method:   "POST",
		Body:     mustJSONBody(payload),
		Format:   gf.Format,
		JQ:       gf.JQ,
		DryRun:   gf.DryRun,
	})
}

func addOptionalStringList(body map[string]any, key string, dataArg string, flagName string) error {
	items, err := parseOptionalStringListArg(dataArg, flagName)
	if err != nil {
		return err
	}
	if items != nil {
		body[key] = items
	}
	return nil
}

func parseOptionalStringListArg(dataArg string, flagName string) ([]string, error) {
	raw := strings.TrimSpace(dataArg)
	if raw == "" || strings.EqualFold(raw, "null") {
		return nil, nil
	}

	var payload []string
	if err := json.Unmarshal([]byte(raw), &payload); err != nil || payload == nil {
		key := "invalid_" + strings.TrimPrefix(flagName, "--")
		return nil, NewCLIError(key, fmt.Sprintf("%s must be a JSON string array", flagName))
	}
	return payload, nil
}

func parseRequiredJSONArrayArg(dataArg string, in io.Reader, flagName string) ([]any, error) {
	if strings.TrimSpace(dataArg) == "" {
		return nil, NewCLIError("missing_data", fmt.Sprintf("%s is required", flagName))
	}

	raw, err := parseJSONArg(dataArg, in)
	if err != nil {
		return nil, NewCLIError("invalid_data", err.Error())
	}

	var payload []any
	if err := json.Unmarshal(raw, &payload); err != nil || payload == nil {
		return nil, NewCLIError("invalid_data", fmt.Sprintf("%s must be a JSON array", flagName))
	}
	return payload, nil
}

func mustJSONBody(value any) json.RawMessage {
	body, _ := json.Marshal(value)
	return body
}
