package cmd

import "github.com/spf13/cobra"

func newWhoamiCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "显示当前 PAT 对应的用户信息",
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
