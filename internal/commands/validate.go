package commands

import (
	"fmt"
	"os"

	"github.com/BROTNI/brotni-cli/internal/output"
	"github.com/BROTNI/brotni-cli/internal/validate"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate Brotni spec files",
	Long: `Validate YAML spec files against Brotni schemas.

Exits with code 0 on success, non-zero on validation failure.
Use --output json for machine-readable output in CI pipelines.`,
}

var validateRecipeCmd = &cobra.Command{
	Use:   "recipe <file>",
	Short: "Validate a runtime execution recipe file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runValidate(args[0], validate.ValidateRecipe)
	},
}

var validateContextCmd = &cobra.Command{
	Use:   "context <file>",
	Short: "Validate a context definition file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runValidate(args[0], validate.ValidateContext)
	},
}

var validateSimulationCmd = &cobra.Command{
	Use:   "simulation <file>",
	Short: "Validate a simulation spec file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runValidate(args[0], validate.ValidateSimulation)
	},
}

var validateCampaignCmd = &cobra.Command{
	Use:   "campaign <file>",
	Short: "Validate a campaign manifest (.brotni/simulation.yaml)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runValidate(args[0], validate.ValidateCampaign)
	},
}

func init() {
	validateCmd.AddCommand(validateRecipeCmd)
	validateCmd.AddCommand(validateContextCmd)
	validateCmd.AddCommand(validateSimulationCmd)
	validateCmd.AddCommand(validateCampaignCmd)
}

func runValidate(file string, validateFn func(string) (*validate.ValidationResult, error)) error {
	output.PrintDebug(cfg.Debug, "validating file: %s", file)

	result, err := validateFn(file)
	if err != nil {
		return err
	}

	if cfg.Output == "json" {
		return printer.PrintJSON(result)
	}

	if result.Valid {
		fmt.Printf("✓ %s is valid\n", file)
		for _, w := range result.Warnings {
			fmt.Printf("  warning: %s\n", w)
		}
	} else {
		fmt.Fprintf(os.Stderr, "✗ %s is invalid\n", file)
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  error: %s\n", e)
		}
		for _, w := range result.Warnings {
			fmt.Fprintf(os.Stderr, "  warning: %s\n", w)
		}
		os.Exit(1)
	}

	return nil
}
