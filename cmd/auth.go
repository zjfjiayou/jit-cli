package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

func newAuthCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "PAT 与 profile 管理",
		Long: helpSections(
			helpSection(
				"这组命令是什么",
				"这组命令用来管理本地身份与连接上下文，而不是管理远端用户账号本身。",
				"profile 会保存 server、default_app 和本地 token 存储位置，后续大多数命令默认都从当前 profile 读取连接信息。",
			),
			helpSection(
				"什么时候使用",
				"第一次把 PAT 绑定到本机、切换默认上下文、确认当前身份、或清理本地 token/profile 时使用这组命令。",
			),
			helpSection(
				"默认上下文",
				"`auth login` 不依赖当前 profile；其它命令默认读取当前 profile，除非显式传 `--profile` 或位置参数。",
			),
			helpSection(
				"如果它不适合",
				"如果你已经知道完整 endpoint，只想直接访问接口，改用 `jit api`。",
			),
		),
	}

	cmd.AddCommand(newAuthLoginCmd(f, gf))
	cmd.AddCommand(newAuthWhoamiCmd(f, gf))
	cmd.AddCommand(newAuthLogoutCmd(f, gf))
	cmd.AddCommand(newAuthRmCmd(f))
	cmd.AddCommand(newAuthLsCmd(f))
	cmd.AddCommand(newAuthUseCmd(f))

	return cmd
}

func newAuthLoginCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var server string
	var app string
	var token string
	var profile string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "使用 PAT 登录并创建或更新 profile",
		Long: helpSections(
			helpSection(
				"这是什么",
				"把一个可用的 PAT 绑定成一个本地 profile，后续命令就能复用这个上下文访问系统。",
			),
			helpSection(
				"什么时候使用",
				"第一次在本机配置 CLI、需要替换 PAT、或想把另一个 server/app 保存成新的本地上下文时使用。",
			),
			helpSection(
				"默认上下文",
				"这个命令不依赖当前 profile。",
				"未传 `--profile` 时，会根据 `--server` 自动推导 profile 名称。",
			),
			helpSection(
				"副作用",
				"会校验 PAT 是否可用，并把 server、default_app 和 token 写入本地。",
				"成功后会把该 profile 设为当前默认 profile。",
			),
			helpSection(
				"数据来源",
				"会调用当前用户接口验证 PAT，对后端的校验结果以实时响应为准。",
			),
			helpSection(
				"输入说明",
				"`--server` 和 `--app` 必填。",
				"`--token` 省略时，会尝试从 stdin 读取 PAT。",
			),
			helpSection(
				"输出说明",
				"成功时输出当前用户接口返回的 JSON；失败时在 stderr 输出 CLI 错误。",
			),
			helpSection(
				"如果它不适合",
				"如果只是想确认当前身份，不需要重新登录，改用 `jit auth whoami` 或 `jit whoami`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "把一个 PAT 绑定成新的本地上下文",
				Command:     "jit auth login --server http://127.0.0.1:8080 --app whwy/mmm --token jit_pat_xxx",
			},
			helpExample{
				Description: "从 stdin 读取 PAT，而不是写在命令行参数里",
				Command:     "printf '%s' \"$TOKEN\" | jit auth login --server http://127.0.0.1:8080 --app whwy/mmm",
			},
		),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(server) == "" {
				return NewCLIError("missing_server", "--server is required")
			}
			if strings.TrimSpace(app) == "" {
				return NewCLIError("missing_app", "--app is required")
			}

			if strings.TrimSpace(token) == "" {
				readToken, err := readTokenFromStdin(f.IO.In)
				if err != nil {
					return NewCLIError("read_token_failed", err.Error())
				}
				token = readToken
			}

			if strings.TrimSpace(token) == "" {
				return NewCLIError("missing_token", "token is required, pass --token or pipe from stdin")
			}

			resp, err := f.Runtime.AuthLogin(cmd.Context(), AuthLoginInput{
				Server:  server,
				App:     app,
				Profile: firstNonEmpty(profile, gf.Profile),
				Token:   token,
				DryRun:  gf.DryRun,
			})
			if err != nil {
				return err
			}
			return writeResponsePayload(f.IO.Out, resp)
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "JIT 服务地址")
	cmd.Flags().StringVar(&app, "app", "", "默认应用，格式为 org/app")
	cmd.Flags().StringVar(&token, "token", "", "PAT 值；省略时从 stdin 读取")
	cmd.Flags().StringVar(&profile, "profile", "", "profile 名称")
	return cmd
}

func newAuthWhoamiCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	return newCurrentUserCmd(
		"whoami",
		"显示当前 profile 的身份信息",
		helpSections(
			helpSection(
				"这是什么",
				"用当前 profile 里的 PAT 请求当前用户接口，确认这个 profile 实际对应的身份。",
			),
			helpSection(
				"什么时候使用",
				"切换 profile 后、怀疑 token 过期时、或想确认当前默认上下文到底是谁时使用。",
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
				"输出后端返回的当前用户 JSON，常见字段包括用户、成员和组织信息。",
			),
			helpSection(
				"如果它不适合",
				"如果你只想用最短命令查看身份，直接用根命令 `jit whoami`。",
			),
		),
		helpExamples(
			helpExample{
				Description: "确认当前默认 profile 对应的身份",
				Command:     "jit auth whoami",
			},
		),
		f,
		gf,
	)
}

func newAuthLogoutCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var localProfile string
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "移除指定 profile 的 PAT，不删除 profile",
		Long: helpSections(
			helpSection(
				"这是什么",
				"只删除指定 profile 上保存的 PAT，不删除 profile 配置本身。",
			),
			helpSection(
				"什么时候使用",
				"需要重新登录、暂时清掉本地 token、但仍想保留 server 和 default_app 时使用。",
			),
			helpSection(
				"默认上下文",
				"默认操作当前 profile；传 `--profile` 时会覆盖目标 profile。",
			),
			helpSection(
				"副作用",
				"会删除本地保存的 token。",
				"profile 配置和 appInfo 缓存会继续保留。",
			),
			helpSection(
				"输出说明",
				"成功时输出 `{ok:true}`，如果显式指定了 profile，还会输出 profile 名称。",
			),
			helpSection(
				"如果它不适合",
				"如果你想连 profile 配置和缓存一起删除，改用 `jit auth rm <profile>`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "只移除当前 profile 的 token，保留本地配置",
				Command:     "jit auth logout",
			},
		),
		RunE: func(cmd *cobra.Command, _ []string) error {
			profile := firstNonEmpty(localProfile, gf.Profile)
			if err := f.Runtime.AuthLogout(cmd.Context(), profile); err != nil {
				return err
			}
			payload := map[string]any{"ok": true}
			if profile != "" {
				payload["profile"] = profile
			}
			return writeJSON(f.IO.Out, payload)
		},
	}
	cmd.Flags().StringVar(&localProfile, "profile", "", "覆盖当前使用的 profile 名称")
	return cmd
}

func newAuthRmCmd(f *Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "rm <profile>",
		Short: "删除整个 profile（配置、PAT、缓存）",
		Long: helpSections(
			helpSection(
				"这是什么",
				"彻底删除一个本地 profile，包括它的配置、PAT 和本地 appInfo 缓存。",
			),
			helpSection(
				"什么时候使用",
				"当某个本地上下文已经不再需要，或者你想把它从本机完全移除时使用。",
			),
			helpSection(
				"默认上下文",
				"这个命令必须显式传入要删除的 profile 名称。",
			),
			helpSection(
				"副作用",
				"会删除 config、credentials 和 appinfo 缓存文件。",
				"如果删除的是当前 profile，会同步清空 current_profile。",
			),
			helpSection(
				"输出说明",
				"成功时输出 `{ok:true, profile:<name>}`。",
			),
			helpSection(
				"如果它不适合",
				"如果你只是想删 token、保留 profile 配置，改用 `jit auth logout`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "彻底删除名为 demo 的本地上下文",
				Command:     "jit auth rm demo",
			},
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := f.Runtime.AuthRemove(cmd.Context(), args[0]); err != nil {
				return err
			}
			return writeJSON(f.IO.Out, map[string]any{
				"ok":      true,
				"profile": args[0],
			})
		},
	}
}

func newAuthLsCmd(f *Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "列出所有 profile",
		Long: helpSections(
			helpSection(
				"这是什么",
				"列出当前机器上保存的所有本地上下文。",
			),
			helpSection(
				"什么时候使用",
				"当你不知道本机上有哪些 profile、哪个是当前 profile，或者想拿到 index 供 `auth use` 选择时使用。",
			),
			helpSection(
				"数据来源",
				"数据来自本地 profile 目录和本地 token 存储，不会请求后端。",
			),
			helpSection(
				"输出说明",
				"输出 `profiles` 数组，每项包含名称、server、default_app、是否当前 profile，以及是否保存了 PAT。",
				"列表顺序稳定，并附带 `index` 字段，可直接用于 `jit auth use <index>`。",
			),
			helpSection(
				"如果它不适合",
				"如果你已经知道要切换到哪个 profile，直接用 `jit auth use <profile|index>`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "列出本机所有可用的本地上下文",
				Command:     "jit auth ls",
			},
		),
		RunE: func(cmd *cobra.Command, _ []string) error {
			items, err := f.Runtime.AuthList(cmd.Context())
			if err != nil {
				return err
			}
			return writeJSON(f.IO.Out, map[string]any{
				"profiles": items,
			})
		},
	}
}

func newAuthUseCmd(f *Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "use <profile|index>",
		Short: "切换当前 profile，支持名称或列表索引",
		Long: helpSections(
			helpSection(
				"这是什么",
				"把某个 profile 设为当前默认上下文，后续命令在不传 `--profile` 时就会使用它。",
			),
			helpSection(
				"什么时候使用",
				"当你需要在多个本地上下文之间切换默认环境时使用。",
			),
			helpSection(
				"输入说明",
				"支持直接传 profile 名称，也支持传 `jit auth ls` 输出中的 index。",
				"解析顺序是先按真实 profile 名匹配，名称不存在时再按索引解析。",
			),
			helpSection(
				"副作用",
				"会更新本地全局配置里的 current_profile。",
			),
			helpSection(
				"输出说明",
				"成功时输出 `{current_profile:<resolved_name>}`，其中值是解析后的真实 profile 名称。",
			),
			helpSection(
				"如果它不适合",
				"如果你只是想对单次命令临时指定上下文，不需要切换默认值，直接在命令上使用 `--profile`。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "按名称切换默认 profile",
				Command:     "jit auth use demo",
			},
			helpExample{
				Description: "按 `jit auth ls` 输出里的 index 切换默认 profile",
				Command:     "jit auth use 0",
			},
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileName, err := f.Runtime.AuthUse(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeJSON(f.IO.Out, map[string]any{
				"current_profile": profileName,
			})
		},
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
