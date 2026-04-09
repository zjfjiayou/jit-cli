package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"jit-cli/internal/appinfo"

	"github.com/spf13/cobra"
)

func newModelCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "模型浏览与数据查询快捷命令",
		Long: helpSections(
			helpSection(
				"这组命令是什么",
				"`model` 用来处理数据模型元素。这里的模型通常是形如 `models.Customer` 的业务数据定义。",
				"`model ls` 用缓存找模型目录，`model get` 查实时模型定义，`model query` 查模型数据，`model tql` 则走 TQL 查询入口。",
			),
			helpSection(
				"什么时候使用",
				"当你已经知道自己要处理的是数据模型，而不是服务函数时，优先进入这组命令。",
			),
			helpSection(
				"默认上下文",
				"默认使用当前 profile 的 default_app。",
				"`model ls` 依赖本地 appInfo 缓存；其它命令会实际请求后端。",
			),
			helpSection(
				"如果它不适合",
				"如果你要调用的是服务函数，改用 `jit service call`。",
				"如果你已经知道完整 endpoint，改用 `jit api`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "先看当前 app 里有哪些公开模型",
				Command:     "jit model ls",
			},
			helpExample{
				Description: "查看某个模型的完整定义，再决定后续查询方式",
				Command:     "jit model get models.Customer",
			},
		),
	}

	cmd.AddCommand(newModelLsCmd(f, gf))
	cmd.AddCommand(newModelGetCmd(f, gf))
	cmd.AddCommand(newModelTQLCmd(f, gf))
	cmd.AddCommand(newModelQueryCmd(f, gf))
	return cmd
}

func newModelLsCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "ls",
		Short: "列出 appInfo 缓存中的模型",
		Long: helpSections(
			helpSection(
				"这是什么",
				"从本地 appInfo 缓存中筛出被识别为模型的元素目录，帮助你先拿到可用的 `fullName`。",
			),
			helpSection(
				"什么时候使用",
				"当你还不知道模型 fullName，或者想先看当前 app 暴露了哪些公开模型时使用。",
			),
			helpSection(
				"数据来源",
				"只读取本地 appInfo 缓存，不直接请求 ModelSvc。",
				"默认只看当前 app 自身元素；传 `--all` 时会把 `extendApps` 中集成进来的模型也带上。",
			),
			helpSection(
				"输出说明",
				"输出 `data` 数组，每项包含模型元素的 `fullName`、`title`、`type` 和部分 `meta`。",
			),
			helpSection(
				"如果它不适合",
				"如果你要查看字段详情，改用 `jit model get <fullName>`。",
				"如果缓存还不存在，先执行 `jit app refresh`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "列出当前 app 自身公开的模型",
				Command:     "jit model ls",
			},
			helpExample{
				Description: "连同集成进来的模型一起列出",
				Command:     "jit model ls --all",
			},
		),
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, elements, err := loadCachedElements(gf.Profile, gf.App, all)
			if err != nil {
				return err
			}

			items := make([]elementSummary, 0, len(elements))
			for _, element := range elements {
				if !isModelElement(element) {
					continue
				}
				items = append(items, summarizeElement(element))
			}

			return writeValue(f.IO.Out, map[string]any{"data": items}, gf.JQ)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "包含 extendApps 中集成的模型")
	return cmd
}

func newModelGetCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "get <fullName>",
		Short: "显示指定 fullName 的模型定义",
		Long: helpSections(
			helpSection(
				"这是什么",
				"根据模型 `fullName` 调用 `ModelSvc/getModelInfo`，获取以后端实时定义为准的模型结构。",
			),
			helpSection(
				"什么时候使用",
				"当你已经拿到模型 fullName，想确认字段列表、字段类型、标题或其它模型元信息时使用。",
			),
			helpSection(
				"输入说明",
				"位置参数必须是完整模型名，例如 `models.Customer`。",
			),
			helpSection(
				"数据来源",
				"数据来自后端 `ModelSvc/getModelInfo`，不是本地缓存。",
			),
			helpSection(
				"输出说明",
				"输出后端返回的模型定义 JSON，通常会包含字段列表和模型级元信息。",
			),
			helpSection(
				"如果它不适合",
				"如果你只是想先浏览有哪些模型，改用 `jit model ls`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "查看 Customer 模型的完整定义和字段信息",
				Command:     "jit model get models.Customer",
			},
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, _ := json.Marshal(map[string]any{"fullName": args[0]})
			return runModelSvcCall(cmd, f, gf, modelSvcGetModelInfo, body)
		},
	}
}

func newModelTQLCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var limit int
	var offset int
	cmd := &cobra.Command{
		Use:   "tql <expr>",
		Short: "通过 ModelSvc/aiSelect 执行 TQL 查询",
		Long: helpSections(
			helpSection(
				"这是什么",
				"把一段 TQL 表达式发送到 `ModelSvc/aiSelect`，用于执行面向模型选择的查询。",
			),
			helpSection(
				"什么时候使用",
				"当你已经明确要用 TQL 表达式，而不是按某个具体模型执行标准 `query` 时使用。",
			),
			helpSection(
				"输入说明",
				"位置参数是完整的 TQL 表达式字符串。",
				"`--limit` 控制返回条数上限，`--offset` 控制偏移量。",
			),
			helpSection(
				"数据来源",
				"数据来自后端 `ModelSvc/aiSelect`。",
			),
			helpSection(
				"输出说明",
				"输出后端返回的原始 JSON，不做二次包装。",
			),
			helpSection(
				"如果它不适合",
				"如果你已经知道目标模型 fullName，并且只想查这个模型的数据，改用 `jit model query <fullName>`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "执行一条 TQL，并限制返回 20 条",
				Command:     `jit model tql 'select models.Customer' --limit 20`,
			},
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, _ := json.Marshal(map[string]any{
				"tql":    args[0],
				"limit":  limit,
				"offset": offset,
			})
			return runModelSvcCall(cmd, f, gf, modelSvcAISelect, body)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "查询条数上限")
	cmd.Flags().IntVar(&offset, "offset", 0, "查询偏移量")
	return cmd
}

func newModelQueryCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var filterArg string
	var page int
	var size int
	cmd := &cobra.Command{
		Use:   "query <fullName>",
		Short: "查询模型数据",
		Long: helpSections(
			helpSection(
				"这是什么",
				"按模型类方法协议调用 `<modelPath>/query`，查询某个具体模型的业务数据。",
			),
			helpSection(
				"什么时候使用",
				"当你已经知道模型 fullName，并且想用标准分页查询拿数据时使用。",
			),
			helpSection(
				"输入说明",
				"位置参数是模型 fullName，例如 `models.Customer`。",
				"`--filter` 传的是 Q 表达式字符串；省略时会按空过滤查询。",
				"`--page` 和 `--size` 分别控制页码与每页条数。",
			),
			helpSection(
				"数据来源",
				"会实际请求目标模型的 `query` 接口，请求体中包含 `methodType=cls` 和 `argDict`。",
			),
			helpSection(
				"输出说明",
				"输出后端返回的原始 JSON，不做二次包装。",
			),
			helpSection(
				"如果它不适合",
				"如果你要查看模型结构而不是模型数据，改用 `jit model get`。",
				"如果你需要的是 TQL 能力，改用 `jit model tql`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "查询 Customer 模型第一页数据",
				Command:     "jit model query models.Customer",
			},
			helpExample{
				Description: "带 Q 表达式过滤查询 Customer 数据",
				Command:     `jit model query models.Customer --filter 'Q("name", "=", "Alice")' --page 1 --size 20`,
			},
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filter := strings.TrimSpace(filterArg)
			var filterValue any
			if filter != "" && !strings.EqualFold(filter, "null") {
				filterValue = filter
			}

			body, _ := json.Marshal(map[string]any{
				"methodType": "cls",
				"argDict": map[string]any{
					"filter":    filterValue,
					"fieldList": nil,
					"orderList": nil,
					"page":      page,
					"size":      size,
				},
			})
			endpoint := fmt.Sprintf("%s/query", modelPath(args[0]))
			return runModelSvcCall(cmd, f, gf, endpoint, body)
		},
	}
	cmd.Flags().StringVar(&filterArg, "filter", "", `查询过滤条件 Q 表达式字符串，例如 Q("name", "=", "Alice")`)
	cmd.Flags().IntVar(&page, "page", 1, "页码")
	cmd.Flags().IntVar(&size, "size", 10, "每页条数")
	return cmd
}

func runModelSvcCall(cmd *cobra.Command, f *Factory, gf *GlobalFlags, endpoint string, body json.RawMessage) error {
	targetApp, err := f.Runtime.ResolveApp(cmd.Context(), gf.Profile, gf.App)
	if err != nil {
		return err
	}
	return runAPIRequest(cmd, f, APIRequest{
		Profile:  gf.Profile,
		App:      targetApp,
		Endpoint: endpoint,
		Method:   "POST",
		Body:     body,
		Format:   gf.Format,
		JQ:       gf.JQ,
		DryRun:   gf.DryRun,
	})
}

func modelPath(fullName string) string {
	return strings.ReplaceAll(strings.TrimSpace(fullName), ".", "/")
}

func isModelElement(element appinfo.ElementDefine) bool {
	if hasModelNamespace(element.Type) {
		return true
	}
	if len(element.FieldList) > 0 {
		return true
	}
	modelType, _ := element.Meta["modelType"].(string)
	return strings.TrimSpace(modelType) != ""
}

func hasModelNamespace(value string) bool {
	clean := strings.TrimSpace(value)
	return strings.HasPrefix(clean, "models.") || strings.Contains(clean, ".models.")
}
