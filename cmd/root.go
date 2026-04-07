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
		Short:         "JIT CLI for PAT-based auth and raw API access",
		Version:       build.Version,
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if gf.Format == "" {
				gf.Format = "json"
			}
			if gf.Format != "json" {
				return NewCLIError("unsupported_format", "only --format json is available in this release")
			}
			return nil
		},
	}

	root.PersistentFlags().StringVar(&gf.Profile, "profile", "", "profile name")
	root.PersistentFlags().StringVar(&gf.App, "app", "", "target app in org/app format")
	root.PersistentFlags().StringVar(&gf.JQ, "jq", "", "jq expression for JSON output")
	root.PersistentFlags().StringVar(&gf.Format, "format", "json", "output format")
	root.PersistentFlags().BoolVar(&gf.DryRun, "dry-run", false, "print request without executing")

	root.AddCommand(newAuthCmd(f, gf))
	root.AddCommand(newAPICmd(f, gf))
	root.AddCommand(newModelCmd(f, gf))
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
