package cmd

import (
	"context"

	"jit-cli/internal/build"

	"github.com/spf13/cobra"
)

func Execute(ctx context.Context) int {
	factory := NewDefaultFactory()
	root := NewRootCmd(factory)
	if err := root.ExecuteContext(ctx); err != nil {
		return handleRootError(factory, err)
	}
	return ExitOK
}

func NewRootCmd(f *Factory) *cobra.Command {
	gf := &GlobalFlags{Format: "json"}

	root := &cobra.Command{
		Use:   "jit",
		Short: "基于 PAT 鉴权的 JIT 命令行工具，可直接访问原始 API",
		Long: helpSections(
			helpSection(
				"这是什么",
				"JitCli 是一个用 PAT 直接访问 JIT 后端接口的命令行工具，适合 AI Agent、脚本和排障场景。",
				"它的目标不是模拟浏览器，而是让你在不了解前端登录流程的前提下，也能从命令行直接访问系统接口。",
			),
			helpSection(
				"术语说明",
				"PAT：长期有效的个人访问令牌，请求时通过 `Authorization: Bearer <token>` 发送。",
				"profile：保存在本地的一组连接上下文，至少包含 server、default_app 和 token 存储位置。",
				"app：要访问的业务应用，格式为 `org/app`。大多数命令默认读取当前 profile 的 default_app。",
				"appInfo 缓存：从前端 `appInfo.js` 拉取并保存在本地的元素目录缓存，用来浏览元素，不等同于后端最终可调用全集。",
				"model：数据模型元素，可用于查看定义或查询业务数据。",
				"service：带函数列表的可调用元素，可通过标准 API 路径调用函数。",
				"api：最低层原始调用入口，其它快捷命令本质上是对缓存或 API 的封装。",
			),
			helpSection(
				"默认上下文",
				"除显式传入 `--profile` 或 `--app` 外，大多数命令默认使用当前 profile 和它的 default_app。",
				"当前版本默认所有输出都是 JSON，可继续配合 `--jq` 做提取。",
			),
			helpSection(
				"如何选命令",
				"确认当前身份：`jit whoami` 或 `jit auth whoami`。",
				"切换本地上下文：先 `jit auth ls`，再 `jit auth use <profile|index>`。",
				"刷新当前 app 的元素目录缓存：`jit app refresh`。",
				"浏览当前 app 暴露的元素：`jit app ls`。",
				"查看模型定义：`jit model get <fullName>`；查询模型数据：`jit model query <fullName>`；只有已经明确要用 TQL 表达式时再用 `jit model tql <expr>`。",
				"调用服务函数：`jit service call <fullName> <functionName>`。",
				"上层快捷命令不适用时，使用 `jit api <endpoint>` 直接访问接口。",
			),
		),
		Example: helpExamples(
			helpExample{
				Description: "把一个 PAT 绑定成当前机器上的本地上下文",
				Command:     "jit auth login --server http://127.0.0.1:8080 --app whwy/mmm --token jit_pat_xxx",
			},
			helpExample{
				Description: "确认当前默认上下文对应的身份",
				Command:     "jit whoami",
			},
		),
		Version:       build.Version,
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if gf.Format == "" {
				gf.Format = "json"
			}
			if gf.Format != "json" {
				return NewCLIError("unsupported_format", "当前版本仅支持 --format json")
			}
			return nil
		},
	}

	root.PersistentFlags().StringVar(&gf.Profile, "profile", "", "profile 名称")
	root.PersistentFlags().StringVar(&gf.App, "app", "", "目标应用，格式为 org/app")
	root.PersistentFlags().StringVar(&gf.JQ, "jq", "", "用于 JSON 输出的 jq 表达式")
	root.PersistentFlags().StringVar(&gf.Format, "format", "json", "输出格式")
	root.PersistentFlags().BoolVar(&gf.DryRun, "dry-run", false, "仅输出请求预览，不实际执行")

	root.AddCommand(newAuthCmd(f, gf))
	root.AddCommand(newAPICmd(f, gf))
	root.AddCommand(newAppCmd(f, gf))
	root.AddCommand(newModelCmd(f, gf))
	root.AddCommand(newServiceCmd(f, gf))
	root.AddCommand(newWhoamiCmd(f, gf))
	return root
}

func handleRootError(f *Factory, err error) int {
	switch typed := err.(type) {
	case *CLIError:
		writeCLIErrorJSON(f.IO.ErrOut, typed)
		return typed.Code
	case *ExitCodeOnlyError:
		return typed.Code
	default:
		writeCLIErrorJSON(f.IO.ErrOut, NewCLIError("cli_error", err.Error()))
		return ExitCLIError
	}
}
