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
		Use:           "jit",
		Short:         "基于 PAT 鉴权的 JIT 命令行工具，可直接访问原始 API",
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
