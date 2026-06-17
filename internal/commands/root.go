package commands

import (
	"os"

	"github.com/BROTNI/brotni-cli/internal/config"
	"github.com/BROTNI/brotni-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	cfgFile   string
	outFormat string
	debugMode bool
	dryRun    bool

	cfg     *config.Config
	printer *output.Printer
)

var rootCmd = &cobra.Command{
	Use:   "brotni",
	Short: "Submit candidates and manage Brotni simulation workflows",
	Long: `brotni-cli is the official command-line tool for Brotni-compatible simulation workflows.

Submit candidates, validate specs, trigger simulations, inspect status,
and export reports — from local development, CI/CD pipelines, or automation.

Environment variables:
  BROTNI_API_URL   Base URL of the Brotni API (default: https://api.brotni.io)
  BROTNI_TOKEN     API authentication token`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		output.PrintError(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: .brotni.yaml)")
	rootCmd.PersistentFlags().StringVarP(&outFormat, "output", "o", "", "output format: table or json")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "enable debug output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "validate and print without executing")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(candidateCmd)
	rootCmd.AddCommand(simulationCmd)
	rootCmd.AddCommand(reportCmd)
}

func initConfig() {
	var err error
	cfg, err = config.Load(cfgFile)
	if err != nil {
		output.PrintError(err)
		os.Exit(1)
	}

	if outFormat != "" {
		cfg.Output = outFormat
	}
	if debugMode {
		cfg.Debug = true
	}
	if dryRun {
		cfg.DryRun = true
	}

	printer = output.NewPrinter(cfg.Output)
}
