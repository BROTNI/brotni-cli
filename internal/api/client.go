package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	token      string
	debug      bool
	httpClient *http.Client
}

func NewClient(baseURL, token string, debug bool) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		debug:   debug,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type CandidateSubmitRequest struct {
	Provider       string `json:"provider"`
	Repository     string `json:"repository"`
	Branch         string `json:"branch,omitempty"`
	CommitSHA      string `json:"commit_sha"`
	PRMRID         string `json:"pr_mr_id,omitempty"`
	ArtifactType   string `json:"artifact_type"`
	ArtifactURI    string `json:"artifact_uri"`
	ArtifactDigest string `json:"artifact_digest,omitempty"`
	SimulationSpec string `json:"simulation_spec,omitempty"`
	RecipeSpec     string `json:"recipe_spec,omitempty"`
	ContextSpec    string `json:"context_spec,omitempty"`
	CampaignID     string `json:"campaign_id,omitempty"`
	SourceKind     string `json:"source_kind,omitempty"`
}

type CandidateSubmitResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	URL    string `json:"url,omitempty"`
}

type SimulationRunRequest struct {
	CandidateID string `json:"candidate_id,omitempty"`
	DryRun      bool   `json:"dry_run"`
}

type SimulationRunResponse struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type SimulationStatusResponse struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Phase     string `json:"phase,omitempty"`
	StartedAt string `json:"started_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
	Message   string `json:"message,omitempty"`
}

type ReportExportResponse struct {
	Format  string `json:"format"`
	Content string `json:"content,omitempty"`
	URL     string `json:"url,omitempty"`
}

// --- Simulation Campaign -----------------------------------------------------

type WorkItem struct {
	Provider string `json:"provider"`
	Type     string `json:"type"`
	Repo     string `json:"repo,omitempty"`
	Number   int    `json:"number,omitempty"`
	URL      string `json:"url,omitempty"`
}

type Goal struct {
	Name      string  `json:"name"`
	Metric    string  `json:"metric"`
	Weight    float64 `json:"weight"`
	Direction string  `json:"direction"`
}

type Constraint struct {
	Name      string  `json:"name"`
	Metric    string  `json:"metric"`
	Operator  string  `json:"operator"`
	Threshold float64 `json:"threshold"`
	Severity  string  `json:"severity,omitempty"`
}

type CampaignCreateRequest struct {
	Title       string       `json:"title"`
	WorkItem    *WorkItem    `json:"workItem,omitempty"`
	Environment string       `json:"environment,omitempty"`
	Dataset     string       `json:"dataset,omitempty"`
	Goals       []Goal       `json:"goals,omitempty"`
	Constraints []Constraint `json:"constraints,omitempty"`
}

type CampaignResponse struct {
	ID                   string `json:"id"`
	Title                string `json:"title"`
	Status               string `json:"status"`
	ActiveScoringVersion int    `json:"activeScoringVersion"`
}

type SourceRef struct {
	Provider          string `json:"provider,omitempty"`
	Repository        string `json:"repository,omitempty"`
	ChangeRequestType string `json:"changeRequestType,omitempty"`
	ChangeRequestID   string `json:"changeRequestId,omitempty"`
	Branch            string `json:"branch,omitempty"`
	HeadSHA           string `json:"headSha,omitempty"`
	BaseSHA           string `json:"baseSha,omitempty"`
	URL               string `json:"url,omitempty"`
}

type ArtifactRef struct {
	Kind   string `json:"kind,omitempty"`
	URI    string `json:"uri,omitempty"`
	Digest string `json:"digest,omitempty"`
}

type ExecutionSpec struct {
	Command        []string          `json:"command"`
	Env            map[string]string `json:"env,omitempty"`
	MetricsPath    string            `json:"metricsPath,omitempty"`
	TimeoutSeconds int               `json:"timeoutSeconds,omitempty"`
}

type ChangeCandidateRequest struct {
	Name          string         `json:"name"`
	SourceKind    string         `json:"sourceKind"`
	SourceRef     SourceRef      `json:"sourceRef"`
	ArtifactRef   *ArtifactRef   `json:"artifactRef,omitempty"`
	RecipeRef     string         `json:"recipeRef,omitempty"`
	DiscoveredVia string         `json:"discoveredVia,omitempty"`
	Execution     *ExecutionSpec `json:"execution,omitempty"`
}

type RunOutcome struct {
	CandidateID   string `json:"candidateId"`
	CandidateName string `json:"candidateName"`
	Status        string `json:"status"`
	MetricCount   int    `json:"metricCount"`
	Error         string `json:"error,omitempty"`
}

type RunReport struct {
	CampaignID string         `json:"campaignId"`
	Runs       []RunOutcome   `json:"runs"`
	Decision   DecisionReport `json:"decision"`
}

type ChangeCandidateResponse struct {
	ID         string `json:"id"`
	CampaignID string `json:"campaignId"`
	Name       string `json:"name"`
	SourceKind string `json:"sourceKind"`
	Status     string `json:"status"`
}

type Ranking struct {
	Rank           int     `json:"rank"`
	CandidateID    string  `json:"candidateId"`
	OverallScore   float64 `json:"overallScore"`
	PassedBlocking bool    `json:"passedBlocking"`
}

type DecisionReport struct {
	CampaignID        string    `json:"campaignId"`
	ScoringVersion    int       `json:"scoringVersion"`
	WinnerCandidateID string    `json:"winnerCandidateId,omitempty"`
	Ranking           []Ranking `json:"ranking"`
	Rationale         string    `json:"rationale"`
}

type Scorecard struct {
	CandidateID    string  `json:"candidateId"`
	ScoringVersion int     `json:"scoringVersion"`
	OverallScore   float64 `json:"overallScore"`
	PassedBlocking bool    `json:"passedBlocking"`
}

type ScorecardListResponse struct {
	Items []Scorecard `json:"items"`
}

// campaignBase is the public API path prefix for the campaign surface. The
// simulation-engine serves these routes directly, acting as the studio.
const campaignBase = "/api/v1/campaigns"

// CreateCampaign creates a simulation campaign from a parsed manifest.
func (c *Client) CreateCampaign(ctx context.Context, req CampaignCreateRequest) (*CampaignResponse, error) {
	var resp CampaignResponse
	if err := c.doRequest(ctx, http.MethodPost, campaignBase, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetCampaign fetches a campaign by ID.
func (c *Client) GetCampaign(ctx context.Context, id string) (*CampaignResponse, error) {
	var resp CampaignResponse
	if err := c.doRequest(ctx, http.MethodGet, campaignBase+"/"+id, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// AddCandidate registers a change candidate with a campaign and returns the
// studio-minted candidate ID.
func (c *Client) AddCandidate(ctx context.Context, campaignID string, req ChangeCandidateRequest) (*ChangeCandidateResponse, error) {
	var resp ChangeCandidateResponse
	path := fmt.Sprintf("%s/%s/candidates", campaignBase, campaignID)
	if err := c.doRequest(ctx, http.MethodPost, path, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// IngestMetrics records a candidate's run metrics for a campaign.
func (c *Client) IngestMetrics(ctx context.Context, campaignID, candidateID string, metrics map[string]float64) error {
	path := fmt.Sprintf("%s/%s/runs/%s/metrics", campaignBase, campaignID, candidateID)
	body := map[string]any{"metrics": metrics}
	return c.doRequest(ctx, http.MethodPost, path, body, nil)
}

// GetScorecards returns the per-candidate scorecards for a campaign. A
// scoringVersion of 0 means the campaign's active version.
func (c *Client) GetScorecards(ctx context.Context, campaignID string, scoringVersion int) (*ScorecardListResponse, error) {
	path := fmt.Sprintf("%s/%s/scorecards", campaignBase, campaignID)
	if scoringVersion > 0 {
		path = fmt.Sprintf("%s?scoringVersion=%d", path, scoringVersion)
	}
	var resp ScorecardListResponse
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// RunCampaign triggers execution of the campaign's candidates (those with an
// execution spec) and returns the run outcomes plus the resulting decision.
func (c *Client) RunCampaign(ctx context.Context, campaignID string) (*RunReport, error) {
	var resp RunReport
	path := fmt.Sprintf("%s/%s/run", campaignBase, campaignID)
	if err := c.doRequest(ctx, http.MethodPost, path, map[string]any{}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetDecision returns the decision report for a campaign.
func (c *Client) GetDecision(ctx context.Context, campaignID string, scoringVersion int) (*DecisionReport, error) {
	path := fmt.Sprintf("%s/%s/decision", campaignBase, campaignID)
	if scoringVersion > 0 {
		path = fmt.Sprintf("%s?scoringVersion=%d", path, scoringVersion)
	}
	var resp DecisionReport
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SubmitCandidate submits a simulation candidate to the Brotni platform.
// Returns a mock response when the backend is not yet configured.
func (c *Client) SubmitCandidate(ctx context.Context, req CandidateSubmitRequest) (*CandidateSubmitResponse, error) {
	var resp CandidateSubmitResponse
	if err := c.doRequest(ctx, http.MethodPost, "/v1/candidates", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// RunSimulation triggers a new simulation run.
func (c *Client) RunSimulation(ctx context.Context, req SimulationRunRequest) (*SimulationRunResponse, error) {
	var resp SimulationRunResponse
	if err := c.doRequest(ctx, http.MethodPost, "/v1/simulations", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetSimulationStatus retrieves the current status of a simulation run.
func (c *Client) GetSimulationStatus(ctx context.Context, simulationID string) (*SimulationStatusResponse, error) {
	var resp SimulationStatusResponse
	path := fmt.Sprintf("/v1/simulations/%s", simulationID)
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ExportReport requests a report export for a simulation run.
func (c *Client) ExportReport(ctx context.Context, simulationID, format string) (*ReportExportResponse, error) {
	var resp ReportExportResponse
	path := fmt.Sprintf("/v1/simulations/%s/report?format=%s", simulationID, format)
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) doRequest(ctx context.Context, method, path string, body, out interface{}) error {
	var reqBody bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&reqBody).Encode(body); err != nil {
			return fmt.Errorf("encoding request: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, &reqBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "brotni-cli/1.0")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if jsonErr := json.NewDecoder(resp.Body).Decode(&apiErr); jsonErr == nil && apiErr.Message != "" {
			return fmt.Errorf("API error %d: %s", resp.StatusCode, apiErr.Message)
		}
		return fmt.Errorf("API error: HTTP %d", resp.StatusCode)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}
