package cmd

import (
	"io"

	"github.com/spf13/cobra"
)

func newAPICmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var dataArg string
	cmd := &cobra.Command{
		Use:   "api <elementPath/functionName>",
		Short: "raw API gateway, transparently prints backend response",
		Args:  cobra.ExactArgs(1),
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

	cmd.Flags().StringVar(&dataArg, "data", "{}", "request body JSON, or @- to read from stdin")
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
