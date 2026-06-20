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

// CreateCampaign creates a simulation campaign from a parsed manifest.
func (c *Client) CreateCampaign(ctx context.Context, req CampaignCreateRequest) (*CampaignResponse, error) {
	var resp CampaignResponse
	if err := c.doRequest(ctx, http.MethodPost, "/v1/campaigns", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetCampaign fetches a campaign by ID.
func (c *Client) GetCampaign(ctx context.Context, id string) (*CampaignResponse, error) {
	var resp CampaignResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v1/campaigns/"+id, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetScorecards returns the per-candidate scorecards for a campaign. A
// scoringVersion of 0 means the campaign's active version.
func (c *Client) GetScorecards(ctx context.Context, campaignID string, scoringVersion int) (*ScorecardListResponse, error) {
	path := fmt.Sprintf("/v1/campaigns/%s/scorecards", campaignID)
	if scoringVersion > 0 {
		path = fmt.Sprintf("%s?scoringVersion=%d", path, scoringVersion)
	}
	var resp ScorecardListResponse
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetDecision returns the decision report for a campaign.
func (c *Client) GetDecision(ctx context.Context, campaignID string, scoringVersion int) (*DecisionReport, error) {
	path := fmt.Sprintf("/v1/campaigns/%s/decision", campaignID)
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
