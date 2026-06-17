package commands

import (
	"context"
	"fmt"

	"github.com/BROTNI/brotni-cli/internal/api"
	"github.com/BROTNI/brotni-cli/internal/output"
	"github.com/spf13/cobra"
)

var simulationCmd = &cobra.Command{
	Use:   "simulation",
	Short: "Manage simulation runs",
}

var (
	simCandidateID string
	simStatusID    string
)

var simulationRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Trigger a simulation run",
	Long: `Trigger a new simulation run on the Brotni platform.

Use --candidate to run a previously submitted candidate, or omit to trigger
a run based on the latest configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		output.PrintDebug(cfg.Debug, "triggering simulation run: candidate=%s", simCandidateID)

		if cfg.DryRun {
			if simCandidateID != "" {
				fmt.Printf("[dry-run] would trigger simulation run for candidate %s\n", simCandidateID)
			} else {
				fmt.Println("[dry-run] would trigger simulation run")
			}
			return nil
		}

		client := api.NewClient(cfg.APIURL, cfg.Token, cfg.Debug)

		resp, err := client.RunSimulation(context.Background(), api.SimulationRunRequest{
			CandidateID: simCandidateID,
			DryRun:      cfg.DryRun,
		})
		if err != nil {
			return fmt.Errorf("running simulation: %w", err)
		}

		if cfg.Output == "json" {
			return printer.PrintJSON(resp)
		}

		fmt.Println("Simulation triggered")
		fmt.Printf("  ID:      %s\n", resp.ID)
		fmt.Printf("  Status:  %s\n", resp.Status)
		if resp.Message != "" {
			fmt.Printf("  Message: %s\n", resp.Message)
		}

		return nil
	},
}

var simulationStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get simulation run status",
	Long: `Retrieve the current status of a simulation run.

Use --output json for polling in CI scripts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if simStatusID == "" {
			return fmt.Errorf("--id is required")
		}

		output.PrintDebug(cfg.Debug, "getting simulation status: id=%s", simStatusID)

		client := api.NewClient(cfg.APIURL, cfg.Token, cfg.Debug)

		resp, err := client.GetSimulationStatus(context.Background(), simStatusID)
		if err != nil {
			return fmt.Errorf("getting simulation status: %w", err)
		}

		if cfg.Output == "json" {
			return printer.PrintJSON(resp)
		}

		fmt.Println("Simulation Status")
		fmt.Printf("  ID:      %s\n", resp.ID)
		fmt.Printf("  Status:  %s\n", resp.Status)
		if resp.Phase != "" {
			fmt.Printf("  Phase:   %s\n", resp.Phase)
		}
		if resp.StartedAt != "" {
			fmt.Printf("  Started: %s\n", resp.StartedAt)
		}
		if resp.UpdatedAt != "" {
			fmt.Printf("  Updated: %s\n", resp.UpdatedAt)
		}
		if resp.Message != "" {
			fmt.Printf("  Message: %s\n", resp.Message)
		}

		return nil
	},
}

func init() {
	simulationCmd.AddCommand(simulationRunCmd)
	simulationCmd.AddCommand(simulationStatusCmd)

	simulationRunCmd.Flags().StringVar(&simCandidateID, "candidate", "", "candidate ID to simulate")
	simulationStatusCmd.Flags().StringVar(&simStatusID, "id", "", "simulation run ID (required)")
}
