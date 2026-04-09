package cmd

import "github.com/spf13/cobra"

func newWhoamiCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	return newCurrentUserCmd(
		"whoami",
		"显示当前 PAT 对应的用户信息",
		helpSections(
			helpSection(
				"这是什么",
				"这是查看当前 CLI 身份的最短入口，会用当前 PAT 请求当前用户接口。",
			),
			helpSection(
				"什么时候使用",
				"当你想确认 CLI 现在是以哪个用户、哪个成员身份访问系统时使用它。",
			),
			helpSection(
				"默认上下文",
				"默认读取当前 profile；传 `--profile` 或 `--app` 时会覆盖默认上下文。",
			),
			helpSection(
				"副作用",
				"不会修改任何本地状态，只会发起一次只读身份查询请求。",
			),
			helpSection(
				"数据来源",
				"数据来自后端当前用户接口，而不是本地缓存。",
			),
			helpSection(
				"输出说明",
				"输出后端返回的当前用户 JSON，常见字段包括用户信息、成员信息和组织信息。",
			),
			helpSection(
				"如果它不适合",
				"如果你想看有哪些本地上下文可选，改用 `jit auth ls`。",
			),
		),
		helpExamples(
			helpExample{
				Description: "确认当前默认上下文对应的身份",
				Command:     "jit whoami",
			},
			helpExample{
				Description: "临时查看另一个 profile 的身份",
				Command:     "jit whoami --profile demo",
			},
		),
		f,
		gf,
	)
}

func newCurrentUserCmd(use string, short string, long string, example string, f *Factory, gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Example: example,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resp, err := f.Runtime.GetCurrUserInfo(cmd.Context(), UserInfoInput{
				Profile: gf.Profile,
				App:     gf.App,
				JQ:      gf.JQ,
				DryRun:  gf.DryRun,
			})
			if err != nil {
				return err
			}
			return writeRawPayload(f.IO.Out, resp)
		},
	}
}
