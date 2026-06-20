package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/BROTNI/brotni-cli/internal/api"
	"github.com/BROTNI/brotni-cli/internal/output"
	"github.com/BROTNI/brotni-cli/internal/validate"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var campaignCmd = &cobra.Command{
	Use:   "campaign",
	Short: "Create and compare simulation campaigns",
	Long: `A simulation campaign groups multiple change candidates under one work
item and compares them on the same goals and constraints.

Define a campaign declaratively in .brotni/simulation.yaml, create it, register
or discover candidates, then compare scorecards and read the decision report.`,
}

var (
	campaignManifest       string
	campaignID             string
	campaignScoringVersion int
	campaignDecisionFormat string
)

var campaignCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a campaign from a manifest",
	Long:  "Create a simulation campaign from a .brotni/simulation.yaml manifest.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if campaignManifest == "" {
			return fmt.Errorf("--manifest is required")
		}

		manifest, err := loadCampaignManifest(campaignManifest)
		if err != nil {
			return err
		}
		req := manifestToCreateRequest(manifest)

		if cfg.DryRun {
			fmt.Printf("[dry-run] would create campaign %q with %d goal(s) and %d constraint(s)\n",
				req.Title, len(req.Goals), len(req.Constraints))
			return nil
		}

		client := api.NewClient(cfg.APIURL, cfg.Token, cfg.Debug)
		resp, err := client.CreateCampaign(context.Background(), req)
		if err != nil {
			return fmt.Errorf("creating campaign: %w", err)
		}

		if cfg.Output == "json" {
			return printer.PrintJSON(resp)
		}
		fmt.Println("Campaign created")
		fmt.Printf("  ID:              %s\n", resp.ID)
		fmt.Printf("  Title:           %s\n", resp.Title)
		fmt.Printf("  Status:          %s\n", resp.Status)
		fmt.Printf("  Scoring version: %d\n", resp.ActiveScoringVersion)
		return nil
	},
}

var campaignDiscoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover candidates for a campaign",
	Long: `Discover candidates declared in a manifest and register them with an
existing campaign. Label-based discovery (linked PRs/MRs) is delegated to the
CI/CD integration (brotni-github-action, brotni-gitlab-component).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if campaignID == "" {
			return fmt.Errorf("--id is required")
		}
		if campaignManifest == "" {
			return fmt.Errorf("--manifest is required")
		}
		manifest, err := loadCampaignManifest(campaignManifest)
		if err != nil {
			return err
		}
		candidates := manifest.Candidates.List
		fmt.Printf("%d manifest candidate(s) for campaign %s:\n", len(candidates), campaignID)
		for _, c := range candidates {
			fmt.Printf("  - %s (%s)\n", c.Name, c.SourceKind)
		}
		// NOTE: client-side candidate registration is not yet wired. Registering
		// these candidates with the studio (and label-based discovery of linked
		// PRs/MRs) is the next integration step. This command currently only
		// resolves and lists the manifest-declared candidates.
		fmt.Println("\nNote: candidate registration with the studio is not yet wired; " +
			"this command lists the manifest candidates only.")
		return nil
	},
}

var campaignStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show campaign status",
	RunE: func(cmd *cobra.Command, args []string) error {
		if campaignID == "" {
			return fmt.Errorf("--id is required")
		}
		client := api.NewClient(cfg.APIURL, cfg.Token, cfg.Debug)
		resp, err := client.GetCampaign(context.Background(), campaignID)
		if err != nil {
			return fmt.Errorf("getting campaign: %w", err)
		}
		if cfg.Output == "json" {
			return printer.PrintJSON(resp)
		}
		fmt.Println("Campaign")
		fmt.Printf("  ID:              %s\n", resp.ID)
		fmt.Printf("  Title:           %s\n", resp.Title)
		fmt.Printf("  Status:          %s\n", resp.Status)
		fmt.Printf("  Scoring version: %d\n", resp.ActiveScoringVersion)
		return nil
	},
}

var campaignCompareCmd = &cobra.Command{
	Use:   "compare",
	Short: "Compare candidate scorecards",
	RunE: func(cmd *cobra.Command, args []string) error {
		if campaignID == "" {
			return fmt.Errorf("--id is required")
		}
		client := api.NewClient(cfg.APIURL, cfg.Token, cfg.Debug)
		resp, err := client.GetScorecards(context.Background(), campaignID, campaignScoringVersion)
		if err != nil {
			return fmt.Errorf("getting scorecards: %w", err)
		}
		if cfg.Output == "json" {
			return printer.PrintJSON(resp)
		}
		fmt.Printf("Scorecards (campaign %s)\n", campaignID)
		fmt.Printf("  %-24s %-8s %-8s %s\n", "CANDIDATE", "SCORE", "BLOCK", "VERSION")
		for _, sc := range resp.Items {
			block := "pass"
			if !sc.PassedBlocking {
				block = "FAIL"
			}
			fmt.Printf("  %-24s %-8.2f %-8s %d\n", sc.CandidateID, sc.OverallScore, block, sc.ScoringVersion)
		}
		return nil
	},
}

var campaignDecisionCmd = &cobra.Command{
	Use:   "decision",
	Short: "Show the campaign decision report",
	RunE: func(cmd *cobra.Command, args []string) error {
		if campaignID == "" {
			return fmt.Errorf("--id is required")
		}
		client := api.NewClient(cfg.APIURL, cfg.Token, cfg.Debug)
		report, err := client.GetDecision(context.Background(), campaignID, campaignScoringVersion)
		if err != nil {
			return fmt.Errorf("getting decision: %w", err)
		}
		if cfg.Output == "json" {
			return printer.PrintJSON(report)
		}
		if campaignDecisionFormat == "md" {
			fmt.Print(renderDecisionMarkdown(report))
			return nil
		}
		fmt.Printf("Decision report (campaign %s, scoring v%d)\n", report.CampaignID, report.ScoringVersion)
		if report.WinnerCandidateID != "" {
			fmt.Printf("  Winner: %s\n", report.WinnerCandidateID)
		} else {
			fmt.Println("  Winner: none (no candidate passed all blocking constraints)")
		}
		fmt.Println("  Ranking:")
		for _, r := range report.Ranking {
			block := "pass"
			if !r.PassedBlocking {
				block = "FAIL"
			}
			fmt.Printf("    %d. %-24s %6.2f  [%s]\n", r.Rank, r.CandidateID, r.OverallScore, block)
		}
		fmt.Printf("  %s\n", report.Rationale)
		return nil
	},
}

func init() {
	campaignCmd.AddCommand(campaignCreateCmd)
	campaignCmd.AddCommand(campaignDiscoverCmd)
	campaignCmd.AddCommand(campaignStatusCmd)
	campaignCmd.AddCommand(campaignCompareCmd)
	campaignCmd.AddCommand(campaignDecisionCmd)

	campaignCreateCmd.Flags().StringVar(&campaignManifest, "manifest", "", "path to campaign manifest, e.g. .brotni/simulation.yaml (required)")

	campaignDiscoverCmd.Flags().StringVar(&campaignID, "id", "", "campaign ID (required)")
	campaignDiscoverCmd.Flags().StringVar(&campaignManifest, "manifest", "", "path to campaign manifest (required)")

	campaignStatusCmd.Flags().StringVar(&campaignID, "id", "", "campaign ID (required)")

	campaignCompareCmd.Flags().StringVar(&campaignID, "id", "", "campaign ID (required)")
	campaignCompareCmd.Flags().IntVar(&campaignScoringVersion, "scoring-version", 0, "scoring version (0 = active)")

	campaignDecisionCmd.Flags().StringVar(&campaignID, "id", "", "campaign ID (required)")
	campaignDecisionCmd.Flags().IntVar(&campaignScoringVersion, "scoring-version", 0, "scoring version (0 = active)")
	campaignDecisionCmd.Flags().StringVar(&campaignDecisionFormat, "format", "table", "output format for table mode: table or md")
}

func loadCampaignManifest(path string) (*validate.CampaignManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest %s: %w", path, err)
	}
	var m validate.CampaignManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest %s: %w", path, err)
	}
	output.PrintDebug(cfg.Debug, "loaded campaign manifest %q with %d goals", m.Campaign.Title, len(m.Goals))
	return &m, nil
}

func manifestToCreateRequest(m *validate.CampaignManifest) api.CampaignCreateRequest {
	req := api.CampaignCreateRequest{
		Title:       m.Campaign.Title,
		Environment: m.Simulation.Environment,
		Dataset:     m.Simulation.Dataset,
	}
	if w := m.Campaign.LinkedWorkItem; w != nil {
		req.WorkItem = &api.WorkItem{
			Provider: w.Provider,
			Type:     w.Type,
			Repo:     w.Repo,
			Number:   w.Number,
			URL:      w.URL,
		}
	}
	for _, g := range m.Goals {
		req.Goals = append(req.Goals, api.Goal{Name: g.Name, Metric: g.Metric, Weight: g.Weight, Direction: g.Direction})
	}
	for _, c := range m.Constraints {
		severity := c.Severity
		if severity == "" {
			severity = "blocking"
		}
		req.Constraints = append(req.Constraints, api.Constraint{
			Name: c.Name, Metric: c.Metric, Operator: c.Operator, Threshold: c.Threshold, Severity: severity,
		})
	}
	return req
}

func renderDecisionMarkdown(r *api.DecisionReport) string {
	out := fmt.Sprintf("### Brotni Simulation Decision\n\nCampaign: `%s` (scoring v%d)\n\n", r.CampaignID, r.ScoringVersion)
	if r.WinnerCandidateID != "" {
		out += fmt.Sprintf("**Winner:** `%s`\n\n", r.WinnerCandidateID)
	} else {
		out += "**Winner:** none — no candidate passed all blocking constraints\n\n"
	}
	out += "| Rank | Candidate | Score | Blocking |\n|------|-----------|-------|----------|\n"
	for _, rk := range r.Ranking {
		block := "✅ pass"
		if !rk.PassedBlocking {
			block = "❌ fail"
		}
		out += fmt.Sprintf("| %d | `%s` | %.2f | %s |\n", rk.Rank, rk.CandidateID, rk.OverallScore, block)
	}
	out += fmt.Sprintf("\n%s\n", r.Rationale)
	return out
}
