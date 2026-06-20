# brotni-cli

**brotni-cli** is the official command-line tool for [Brotni](https://brotni.io)-compatible simulation workflows.

Submit candidates, validate specs, trigger simulations, inspect status, and export reports — from local development, CI/CD pipelines (GitHub Actions, GitLab CI), or any automation environment.

## Overview

```
brotni <command> [subcommand] [flags]

Commands:
  version             Print version information
  validate recipe     Validate a runtime execution recipe file
  validate context    Validate a context definition file
  validate simulation Validate a simulation spec file
  candidate submit    Submit a simulation candidate
  simulation run      Trigger a simulation run
  simulation status   Get simulation run status
  report export       Export a simulation report
```

## Installation

### Download a pre-built binary

Download the latest release for your platform from the [Releases](https://github.com/BROTNI/brotni-cli/releases) page.

```bash
# Linux (amd64)
curl -L https://github.com/BROTNI/brotni-cli/releases/latest/download/brotni-linux-amd64 -o brotni
chmod +x brotni
sudo mv brotni /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/BROTNI/brotni-cli/releases/latest/download/brotni-darwin-arm64 -o brotni
chmod +x brotni
sudo mv brotni /usr/local/bin/
```

### Build from source

Requires Go 1.22+.

```bash
git clone https://github.com/BROTNI/brotni-cli.git
cd brotni-cli
make build
./brotni version
```

## Configuration

brotni-cli reads configuration in the following order (highest priority first):

1. Command-line flags
2. Environment variables
3. Config file (`.brotni.yaml` in the current directory or `$HOME`)

### Environment variables

| Variable        | Description                                    | Default                    |
|-----------------|------------------------------------------------|----------------------------|
| `BROTNI_API_URL` | Base URL of the Brotni API                    | `https://api.brotni.io`    |
| `BROTNI_TOKEN`  | API authentication token                       | *(required for API calls)* |

### Config file

Create a `.brotni.yaml` in your project root or home directory:

```yaml
api_url: https://api.brotni.io
output: table   # table or json
```

Tokens should be supplied via `BROTNI_TOKEN` and never committed to source control.

## Spec files

Brotni uses three spec file types, all written in YAML:

### Recipe (`runtime.yaml`)

Defines the container image and execution environment.

```yaml
apiVersion: brotni.io/v1
kind: Recipe
metadata:
  name: my-service-recipe
runtime:
  image: registry.example.com/my-service:latest
  command: ["./run-tests.sh"]
  env:
    LOG_LEVEL: "info"
  resources:
    cpu: "1"
    memory: "1Gi"
```

### Context (`context.yaml`)

Declares the inputs and outputs of a simulation.

```yaml
apiVersion: brotni.io/v1
kind: Context
metadata:
  name: my-service-context
inputs:
  - name: artifact_image
    type: artifact
    required: true
outputs:
  - name: test_results
    type: report
```

### Simulation (`simulation.yaml`)

Wires a recipe and context together.

```yaml
apiVersion: brotni.io/v1
kind: Simulation
metadata:
  name: my-service-integration
spec:
  recipeRef: my-service-recipe
  contextRef: my-service-context
  timeout: 30m
```

See the [`examples/`](./examples/) directory for complete examples and [`schemas/`](./schemas/) for JSON schemas.

## Validating specs

Validate spec files locally before submitting — non-zero exit code on failure, making it CI-friendly.

```bash
brotni validate recipe   .brotni/runtime.yaml
brotni validate context  .brotni/context.yaml
brotni validate simulation .brotni/simulation.yaml
```

Machine-readable output:

```bash
brotni validate recipe .brotni/runtime.yaml --output json
```

## Submitting a candidate

A *candidate* bundles a source reference, container artifact, and spec files into a single submission for simulation.

```bash
brotni candidate submit \
  --provider github \
  --repo owner/repo \
  --branch main \
  --commit abc123def456 \
  --artifact-type image \
  --artifact-uri registry.example.com/myapp:1.2.3 \
  --artifact-digest sha256:abc123... \
  --simulation .brotni/simulation.yaml \
  --recipe .brotni/runtime.yaml \
  --context .brotni/context.yaml
```

### Flags

| Flag               | Description                                      | Required |
|--------------------|--------------------------------------------------|----------|
| `--provider`       | Source provider: `github`, `gitlab`, `bitbucket` | yes      |
| `--repo`           | Repository in `owner/name` format                | yes      |
| `--branch`         | Source branch                                    | no       |
| `--commit`         | Commit SHA                                       | yes      |
| `--pr`             | Pull request or merge request ID                 | no       |
| `--artifact-type`  | Artifact type: `image`, `binary`, `archive`      | no       |
| `--artifact-uri`   | Artifact URI                                     | yes      |
| `--artifact-digest`| Artifact digest, e.g. `sha256:...`               | no       |
| `--simulation`     | Path to simulation spec file                     | no       |
| `--recipe`         | Path to execution recipe file                    | no       |
| `--context`        | Path to context definition file                  | no       |

## Simulation campaigns

A **campaign** groups multiple change candidates (pull requests, OCI images,
config bundles) under one work item and compares them on the same goals and
constraints. Declare it in `.brotni/simulation.yaml`:

```yaml
version: 1
campaign:
  title: "Optimize routing strategy"
  linkedWorkItem: { provider: github, type: issue, number: 482 }
goals:
  - { name: latency, metric: p99_latency_ms, weight: 0.5, direction: minimize }
  - { name: throughput, metric: throughput_rps, weight: 0.5, direction: maximize }
constraints:
  - { name: error_rate, metric: error_rate, operator: "<", threshold: 0.2, severity: blocking }
```

```bash
# Validate the manifest (non-zero exit on failure — CI-friendly)
brotni validate campaign .brotni/simulation.yaml

# Create the campaign and register the manifest's candidates.
# The output prints each candidate's name -> studio-minted ID.
brotni campaign create --manifest .brotni/simulation.yaml

# Execute candidates and auto-collect the metrics they emit. Requires the
# studio's command runner (BROTNI_ENABLE_COMMAND_RUNNER=1) and candidates
# registered with a --command that writes JSON to $BROTNI_METRICS_PATH:
brotni campaign add-candidate --id camp-123 --name fast \
  --command 'printf "{\"p99_latency_ms\":120}" > "$BROTNI_METRICS_PATH"'
brotni campaign run --id camp-123

# Or hand-feed metrics directly (demo/test affordance):
brotni campaign ingest --id camp-123 --candidate cand-456 \
  --metrics p99_latency_ms=120,cost_per_1k_requests=8,resilience_score=70

# Compare candidate scorecards and read the decision
brotni campaign compare  --id camp-123
brotni campaign decision --id camp-123 --format md
```

The CLI talks to the campaign API at `/api/v1/campaigns`; point `BROTNI_API_URL`
at a running studio (the simulation-engine serves this surface directly).

Scoring is **comparative** (each goal min-max normalised across candidates) and
**versioned** — re-weighting goals creates a new scoring version without
overwriting prior scorecards. Pass `--scoring-version N` to `compare`/`decision`
to inspect a specific interpretation. See the full example in
[`examples/campaign/`](./examples/campaign/).

## Triggering and monitoring simulations

```bash
# Trigger a simulation
brotni simulation run --candidate cand-abc123

# Check status
brotni simulation status --id sim-abc123

# Poll in a script
while true; do
  STATUS=$(brotni simulation status --id sim-abc123 --output json | jq -r '.status')
  [ "$STATUS" = "completed" ] && break
  sleep 10
done
```

## Exporting reports

```bash
brotni report export --simulation sim-abc123 --format json
brotni report export --simulation sim-abc123 --format html
```

## Using in CI/CD

### GitHub Actions

```yaml
- name: Validate specs
  run: |
    brotni validate recipe   .brotni/runtime.yaml
    brotni validate context  .brotni/context.yaml
    brotni validate simulation .brotni/simulation.yaml

- name: Submit candidate
  env:
    BROTNI_TOKEN: ${{ secrets.BROTNI_TOKEN }}
  run: |
    brotni candidate submit \
      --provider github \
      --repo ${{ github.repository }} \
      --branch ${{ github.ref_name }} \
      --commit ${{ github.sha }} \
      --artifact-uri ghcr.io/${{ github.repository }}:${{ github.sha }} \
      --simulation .brotni/simulation.yaml \
      --recipe .brotni/runtime.yaml \
      --context .brotni/context.yaml
```

### GitLab CI

```yaml
validate-specs:
  script:
    - brotni validate recipe   .brotni/runtime.yaml
    - brotni validate context  .brotni/context.yaml
    - brotni validate simulation .brotni/simulation.yaml

submit-candidate:
  variables:
    BROTNI_TOKEN: $BROTNI_TOKEN
  script:
    - brotni candidate submit
        --provider gitlab
        --repo $CI_PROJECT_PATH
        --branch $CI_COMMIT_BRANCH
        --commit $CI_COMMIT_SHA
        --artifact-uri $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA
        --simulation .brotni/simulation.yaml
        --recipe .brotni/runtime.yaml
        --context .brotni/context.yaml
```

## Dry-run mode

Use `--dry-run` to validate and preview any operation without executing it:

```bash
brotni candidate submit --dry-run \
  --provider github \
  --repo owner/repo \
  --commit abc123 \
  --artifact-uri registry.example.com/myapp:latest
```

## Global flags

| Flag          | Description                          |
|---------------|--------------------------------------|
| `--config`    | Path to config file                  |
| `--output`, `-o` | Output format: `table` or `json` |
| `--debug`     | Enable debug output to stderr        |
| `--dry-run`   | Preview without executing            |

## Security

- API tokens are read from `BROTNI_TOKEN`. Never hardcode tokens in spec files or scripts.
- Use short-lived tokens scoped to the minimum required permissions.
- Artifact digests (`--artifact-digest sha256:...`) are recommended to ensure integrity.
- No production endpoints are hardcoded. The API URL is always configurable via `BROTNI_API_URL`.
- brotni-cli does not store credentials on disk.

## License

Apache License 2.0. See [LICENSE](./LICENSE).
