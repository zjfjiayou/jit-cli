package cmd

import (
	"io"

	"github.com/spf13/cobra"
)

func newAPICmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var dataArg string
	cmd := &cobra.Command{
		Use:   "api <elementPath/functionName>",
		Short: "原始 API 网关，透传输出后端响应",
		Long: helpSections(
			helpSection(
				"这是什么",
				"这是最低层、最通用的原始调用入口，直接对目标 app 下的某个 endpoint 发请求，并把后端响应原样输出到 stdout。",
			),
			helpSection(
				"什么时候使用",
				"当 `model`、`service` 这类快捷命令不适用，或者你已经明确知道目标 endpoint 时，优先使用这个命令。",
			),
			helpSection(
				"默认上下文",
				"默认通过当前 profile 的 default_app 发请求；传 `--app` 时会覆盖目标 app。",
				"默认使用当前 profile 中保存的 server 和 PAT。",
			),
			helpSection(
				"输入说明",
				"第一个位置参数是相对 API 路径，例如 `corps/services/MemberSvc/getCurrUserInfo`。",
				"请求体通过 `--data` 传入 JSON；传 `@-` 时会从 stdin 读取 JSON。",
			),
			helpSection(
				"输出说明",
				"输出就是后端返回的原始 JSON，不会额外包裹 CLI 自定义结构。",
			),
			helpSection(
				"如果它不适合",
				"想浏览模型定义或查询模型数据时，优先用 `jit model ...`。",
				"想调用带函数名的可调用元素时，优先用 `jit service call ...`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "直接调用当前 app 下的当前用户接口",
				Command:     "jit api corps/services/MemberSvc/getCurrUserInfo",
			},
			helpExample{
				Description: "对指定 endpoint 发送 JSON 请求体",
				Command:     `jit api auths/loginTypes/services/AuthSvc/listCliTokens --data '{}'`,
			},
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := parseJSONArg(dataArg, f.IO.In)
			if err != nil {
				return NewCLIError("invalid_data", err.Error())
			}
			return runAPIRequest(cmd, f, APIRequest{
				Profile:  gf.Profile,
				App:      gf.App,
				Endpoint: args[0],
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

func runAPIRequest(cmd *cobra.Command, f *Factory, req APIRequest) error {
	resp, err := f.Runtime.CallAPI(cmd.Context(), req)
	if err != nil {
		return err
	}
	return writeAPIResponse(f.IO.Out, resp)
}

func writeAPIResponse(out io.Writer, resp APIResponse) error {
	if err := writeRawJSON(out, resp.Raw); err != nil {
		return NewCLIError("write_output_failed", err.Error())
	}
	if apiResponseHasBackendError(resp) {
		return &ExitCodeOnlyError{Code: ExitBackendError}
	}
	return nil
}

func apiResponseHasBackendError(resp APIResponse) bool {
	if resp.HasErrCode {
		return resp.ErrCode != 0
	}
	if code, ok := parseErrCode(resp.Raw); ok {
		return code != 0
	}
	return false
}
