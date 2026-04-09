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
		Short: "模型定义与查询快捷命令",
	}

	cmd.AddCommand(newModelListCmd(f, gf))
	cmd.AddCommand(newModelInfoCmd(f, gf))
	cmd.AddCommand(newModelSelectCmd(f, gf))
	cmd.AddCommand(newModelQueryCmd(f, gf))
	return cmd
}

func newModelListCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出 appInfo 缓存中的模型",
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

func newModelInfoCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "info <fullName>",
		Short: "显示指定 fullName 的模型定义",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, _ := json.Marshal(map[string]any{"fullName": args[0]})
			return runModelSvcCall(cmd, f, gf, modelSvcGetModelInfo, body)
		},
	}
}

func newModelSelectCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var limit int
	var offset int
	cmd := &cobra.Command{
		Use:   "select <tql>",
		Short: "通过 ModelSvc/aiSelect 执行 TQL 查询",
		Args:  cobra.ExactArgs(1),
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
		Args:  cobra.ExactArgs(1),
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
