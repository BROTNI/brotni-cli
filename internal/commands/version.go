package commands

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Set by -ldflags at build time.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

type versionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the brotni-cli version",
	RunE: func(cmd *cobra.Command, args []string) error {
		info := versionInfo{
			Version:   Version,
			Commit:    Commit,
			BuildDate: BuildDate,
			GoVersion: runtime.Version(),
			OS:        runtime.GOOS,
			Arch:      runtime.GOARCH,
		}

		if cfg.Output == "json" {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(info)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "brotni-cli %s\n", info.Version)
		fmt.Fprintf(cmd.OutOrStdout(), "  commit:     %s\n", info.Commit)
		fmt.Fprintf(cmd.OutOrStdout(), "  built:      %s\n", info.BuildDate)
		fmt.Fprintf(cmd.OutOrStdout(), "  go version: %s\n", info.GoVersion)
		fmt.Fprintf(cmd.OutOrStdout(), "  platform:   %s/%s\n", info.OS, info.Arch)
		return nil
	},
}
