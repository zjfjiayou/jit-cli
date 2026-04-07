package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

func newAuthCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "PAT and profile management",
	}

	cmd.AddCommand(newAuthLoginCmd(f, gf))
	cmd.AddCommand(newAuthStatusCmd(f, gf))
	cmd.AddCommand(newAuthLogoutCmd(f, gf))
	cmd.AddCommand(newAuthListCmd(f, gf))
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
		Short: "login with PAT and create/update a profile",
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

	cmd.Flags().StringVar(&server, "server", "", "JIT server URL")
	cmd.Flags().StringVar(&app, "app", "", "default app in org/app format")
	cmd.Flags().StringVar(&token, "token", "", "PAT value, reads from stdin when omitted")
	cmd.Flags().StringVar(&profile, "profile", "", "profile name")
	return cmd
}

func newAuthStatusCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "validate PAT and show current user info",
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

func newAuthLogoutCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var localProfile string
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "remove PAT from a profile",
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
	cmd.Flags().StringVar(&localProfile, "profile", "", "profile name override")
	return cmd
}

func newAuthListCmd(f *Factory, _ *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "list profiles",
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
		Use:   "use <profile>",
		Short: "switch current profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := f.Runtime.AuthUse(cmd.Context(), args[0]); err != nil {
				return err
			}
			return writeJSON(f.IO.Out, map[string]any{
				"current_profile": args[0],
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
