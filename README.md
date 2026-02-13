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

## Architecture

See [DESIGN.md](DESIGN.md) for the full system design.

## License

TBD
