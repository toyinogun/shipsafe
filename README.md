# ShipSafe

**Self-hosted AI code verification gateway.** Trust scores for every code change.

ShipSafe sits in your CI/CD pipeline between code generation and merge, running multi-layered verification on code changes (especially AI-generated code) to produce trust scores and verification reports.

## Why ShipSafe?

- **96% of developers don't fully trust AI-generated code** (Sonar 2026 Survey)
- **91% more time spent on code review** since AI adoption (Faros AI)
- **40% quality deficit projected for 2026** — more code entering pipelines than reviewers can validate

ShipSafe bridges the trust gap with automated, multi-layered verification.

## Key Features

- 🔒 **Self-hosted** — All analysis runs on your infrastructure. No code leaves your network.
- 🎯 **Trust Score** — 0-100 score with GREEN/YELLOW/RED rating on every PR
- 🔍 **5+ Static Analyzers** — Complexity, test coverage, secrets, dependencies, anti-patterns
- 🤖 **AI-Powered Review** (optional) — LLM-based semantic, logic, and convention analysis
- 🇪🇺 **EU Data Sovereignty** — GDPR-friendly, NIS2 compliance reporting
- ☸️ **Kubernetes-Native** — Helm chart, ArgoCD-ready, CloudNativePG integration
- 🔌 **Multi-Platform** — Forgejo, GitHub, GitLab

## Quick Start

```bash
# Scan a diff file
shipsafe scan --diff ./my-changes.diff

# Scan current directory against git HEAD
shipsafe scan .

# Run in CI mode (auto-detects environment, posts PR comment)
shipsafe ci

# Output as JSON
shipsafe scan . --format json
```

## Configuration

Create `.shipsafe.yml` in your repository root:

```yaml
version: "1"

thresholds:
  green: 80
  yellow: 50

analyzers:
  complexity:
    enabled: true
    threshold: 15
  secrets:
    enabled: true
  coverage:
    enabled: true

ai:
  enabled: false  # Enable for LLM-powered review
```

See `shipsafe.example.yml` for full configuration reference.

## Local Install

```bash
curl -fsSL -L https://github.com/toyinogun/shipsafe/releases/download/v0.3.0-alpha/shipsafe-linux-amd64 -o /usr/local/bin/shipsafe
chmod +x /usr/local/bin/shipsafe
shipsafe scan --diff <(git diff main)
```

## CI Integration

### GitHub Actions

Add `.github/workflows/shipsafe.yml` to your repo:

```yaml
name: ShipSafe Code Verification
on:
  pull_request:
    types: [opened, synchronize, reopened]

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Install ShipSafe
        run: |
          curl -fsSL -L https://github.com/toyinogun/shipsafe/releases/download/v0.3.0-alpha/shipsafe-linux-amd64 -o /usr/local/bin/shipsafe
          chmod +x /usr/local/bin/shipsafe

      - name: Run ShipSafe
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          SHIPSAFE_AI_API_KEY: ${{ secrets.SHIPSAFE_AI_API_KEY }}
        run: shipsafe ci
```

### Forgejo Actions

Add `.forgejo/workflows/shipsafe.yml` to your repo:

```yaml
name: ShipSafe Code Verification
on:
  pull_request:
    types: [opened, synchronize, reopened]

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Install ShipSafe
        run: |
          curl -fsSL -L https://github.com/toyinogun/shipsafe/releases/download/v0.3.0-alpha/shipsafe-linux-amd64 -o /usr/local/bin/shipsafe
          chmod +x /usr/local/bin/shipsafe

      - name: Run ShipSafe
        env:
          FORGEJO_TOKEN: ${{ secrets.FORGEJO_TOKEN }}
          GITEA_SERVER_URL: ${{ github.server_url }}
          SHIPSAFE_AI_API_KEY: ${{ secrets.SHIPSAFE_AI_API_KEY }}
        run: shipsafe ci
```

### Required Secrets

| Secret | Required | Description |
|--------|----------|-------------|
| `GITHUB_TOKEN` | Auto (GitHub) | Automatic on GitHub Actions, used for PR comments and commit status |
| `FORGEJO_TOKEN` | Yes (Forgejo) | Personal access token with `write:issue` scope |
| `SHIPSAFE_AI_API_KEY` | Optional | OpenAI-compatible API key for AI review (3-pass analysis) |

### What You Get

- Trust score (0-100) on every PR
- Commit status (green/red) for merge gating
- AI-powered code review catching logic bugs, null pointer issues, and missing edge cases
- Static analysis for secrets, complexity, test coverage, and bad patterns

## Architecture

See [DESIGN.md](DESIGN.md) for the full system design.

## License

TBD
