package commands

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

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
		ctx := context.Background()
		resp, err := client.CreateCampaign(ctx, req)
		if err != nil {
			return fmt.Errorf("creating campaign: %w", err)
		}

		// Register the explicit candidates declared in the manifest. `create`
		// owns explicit candidate registration; dynamic discovery (label /
		// linked PRs) is handled separately by the CI integrations.
		type registered struct {
			Name        string `json:"name"`
			CandidateID string `json:"candidateId"`
		}
		var regs []registered
		for _, c := range manifest.Candidates.List {
			cr, err := client.AddCandidate(ctx, resp.ID, manifestCandidateToRequest(c))
			if err != nil {
				return fmt.Errorf("registering candidate %q: %w", c.Name, err)
			}
			regs = append(regs, registered{Name: c.Name, CandidateID: cr.ID})
		}

		if cfg.Output == "json" {
			return printer.PrintJSON(map[string]any{"campaign": resp, "candidates": regs})
		}
		fmt.Println("Campaign created")
		fmt.Printf("  ID:              %s\n", resp.ID)
		fmt.Printf("  Title:           %s\n", resp.Title)
		fmt.Printf("  Status:          %s\n", resp.Status)
		fmt.Printf("  Scoring version: %d\n", resp.ActiveScoringVersion)
		if len(regs) > 0 {
			fmt.Printf("  Registered %d candidate(s):\n", len(regs))
			for _, r := range regs {
				fmt.Printf("    %-22s %s\n", r.Name, r.CandidateID)
			}
		}
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
		// Explicit candidates from candidates.list are registered by
		// `campaign create`. This command surfaces dynamic discovery (label /
		// linked PRs), which is driven by the CI integrations and not yet wired
		// for direct studio registration here.
		mode := ""
		if manifest.Candidates.Discovery != nil {
			mode = manifest.Candidates.Discovery.Mode
		}
		if mode != "" && mode != "manifest" {
			fmt.Printf("\nDynamic discovery mode %q is handled by the CI integrations "+
				"(brotni-github-action / brotni-gitlab-component) and is not registered here.\n", mode)
		} else {
			fmt.Println("\nNote: explicit candidates are registered by `brotni campaign create`.")
		}
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

var (
	ingestCandidateID string
	ingestMetrics     string

	addName       string
	addSourceKind string
	addRecipe     string
	addProvider   string
	addRepo       string
	addCommit     string
	addBranch     string
	addPR         string
	addArtifactURI    string
	addArtifactDigest string
	addArtifactKind   string
	addDiscoveredVia  string
)

var campaignAddCandidateCmd = &cobra.Command{
	Use:   "add-candidate",
	Short: "Register a single candidate with an existing campaign",
	Long: `Register one change candidate (a PR/MR, OCI image, or config bundle) with an
existing campaign. Used by CI integrations to submit the current build.

Registration is idempotent by candidate name: re-running for the same name
updates the existing candidate instead of creating a duplicate.`,
	Example: `  brotni campaign add-candidate --id camp-123 --name pr-501 \
    --source-kind container_image --provider github --repo owner/repo \
    --commit "$GITHUB_SHA" --artifact-uri ghcr.io/owner/repo \
    --artifact-digest sha256:...`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if campaignID == "" {
			return fmt.Errorf("--id (campaign) is required")
		}
		if addName == "" {
			return fmt.Errorf("--name is required")
		}
		req := api.ChangeCandidateRequest{
			Name:          addName,
			SourceKind:    addSourceKind,
			RecipeRef:     addRecipe,
			DiscoveredVia: addDiscoveredVia,
			SourceRef: api.SourceRef{
				Provider:        addProvider,
				Repository:      addRepo,
				ChangeRequestID: addPR,
				Branch:          addBranch,
				HeadSHA:         addCommit,
			},
		}
		if addArtifactURI != "" || addArtifactDigest != "" {
			req.ArtifactRef = &api.ArtifactRef{Kind: addArtifactKind, URI: addArtifactURI, Digest: addArtifactDigest}
		}
		if req.SourceKind == "" {
			if req.ArtifactRef != nil {
				req.SourceKind = "container_image"
			} else {
				req.SourceKind = "git_change"
			}
		}
		if cfg.DryRun {
			fmt.Printf("[dry-run] would register candidate %q (%s) with campaign %s\n", addName, req.SourceKind, campaignID)
			return nil
		}
		client := api.NewClient(cfg.APIURL, cfg.Token, cfg.Debug)
		resp, err := client.AddCandidate(context.Background(), campaignID, req)
		if err != nil {
			return fmt.Errorf("registering candidate: %w", err)
		}
		if cfg.Output == "json" {
			return printer.PrintJSON(resp)
		}
		fmt.Println("Candidate registered")
		fmt.Printf("  Campaign:  %s\n", campaignID)
		fmt.Printf("  Candidate: %s\n", resp.ID)
		fmt.Printf("  Name:      %s\n", resp.Name)
		fmt.Printf("  Status:    %s\n", resp.Status)
		return nil
	},
}

var campaignIngestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Ingest run metrics for a candidate",
	Long: `Record a candidate's run metrics so they can be scored and compared.

This is a demo/test affordance: in production, metrics are produced by
simulation runs against the context, not hand-fed via the CLI.`,
	Example: `  brotni campaign ingest --id camp-123 --candidate cand-456 \
    --metrics p99_latency_ms=100,throughput_rps=2000,error_rate=0.1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if campaignID == "" {
			return fmt.Errorf("--id is required")
		}
		if ingestCandidateID == "" {
			return fmt.Errorf("--candidate is required")
		}
		metrics, err := parseMetrics(ingestMetrics)
		if err != nil {
			return err
		}
		if cfg.DryRun {
			fmt.Printf("[dry-run] would ingest %d metric(s) for candidate %s\n", len(metrics), ingestCandidateID)
			return nil
		}
		client := api.NewClient(cfg.APIURL, cfg.Token, cfg.Debug)
		if err := client.IngestMetrics(context.Background(), campaignID, ingestCandidateID, metrics); err != nil {
			return fmt.Errorf("ingesting metrics: %w", err)
		}
		fmt.Printf("Ingested %d metric(s) for candidate %s\n", len(metrics), ingestCandidateID)
		return nil
	},
}

func init() {
	campaignCmd.AddCommand(campaignCreateCmd)
	campaignCmd.AddCommand(campaignDiscoverCmd)
	campaignCmd.AddCommand(campaignAddCandidateCmd)
	campaignCmd.AddCommand(campaignStatusCmd)
	campaignCmd.AddCommand(campaignIngestCmd)
	campaignCmd.AddCommand(campaignCompareCmd)
	campaignCmd.AddCommand(campaignDecisionCmd)

	campaignCreateCmd.Flags().StringVar(&campaignManifest, "manifest", "", "path to campaign manifest, e.g. .brotni/simulation.yaml (required)")

	campaignDiscoverCmd.Flags().StringVar(&campaignID, "id", "", "campaign ID (required)")
	campaignDiscoverCmd.Flags().StringVar(&campaignManifest, "manifest", "", "path to campaign manifest (required)")

	campaignIngestCmd.Flags().StringVar(&campaignID, "id", "", "campaign ID (required)")
	campaignIngestCmd.Flags().StringVar(&ingestCandidateID, "candidate", "", "candidate ID (required)")
	campaignIngestCmd.Flags().StringVar(&ingestMetrics, "metrics", "", "comma-separated metric=value pairs (required)")

	af := campaignAddCandidateCmd.Flags()
	af.StringVar(&campaignID, "id", "", "campaign ID (required)")
	af.StringVar(&addName, "name", "", "candidate name — stable key for idempotent re-registration, e.g. pr-501 (required)")
	af.StringVar(&addSourceKind, "source-kind", "", "git_change, container_image, or config_bundle (inferred when empty)")
	af.StringVar(&addRecipe, "recipe", "", "recipe reference for this candidate")
	af.StringVar(&addProvider, "provider", "", "source provider: github, gitlab, ...")
	af.StringVar(&addRepo, "repo", "", "repository in owner/name format")
	af.StringVar(&addCommit, "commit", "", "head commit SHA")
	af.StringVar(&addBranch, "branch", "", "source branch")
	af.StringVar(&addPR, "pr", "", "pull/merge request ID")
	af.StringVar(&addArtifactURI, "artifact-uri", "", "artifact URI, e.g. ghcr.io/org/image")
	af.StringVar(&addArtifactDigest, "artifact-digest", "", "artifact digest, e.g. sha256:...")
	af.StringVar(&addArtifactKind, "artifact-kind", "oci-image", "artifact kind")
	af.StringVar(&addDiscoveredVia, "discovered-via", "cli", "how the candidate was discovered: cli, label, manifest")

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

func manifestCandidateToRequest(c validate.CampaignCandidate) api.ChangeCandidateRequest {
	req := api.ChangeCandidateRequest{
		Name:          c.Name,
		SourceKind:    c.SourceKind,
		RecipeRef:     c.RecipeRef,
		DiscoveredVia: "manifest",
	}
	if s := c.Source; s != nil {
		req.SourceRef = api.SourceRef{
			Provider:          s.Provider,
			Repository:        s.Repository,
			ChangeRequestType: s.ChangeRequestType,
			ChangeRequestID:   s.ChangeRequestID,
			Branch:            s.Branch,
			HeadSHA:           s.HeadSha,
			BaseSHA:           s.BaseSha,
			URL:               s.URL,
		}
	}
	if a := c.Artifact; a != nil {
		req.ArtifactRef = &api.ArtifactRef{Kind: a.Kind, URI: a.URI, Digest: a.Digest}
	}
	return req
}

// parseMetrics parses "k1=v1,k2=v2" into a metric map.
func parseMetrics(s string) (map[string]float64, error) {
	out := map[string]float64{}
	if strings.TrimSpace(s) == "" {
		return nil, fmt.Errorf("--metrics is required (e.g. p99_latency_ms=100,error_rate=0.1)")
	}
	for _, pair := range strings.Split(s, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid metric %q, expected name=value", pair)
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(kv[1]), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value for metric %q: %w", strings.TrimSpace(kv[0]), err)
		}
		out[strings.TrimSpace(kv[0])] = v
	}
	return out, nil
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
