package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"jit-cli/internal/profile"

	"github.com/spf13/cobra"
)

func newModelCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "model metadata and data shortcuts",
	}

	cmd.AddCommand(newModelListCmd(f, gf))
	cmd.AddCommand(newModelMetaCmd(f, gf))
	cmd.AddCommand(newModelInfoCmd(f, gf))
	cmd.AddCommand(newModelSelectCmd(f, gf))
	cmd.AddCommand(newModelQueryCmd(f, gf))
	cmd.AddCommand(newModelCreateCmd(f, gf))
	cmd.AddCommand(newModelUpdateCmd(f, gf))
	cmd.AddCommand(newModelDeleteCmd(f, gf))
	return cmd
}

func newModelListCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "list all models from ModelSvc",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runModelSvcCall(cmd, f, gf, modelSvcGetModelList, json.RawMessage("{}"))
		},
	}
}

func newModelMetaCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "meta",
		Short: "load model metadata from ModelSvc",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runModelSvcCall(cmd, f, gf, modelSvcGetModelsMeta, json.RawMessage("{}"))
		},
	}
}

func newModelInfoCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "info <fullName>",
		Short: "show one model definition by fullName",
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
		Short: "execute TQL query via ModelSvc/aiSelect",
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
	cmd.Flags().IntVar(&limit, "limit", 50, "query limit")
	cmd.Flags().IntVar(&offset, "offset", 0, "query offset")
	return cmd
}

func newModelQueryCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var filterArg string
	var page int
	var size int
	cmd := &cobra.Command{
		Use:   "query <fullName>",
		Short: "query model rows",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, err := parseJSONValue(filterArg)
			if err != nil {
				return NewCLIError("invalid_filter", err.Error())
			}
			body, _ := json.Marshal(map[string]any{
				"filter": filter,
				"page":   page,
				"size":   size,
			})
			endpoint := fmt.Sprintf("models/%s/query", modelPath(args[0]))
			return runModelDataCall(cmd, f, gf, endpoint, body)
		},
	}
	cmd.Flags().StringVar(&filterArg, "filter", "{}", "query filter JSON")
	cmd.Flags().IntVar(&page, "page", 1, "page number")
	cmd.Flags().IntVar(&size, "size", 10, "page size")
	return cmd
}

func newModelCreateCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var dataArg string
	cmd := &cobra.Command{
		Use:   "create <fullName>",
		Short: "create model row",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rowData, err := parseJSONValue(dataArg)
			if err != nil {
				return NewCLIError("invalid_data", err.Error())
			}
			body, _ := json.Marshal(map[string]any{
				"rowData": rowData,
			})
			endpoint := fmt.Sprintf("models/%s/create", modelPath(args[0]))
			return runModelDataCall(cmd, f, gf, endpoint, body)
		},
	}
	cmd.Flags().StringVar(&dataArg, "data", "{}", "rowData JSON")
	return cmd
}

func newModelUpdateCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var pkArg string
	var dataArg string
	cmd := &cobra.Command{
		Use:   "update <fullName>",
		Short: "update row by PK",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if pkArg == "" {
				return NewCLIError("missing_pk", "--pk is required")
			}
			pkList, err := parseJSONValue(pkArg)
			if err != nil {
				return NewCLIError("invalid_pk", err.Error())
			}
			updateData, err := parseJSONValue(dataArg)
			if err != nil {
				return NewCLIError("invalid_data", err.Error())
			}
			body, _ := json.Marshal(map[string]any{
				"pkList":     pkList,
				"updateData": updateData,
			})
			endpoint := fmt.Sprintf("models/%s/updateByPK", modelPath(args[0]))
			return runModelDataCall(cmd, f, gf, endpoint, body)
		},
	}
	cmd.Flags().StringVar(&pkArg, "pk", "", "primary key list JSON")
	cmd.Flags().StringVar(&dataArg, "data", "{}", "updateData JSON")
	return cmd
}

func newModelDeleteCmd(f *Factory, gf *GlobalFlags) *cobra.Command {
	var pkArg string
	cmd := &cobra.Command{
		Use:   "delete <fullName>",
		Short: "delete row by PK",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if pkArg == "" {
				return NewCLIError("missing_pk", "--pk is required")
			}
			pkList, err := parseJSONValue(pkArg)
			if err != nil {
				return NewCLIError("invalid_pk", err.Error())
			}
			body, _ := json.Marshal(map[string]any{
				"pkList": pkList,
			})
			endpoint := fmt.Sprintf("models/%s/deleteByPK", modelPath(args[0]))
			return runModelDataCall(cmd, f, gf, endpoint, body)
		},
	}
	cmd.Flags().StringVar(&pkArg, "pk", "", "primary key list JSON")
	return cmd
}

func runModelSvcCall(cmd *cobra.Command, f *Factory, gf *GlobalFlags, endpoint string, body json.RawMessage) error {
	targetApp, err := resolveModelSvcApp(cmd, f, gf)
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

func runModelDataCall(cmd *cobra.Command, f *Factory, gf *GlobalFlags, endpoint string, body json.RawMessage) error {
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

func resolveModelSvcApp(cmd *cobra.Command, f *Factory, gf *GlobalFlags) (string, error) {
	if gf.App != "" {
		return gf.App, nil
	}
	defaultApp, err := f.Runtime.ResolveApp(cmd.Context(), gf.Profile, "")
	if err != nil {
		return "", err
	}
	targetApp, err := profile.ResolveORMApp(defaultApp)
	if err != nil {
		return "", NewCLIError("invalid_default_app", err.Error())
	}
	return targetApp, nil
}

func modelPath(fullName string) string {
	return strings.ReplaceAll(strings.TrimSpace(fullName), ".", "/")
}
