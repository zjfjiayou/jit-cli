package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"jit-cli/internal/appinfo"

	"github.com/spf13/cobra"
)

func newModelCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "模型浏览与原子数据操作命令",
		Long: helpSections(
			helpSection(
				"这组命令是什么",
				"`model` 用来处理数据模型元素。这里的模型通常是形如 `models.Customer` 的业务数据定义。",
				"`model ls` 用缓存找模型目录，`model get` 查实时模型定义，`model query` 查模型明细，`model create/update/delete` 执行原子写操作，`model analyze` 走分析查询入口。",
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
	cmd.AddCommand(newModelQueryCmd(f, gf))
	cmd.AddCommand(newModelCreateCmd(f, gf))
	cmd.AddCommand(newModelUpdateCmd(f, gf))
	cmd.AddCommand(newModelDeleteCmd(f, gf))
	cmd.AddCommand(newModelAnalyzeCmd(f, gf))
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

func newModelAnalyzeCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var limit int
	var offset int
	cmd := &cobra.Command{
		Use:   "analyze <tql>",
		Short: "执行模型统计与分析查询",
		Long: helpSections(
			helpSection(
				"这是什么",
				"把一段 TQL 表达式发送到 `ModelSvc/aiSelect`，用于执行统计、聚合、趋势、排行或多模型关联分析。",
			),
			helpSection(
				"什么时候使用",
				"当你要做统计分析，而不是读取某个模型的明细记录列表时使用。",
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
				"如果你只想按过滤条件分页读取某个模型的明细数据，改用 `jit model query <fullName>`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "执行一条模型分析 TQL，并限制返回 20 条",
				Command:     `jit model analyze 'Select([F("id"), F("name")], From(["models.Customer"]), Limit(0, 20))' --limit 20`,
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
	var fieldsArg string
	var orderArg string
	var page int
	var size int
	var level int
	cmd := &cobra.Command{
		Use:   "query <fullName>",
		Short: "查询模型明细数据",
		Long: helpSections(
			helpSection(
				"这是什么",
				"按 AI 原子查询协议调用 `<modelPath>/aiQuery`，读取某个具体模型的明细记录列表。",
			),
			helpSection(
				"什么时候使用",
				"当你已经知道模型 fullName，并且想按过滤条件、字段、排序和分页规则读取记录时使用。",
			),
			helpSection(
				"输入说明",
				"位置参数是模型 fullName，例如 `models.Customer`。",
				"`--filter` 传的是 Q 表达式字符串；省略或传 `null` 时表示不过滤。",
				"`--fields` 传 JSON 数组字符串，例如 `[\"id\",\"name\"]`。",
				"`--order` 传 JSON 数组字符串，例如 `[[\"id\",-1]]` 或 `[\"-id\"]`。",
				"`--page`、`--size` 和 `--level` 分别控制页码、每页条数和关联层级。",
			),
			helpSection(
				"数据来源",
				"会实际请求目标模型的 `aiQuery` 接口，请求体中包含 `qfilter`、`fieldList`、`orderList`、`page`、`size` 和 `level`。",
			),
			helpSection(
				"输出说明",
				"输出后端返回的原始 JSON，不做二次包装。",
			),
			helpSection(
				"如果它不适合",
				"如果你要查看模型结构而不是模型数据，改用 `jit model get`。",
				"如果你需要的是统计、聚合或分析能力，改用 `jit model analyze`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "查询 Customer 模型第一页明细数据",
				Command:     "jit model query models.Customer",
			},
			helpExample{
				Description: "带过滤、字段和排序查询 Customer 明细数据",
				Command:     `jit model query models.Customer --filter 'Q("name", "=", "Alice")' --fields '["id","name"]' --order '[["id",-1]]' --page 1 --size 20`,
			},
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fieldList, err := parseOptionalJSONArrayArg(fieldsArg, "--fields")
			if err != nil {
				return err
			}
			orderList, err := parseOptionalJSONArrayArg(orderArg, "--order")
			if err != nil {
				return err
			}
			filterValue := optionalFilterValue(filterArg)

			return runModelClassMethodCall(cmd, f, gf, args[0], "aiQuery", map[string]any{
				"qfilter":   filterValue,
				"fieldList": fieldList,
				"orderList": orderList,
				"page":      page,
				"size":      size,
				"level":     level,
			})
		},
	}
	cmd.Flags().StringVar(&filterArg, "filter", "", `查询过滤条件 Q 表达式字符串，例如 Q("name", "=", "Alice")`)
	cmd.Flags().StringVar(&fieldsArg, "fields", "", `返回字段 JSON 数组，例如 ["id","name"]`)
	cmd.Flags().StringVar(&orderArg, "order", "", `排序 JSON 数组，例如 [["id",-1]] 或 ["-id"]`)
	cmd.Flags().IntVar(&page, "page", 1, "页码")
	cmd.Flags().IntVar(&size, "size", 20, "每页条数")
	cmd.Flags().IntVar(&level, "level", 2, "关联层级")
	return cmd
}

func newModelCreateCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var dataArg string
	var triggerEvent int

	cmd := &cobra.Command{
		Use:   "create <fullName>",
		Short: "创建模型记录",
		Long: helpSections(
			helpSection(
				"这是什么",
				"按 AI 原子创建协议调用 `<modelPath>/aiCreate`，向模型新增一条记录。",
			),
			helpSection(
				"什么时候使用",
				"当你已经知道模型 fullName，并且要直接创建一条业务记录时使用。",
			),
			helpSection(
				"输入说明",
				"位置参数是模型 fullName，例如 `models.Customer`。",
				"`--data` 必须是 JSON 对象；传 `@-` 时从 stdin 读取。",
				"`--trigger-event` 控制是否触发事件，默认 1。",
			),
			helpSection(
				"数据来源",
				"会实际请求目标模型的 `aiCreate` 接口，请求体中包含 `data` 和 `triggerEvent`。",
			),
			helpSection(
				"输出说明",
				"输出后端返回的原始 JSON，不做二次包装。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "创建一条 Customer 记录",
				Command:     `jit model create models.Customer --data '{"name":"Alice"}'`,
			},
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := parseRequiredJSONObjectArg(dataArg, f.IO.In, "--data")
			if err != nil {
				return err
			}

			return runModelClassMethodCall(cmd, f, gf, args[0], "aiCreate", map[string]any{
				"data":         data,
				"triggerEvent": triggerEvent,
			})
		},
	}

	cmd.Flags().StringVar(&dataArg, "data", "", "创建数据 JSON 对象；传 @- 表示从 stdin 读取")
	cmd.Flags().IntVar(&triggerEvent, "trigger-event", 1, "是否触发事件，1 开启，0 关闭")
	return cmd
}

func newModelUpdateCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var filterArg string
	var dataArg string
	var triggerEvent int

	cmd := &cobra.Command{
		Use:   "update <fullName>",
		Short: "更新模型记录",
		Long: helpSections(
			helpSection(
				"这是什么",
				"按 AI 原子更新协议调用 `<modelPath>/aiUpdate`，按过滤条件批量更新模型记录。",
			),
			helpSection(
				"什么时候使用",
				"当你已经知道模型 fullName，并且要按条件修改已有业务数据时使用。",
			),
			helpSection(
				"输入说明",
				"位置参数是模型 fullName，例如 `models.Customer`。",
				"`--filter` 必须是非空 Q 表达式字符串。",
				"`--data` 必须是 JSON 对象；传 `@-` 时从 stdin 读取。",
				"`--trigger-event` 控制是否触发事件，默认 1。",
			),
			helpSection(
				"数据来源",
				"会实际请求目标模型的 `aiUpdate` 接口，请求体中包含 `qfilter`、`updateData` 和 `triggerEvent`。",
			),
			helpSection(
				"输出说明",
				"输出后端返回的原始 JSON，不做二次包装。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "按条件更新 Customer 记录",
				Command:     `jit model update models.Customer --filter 'Q("id","=",1)' --data '{"name":"Bob"}'`,
			},
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, err := requireNonEmptyFilter(filterArg)
			if err != nil {
				return err
			}
			data, err := parseRequiredJSONObjectArg(dataArg, f.IO.In, "--data")
			if err != nil {
				return err
			}

			return runModelClassMethodCall(cmd, f, gf, args[0], "aiUpdate", map[string]any{
				"qfilter":      filter,
				"updateData":   data,
				"triggerEvent": triggerEvent,
			})
		},
	}

	cmd.Flags().StringVar(&filterArg, "filter", "", `更新过滤条件 Q 表达式字符串，例如 Q("id","=",1)`)
	cmd.Flags().StringVar(&dataArg, "data", "", "更新数据 JSON 对象；传 @- 表示从 stdin 读取")
	cmd.Flags().IntVar(&triggerEvent, "trigger-event", 1, "是否触发事件，1 开启，0 关闭")
	return cmd
}

func newModelDeleteCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var filterArg string
	var triggerEvent int

	cmd := &cobra.Command{
		Use:   "delete <fullName>",
		Short: "删除模型记录",
		Long: helpSections(
			helpSection(
				"这是什么",
				"按 AI 原子删除协议调用 `<modelPath>/aiDelete`，按过滤条件删除模型记录。",
			),
			helpSection(
				"什么时候使用",
				"当你已经知道模型 fullName，并且要按条件删除业务数据时使用。",
			),
			helpSection(
				"输入说明",
				"位置参数是模型 fullName，例如 `models.Customer`。",
				"`--filter` 必须是非空 Q 表达式字符串。",
				"`--trigger-event` 控制是否触发事件，默认 1。",
			),
			helpSection(
				"数据来源",
				"会实际请求目标模型的 `aiDelete` 接口，请求体中包含 `qfilter` 和 `triggerEvent`。",
			),
			helpSection(
				"输出说明",
				"输出后端返回的原始 JSON，不做二次包装。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "按条件删除 Customer 记录",
				Command:     `jit model delete models.Customer --filter 'Q("id","=",1)'`,
			},
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, err := requireNonEmptyFilter(filterArg)
			if err != nil {
				return err
			}

			return runModelClassMethodCall(cmd, f, gf, args[0], "aiDelete", map[string]any{
				"qfilter":      filter,
				"triggerEvent": triggerEvent,
			})
		},
	}

	cmd.Flags().StringVar(&filterArg, "filter", "", `删除过滤条件 Q 表达式字符串，例如 Q("id","=",1)`)
	cmd.Flags().IntVar(&triggerEvent, "trigger-event", 1, "是否触发事件，1 开启，0 关闭")
	return cmd
}

func parseRequiredJSONObjectArg(dataArg string, in io.Reader, flagName string) (map[string]any, error) {
	if strings.TrimSpace(dataArg) == "" {
		return nil, NewCLIError("missing_data", fmt.Sprintf("%s is required", flagName))
	}

	raw, err := parseJSONArg(dataArg, in)
	if err != nil {
		return nil, NewCLIError("invalid_data", err.Error())
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil || payload == nil {
		return nil, NewCLIError("invalid_data", fmt.Sprintf("%s must be a JSON object", flagName))
	}
	return payload, nil
}

func parseOptionalJSONArrayArg(dataArg string, flagName string) ([]any, error) {
	raw := strings.TrimSpace(dataArg)
	if raw == "" || strings.EqualFold(raw, "null") {
		return nil, nil
	}

	var payload []any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil || payload == nil {
		key := "invalid_" + strings.TrimPrefix(flagName, "--")
		return nil, NewCLIError(key, fmt.Sprintf("%s must be a JSON array", flagName))
	}
	return payload, nil
}

func optionalFilterValue(dataArg string) any {
	value := strings.TrimSpace(dataArg)
	if value == "" || strings.EqualFold(value, "null") {
		return nil
	}
	return value
}

func newModelClassMethodBody(payload map[string]any) json.RawMessage {
	body, _ := json.Marshal(map[string]any{
		"methodType": "cls",
		"argDict":    payload,
	})
	return body
}

func requireNonEmptyFilter(dataArg string) (string, error) {
	value := strings.TrimSpace(dataArg)
	if value == "" || strings.EqualFold(value, "null") {
		return "", NewCLIError("missing_filter", "--filter is required")
	}
	return value, nil
}

func runModelClassMethodCall(
	cmd *cobra.Command,
	f *Factory,
	gf *GlobalFlags,
	fullName string,
	methodName string,
	payload map[string]any,
) error {
	body := newModelClassMethodBody(payload)
	endpoint := fmt.Sprintf("%s/%s", modelPath(fullName), methodName)
	return runModelSvcCall(cmd, f, gf, endpoint, body)
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
