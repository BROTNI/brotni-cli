package commands

import (
	"context"
	"fmt"

	"github.com/BROTNI/brotni-cli/internal/api"
	"github.com/BROTNI/brotni-cli/internal/output"
	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Manage simulation reports",
}

var (
	reportSimID  string
	reportFormat string
)

var reportExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export a simulation report",
	Long: `Export the report for a completed simulation run.

Supported formats: json, html, csv.

Use --output json for machine-readable metadata about the export.`,
	Example: `  brotni report export --simulation sim-abc123 --format json
  brotni report export --simulation sim-abc123 --format html`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if reportSimID == "" {
			return fmt.Errorf("--simulation is required")
		}

		output.PrintDebug(cfg.Debug, "exporting report: simulation=%s format=%s", reportSimID, reportFormat)

		if cfg.DryRun {
			fmt.Printf("[dry-run] would export %s report for simulation %s\n", reportFormat, reportSimID)
			return nil
		}

		client := api.NewClient(cfg.APIURL, cfg.Token, cfg.Debug)

		resp, err := client.ExportReport(context.Background(), reportSimID, reportFormat)
		if err != nil {
			return fmt.Errorf("exporting report: %w", err)
		}

		if cfg.Output == "json" {
			return printer.PrintJSON(resp)
		}

		if resp.Content != "" {
			fmt.Print(resp.Content)
		} else if resp.URL != "" {
			fmt.Printf("Report available at: %s\n", resp.URL)
		}

		return nil
	},
}

func init() {
	reportCmd.AddCommand(reportExportCmd)

	reportExportCmd.Flags().StringVar(&reportSimID, "simulation", "", "simulation run ID (required)")
	reportExportCmd.Flags().StringVar(&reportFormat, "format", "json", "report format: json, html, csv")
}
