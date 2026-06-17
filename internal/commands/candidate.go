package commands

import (
	"context"
	"fmt"

	"github.com/BROTNI/brotni-cli/internal/api"
	"github.com/BROTNI/brotni-cli/internal/output"
	"github.com/spf13/cobra"
)

var candidateCmd = &cobra.Command{
	Use:   "candidate",
	Short: "Manage simulation candidates",
}

var (
	candidateProvider       string
	candidateRepo           string
	candidateBranch         string
	candidateCommit         string
	candidatePRMRID         string
	candidateArtifactType   string
	candidateArtifactURI    string
	candidateArtifactDigest string
	candidateSimulation     string
	candidateRecipe         string
	candidateContext        string
)

var candidateSubmitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit a simulation candidate",
	Long: `Submit a candidate for Brotni simulation.

Bundles a source reference, artifact, and spec files into a single submission
that the Brotni platform will schedule and execute.

Supported providers: github, gitlab, bitbucket (and any custom provider).`,
	Example: `  brotni candidate submit \
    --provider github \
    --repo owner/repo \
    --branch main \
    --commit abc123def456 \
    --artifact-type image \
    --artifact-uri registry.example.com/myapp:latest \
    --artifact-digest sha256:abc123... \
    --simulation .brotni/simulation.yaml \
    --recipe .brotni/runtime.yaml \
    --context .brotni/context.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if candidateProvider == "" {
			return fmt.Errorf("--provider is required")
		}
		if candidateRepo == "" {
			return fmt.Errorf("--repo is required")
		}
		if candidateCommit == "" {
			return fmt.Errorf("--commit is required")
		}
		if candidateArtifactURI == "" {
			return fmt.Errorf("--artifact-uri is required")
		}

		output.PrintDebug(cfg.Debug, "submitting candidate: provider=%s repo=%s commit=%s",
			candidateProvider, candidateRepo, candidateCommit)

		if cfg.DryRun {
			fmt.Println("[dry-run] would submit candidate:")
			fmt.Printf("  provider:        %s\n", candidateProvider)
			fmt.Printf("  repository:      %s\n", candidateRepo)
			if candidateBranch != "" {
				fmt.Printf("  branch:          %s\n", candidateBranch)
			}
			fmt.Printf("  commit:          %s\n", candidateCommit)
			if candidatePRMRID != "" {
				fmt.Printf("  pr/mr id:        %s\n", candidatePRMRID)
			}
			fmt.Printf("  artifact type:   %s\n", candidateArtifactType)
			fmt.Printf("  artifact uri:    %s\n", candidateArtifactURI)
			if candidateArtifactDigest != "" {
				fmt.Printf("  artifact digest: %s\n", candidateArtifactDigest)
			}
			if candidateSimulation != "" {
				fmt.Printf("  simulation spec: %s\n", candidateSimulation)
			}
			if candidateRecipe != "" {
				fmt.Printf("  recipe:          %s\n", candidateRecipe)
			}
			if candidateContext != "" {
				fmt.Printf("  context:         %s\n", candidateContext)
			}
			return nil
		}

		client := api.NewClient(cfg.APIURL, cfg.Token, cfg.Debug)

		resp, err := client.SubmitCandidate(context.Background(), api.CandidateSubmitRequest{
			Provider:       candidateProvider,
			Repository:     candidateRepo,
			Branch:         candidateBranch,
			CommitSHA:      candidateCommit,
			PRMRID:         candidatePRMRID,
			ArtifactType:   candidateArtifactType,
			ArtifactURI:    candidateArtifactURI,
			ArtifactDigest: candidateArtifactDigest,
			SimulationSpec: candidateSimulation,
			RecipeSpec:     candidateRecipe,
			ContextSpec:    candidateContext,
		})
		if err != nil {
			return fmt.Errorf("submitting candidate: %w", err)
		}

		if cfg.Output == "json" {
			return printer.PrintJSON(resp)
		}

		fmt.Println("Candidate submitted successfully")
		fmt.Printf("  ID:     %s\n", resp.ID)
		fmt.Printf("  Status: %s\n", resp.Status)
		if resp.URL != "" {
			fmt.Printf("  URL:    %s\n", resp.URL)
		}

		return nil
	},
}

func init() {
	candidateCmd.AddCommand(candidateSubmitCmd)

	f := candidateSubmitCmd.Flags()
	f.StringVar(&candidateProvider, "provider", "", "source provider: github, gitlab, bitbucket (required)")
	f.StringVar(&candidateRepo, "repo", "", "repository in owner/name format (required)")
	f.StringVar(&candidateBranch, "branch", "", "source branch")
	f.StringVar(&candidateCommit, "commit", "", "commit SHA (required)")
	f.StringVar(&candidatePRMRID, "pr", "", "pull request or merge request ID")
	f.StringVar(&candidateArtifactType, "artifact-type", "image", "artifact type: image, binary, archive")
	f.StringVar(&candidateArtifactURI, "artifact-uri", "", "artifact URI, e.g. registry.example.com/image:tag (required)")
	f.StringVar(&candidateArtifactDigest, "artifact-digest", "", "artifact digest, e.g. sha256:abc123...")
	f.StringVar(&candidateSimulation, "simulation", "", "path to simulation spec file")
	f.StringVar(&candidateRecipe, "recipe", "", "path to execution recipe file")
	f.StringVar(&candidateContext, "context", "", "path to context definition file")
}
