package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

func NewIngestCmd(serverURL string) *cobra.Command {
	return &cobra.Command{
		Use:   "ingest [path]",
		Short: "Ingest a project directory into the context engine",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			projectName, _ := cmd.Flags().GetString("project")
			if projectName == "" {
				projectName = path
			}

			absPath, err := os.Getwd()
			if err != nil {
				absPath = path
			}
			if path != "." {
				absPath = path
			}

			payload := map[string]string{
				"project_name": projectName,
				"path":         absPath,
			}
			body, _ := json.Marshal(payload)

			resp, err := http.Post(serverURL+"/v1/ingest-project", "application/json", bytes.NewReader(body))
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			data, _ := io.ReadAll(resp.Body)
			fmt.Println(string(data))
			return nil
		},
	}
}
