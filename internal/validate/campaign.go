package validate

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// CampaignManifest is the structure of the portable .brotni/simulation.yaml
// file. It declares a campaign, the goals/constraints used to compare
// candidates, and how candidates are discovered. See schemas/campaign.schema.json.
type CampaignManifest struct {
	Version     int                  `yaml:"version"`
	Campaign    CampaignMeta         `yaml:"campaign"`
	Simulation  CampaignSimulation   `yaml:"simulation"`
	Constraints []CampaignConstraint `yaml:"constraints,omitempty"`
	Goals       []CampaignGoal       `yaml:"goals"`
	Candidates  CampaignCandidates   `yaml:"candidates,omitempty"`
}

type CampaignMeta struct {
	Title          string               `yaml:"title"`
	Description    string               `yaml:"description,omitempty"`
	LinkedWorkItem *CampaignWorkItem    `yaml:"linkedWorkItem,omitempty"`
}

type CampaignWorkItem struct {
	Provider string `yaml:"provider"`
	Type     string `yaml:"type"`
	Repo     string `yaml:"repo,omitempty"`
	Number   int    `yaml:"number,omitempty"`
	URL      string `yaml:"url,omitempty"`
}

type CampaignSimulation struct {
	Environment string `yaml:"environment,omitempty"`
	Dataset     string `yaml:"dataset,omitempty"`
}

type CampaignConstraint struct {
	Name      string  `yaml:"name"`
	Metric    string  `yaml:"metric"`
	Operator  string  `yaml:"operator"`
	Threshold float64 `yaml:"threshold"`
	Severity  string  `yaml:"severity,omitempty"`
}

type CampaignGoal struct {
	Name      string  `yaml:"name"`
	Metric    string  `yaml:"metric"`
	Weight    float64 `yaml:"weight"`
	Direction string  `yaml:"direction"`
}

type CampaignCandidates struct {
	Discovery *CampaignDiscovery `yaml:"discovery,omitempty"`
	List      []CampaignCandidate `yaml:"list,omitempty"`
}

type CampaignDiscovery struct {
	Mode     string `yaml:"mode,omitempty"`
	Label    string `yaml:"label,omitempty"`
	WorkItem any    `yaml:"workItem,omitempty"`
}

type CampaignCandidate struct {
	Name       string                   `yaml:"name"`
	SourceKind string                   `yaml:"sourceKind"`
	RecipeRef  string                   `yaml:"recipeRef,omitempty"`
	Source     *CampaignCandidateSource `yaml:"source,omitempty"`
	Artifact   *CampaignCandidateArtifact `yaml:"artifact,omitempty"`
}

type CampaignCandidateSource struct {
	Provider          string `yaml:"provider,omitempty"`
	Repository        string `yaml:"repository,omitempty"`
	ChangeRequestType string `yaml:"changeRequestType,omitempty"`
	ChangeRequestID   string `yaml:"changeRequestId,omitempty"`
	Branch            string `yaml:"branch,omitempty"`
	HeadSha           string `yaml:"headSha,omitempty"`
	BaseSha           string `yaml:"baseSha,omitempty"`
	URL               string `yaml:"url,omitempty"`
}

type CampaignCandidateArtifact struct {
	Kind   string `yaml:"kind,omitempty"`
	URI    string `yaml:"uri,omitempty"`
	Digest string `yaml:"digest,omitempty"`
}

var validDirections = map[string]bool{"minimize": true, "maximize": true}
var validOperators = map[string]bool{"<": true, "<=": true, ">": true, ">=": true, "==": true}
var validSourceKinds = map[string]bool{"git_change": true, "container_image": true, "config_bundle": true}

// ValidateCampaign validates a .brotni/simulation.yaml campaign manifest.
func ValidateCampaign(file string) (*ValidationResult, error) {
	return validateFile(file, "campaign", validateCampaignContent)
}

func validateCampaignContent(data []byte) (errors, warnings []string) {
	var m CampaignManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return []string{fmt.Sprintf("invalid YAML: %v", err)}, nil
	}

	if m.Version == 0 {
		errors = append(errors, "missing required field: version")
	} else if m.Version != 1 {
		errors = append(errors, fmt.Sprintf("unsupported version %d, expected 1", m.Version))
	}

	if m.Campaign.Title == "" {
		errors = append(errors, "missing required field: campaign.title")
	}

	if len(m.Goals) == 0 {
		errors = append(errors, "at least one goal is required under goals")
	}
	for i, g := range m.Goals {
		if g.Name == "" {
			errors = append(errors, fmt.Sprintf("goals[%d]: missing required field: name", i))
		}
		if g.Metric == "" {
			errors = append(errors, fmt.Sprintf("goals[%d]: missing required field: metric", i))
		}
		if g.Weight <= 0 {
			errors = append(errors, fmt.Sprintf("goals[%d]: weight must be greater than 0", i))
		}
		if !validDirections[g.Direction] {
			errors = append(errors, fmt.Sprintf("goals[%d]: direction must be \"minimize\" or \"maximize\"", i))
		}
	}

	for i, c := range m.Constraints {
		if c.Name == "" {
			errors = append(errors, fmt.Sprintf("constraints[%d]: missing required field: name", i))
		}
		if c.Metric == "" {
			errors = append(errors, fmt.Sprintf("constraints[%d]: missing required field: metric", i))
		}
		if !validOperators[c.Operator] {
			errors = append(errors, fmt.Sprintf("constraints[%d]: operator must be one of < <= > >= ==", i))
		}
		if c.Severity != "" && c.Severity != "blocking" && c.Severity != "warning" {
			warnings = append(warnings, fmt.Sprintf("constraints[%d]: unexpected severity %q (expected blocking or warning)", i, c.Severity))
		}
	}

	for i, cand := range m.Candidates.List {
		if cand.Name == "" {
			errors = append(errors, fmt.Sprintf("candidates.list[%d]: missing required field: name", i))
		}
		if !validSourceKinds[cand.SourceKind] {
			errors = append(errors, fmt.Sprintf("candidates.list[%d]: sourceKind must be git_change, container_image, or config_bundle", i))
		}
	}

	if m.Campaign.LinkedWorkItem == nil && m.Candidates.Discovery == nil && len(m.Candidates.List) == 0 {
		warnings = append(warnings, "no candidates declared and no discovery configured — the campaign will start empty")
	}

	return errors, warnings
}
