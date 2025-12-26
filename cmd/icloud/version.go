package icloud

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jingkaihe/icloud-cli/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		outputJSON, _ := cmd.Flags().GetBool("json")

		if outputJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(map[string]any{
				"version":    version.Version,
				"git_commit": version.GitCommit,
				"build_time": version.BuildTime,
			})
			return
		}
		fmt.Printf("icloud %s\n", version.Version)
		fmt.Printf("  Commit: %s\n", version.GitCommit)
		fmt.Printf("  Built:  %s\n", version.BuildTime)
	},
}

func init() {
	versionCmd.Flags().Bool("json", false, "Output in JSON format")
	rootCmd.AddCommand(versionCmd)
}
