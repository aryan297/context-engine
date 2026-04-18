package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

func NewQueryCmd(serverURL string) *cobra.Command {
	return &cobra.Command{
		Use:   "query [query string]",
		Short: "Query the context engine for relevant project context",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			projectName, _ := cmd.Flags().GetString("project")
			if projectName == "" {
				return fmt.Errorf("--project flag is required")
			}

			payload := map[string]string{
				"project_name": projectName,
				"query":        query,
			}
			body, _ := json.Marshal(payload)

			resp, err := http.Post(serverURL+"/v1/query-context", "application/json", bytes.NewReader(body))
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			data, _ := io.ReadAll(resp.Body)

			var pretty map[string]any
			if json.Unmarshal(data, &pretty) == nil {
				out, _ := json.MarshalIndent(pretty, "", "  ")
				fmt.Println(string(out))
			} else {
				fmt.Println(string(data))
			}
			return nil
		},
	}
}
