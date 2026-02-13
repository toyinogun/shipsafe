# CLAUDE.md — ShipSafe Project Rules

## Project Overview

ShipSafe is a self-hosted AI code verification gateway. It sits in CI/CD pipelines between code generation and merge, running multi-layered verification on AI-generated code to produce trust scores and verification reports.

**Target users:** Engineering teams (especially EU-based) who need self-hosted, data-sovereign code quality assurance for AI-generated code.

**Core value prop:** The only production-grade, self-hosted AI code verification tool that keeps all analysis on-prem.

---

## Tech Stack

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| CLI & Core | Go 1.23+ | Single binary, fast, native K8s ecosystem |
| AI Analysis | Python 3.12+ | LLM libraries, AST parsing ecosystem |
| API Framework | Go net/http + chi router | Lightweight, stdlib-aligned |
| Frontend | React + TypeScript + Tailwind | Dashboard (Phase 4+) |
| Database | PostgreSQL 16+ (CloudNativePG) | Persistent analysis history |
| Queue | NATS JetStream | Async analysis jobs |
| Packaging | Helm 3 + Docker multi-stage | K8s-native deployment |
| CI | Forgejo Actions | Self-hosted CI |
| Git Hosting | Forgejo | Self-hosted git |
| Deployment | ArgoCD GitOps | Declarative K8s deployment |

---

## Project Structure

```
shipsafe/
├── CLAUDE.md                    # THIS FILE — project rules
├── README.md                    # User-facing docs
├── go.mod
├── go.sum
├── main.go                      # Entrypoint, minimal — delegates to cmd/
│
├── cmd/                         # CLI commands (Cobra)
│   ├── root.go                  # Root command, global flags
│   ├── scan.go                  # `shipsafe scan` command
│   ├── ci.go                    # `shipsafe ci` command (CI mode)
│   ├── server.go                # `shipsafe server` command (server mode)
│   └── version.go               # `shipsafe version`
│
├── pkg/
│   ├── interfaces/              # Shared interfaces and types (THE CONTRACT LAYER)
│   │   ├── types.go             # Core domain types (Finding, TrustScore, Report, etc.)
│   │   ├── analyzer.go          # Analyzer interface
│   │   ├── scorer.go            # Scorer interface
│   │   ├── reporter.go          # Reporter interface
│   │   ├── ai.go                # AI reviewer interface
│   │   └── vcs.go               # Version control system interface
│   │
│   ├── cli/                     # CLI-specific logic (output formatting, progress)
│   │   ├── output.go            # Terminal output helpers
│   │   ├── progress.go          # Progress indicators
│   │   └── config.go            # Config file loading (.shipsafe.yml)
│   │
│   ├── analyzer/                # Static analysis engine
│   │   ├── engine.go            # Analysis orchestrator
│   │   ├── complexity.go        # Cyclomatic complexity analysis
│   │   ├── coverage.go          # Test coverage delta detection
│   │   ├── secrets.go           # Hardcoded secrets detection
│   │   ├── imports.go           # Import/dependency change analysis
│   │   ├── patterns.go          # Anti-pattern detection
│   │   └── registry.go          # Analyzer plugin registry
│   │
│   ├── ai/                      # AI-powered analysis layer
│   │   ├── reviewer.go          # LLM-based code review
│   │   ├── provider.go          # LLM provider abstraction
│   │   ├── providers/           # Concrete provider implementations
│   │   │   ├── openai.go        # OpenAI-compatible (also covers Ollama, vLLM, OpenRouter)
│   │   │   └── anthropic.go     # Anthropic Claude API
│   │   ├── prompts/             # Prompt templates
│   │   │   ├── semantic.go      # Semantic diff analysis prompts
│   │   │   ├── logic.go         # Logic error detection prompts
│   │   │   └── convention.go    # Convention compliance prompts
│   │   └── context.go           # Codebase context builder for LLM
│   │
│   ├── security/                # Security scanning layer
│   │   ├── scanner.go           # Security scan orchestrator
│   │   ├── deps.go              # Dependency vulnerability checking
│   │   ├── sbom.go              # SBOM generation (CycloneDX)
│   │   ├── injection.go         # SQL/command injection detection
│   │   └── compliance/          # Compliance frameworks
│   │       ├── nis2.go          # NIS2 mapping
│   │       └── gdpr.go          # GDPR data handling checks
│   │
│   ├── scorer/                  # Trust score calculation
│   │   ├── calculator.go        # Weighted scoring model
│   │   ├── weights.go           # Default and configurable weights
│   │   └── thresholds.go        # RED/YELLOW/GREEN thresholds
│   │
│   ├── report/                  # Report generation
│   │   ├── generator.go         # Report orchestrator
│   │   ├── markdown.go          # Markdown report format
│   │   ├── json.go              # JSON report format
│   │   ├── terminal.go          # Terminal pretty-print format
│   │   ├── html.go              # HTML report format
│   │   └── templates/           # Go templates for reports
│   │       ├── report.md.tmpl
│   │       ├── report.html.tmpl
│   │       └── ci-comment.md.tmpl
│   │
│   ├── vcs/                     # Version control integration
│   │   ├── git.go               # Git operations (diff parsing, log, blame)
│   │   ├── diff.go              # Unified diff parser
│   │   ├── forgejo.go           # Forgejo API client (PR comments, webhooks)
│   │   ├── github.go            # GitHub API client
│   │   └── gitlab.go            # GitLab API client
│   │
│   └── server/                  # Server mode (Phase 4+)
│       ├── server.go            # HTTP server
│       ├── routes.go            # API routes
│       ├── webhook.go           # Webhook handlers
│       └── middleware.go        # Auth, logging middleware
│
├── tests/                       # Integration tests
│   ├── fixtures/                # Test fixture repos and diffs
│   │   ├── good-pr/             # Clean, well-tested PR
│   │   ├── bad-pr/              # PR with issues
│   │   ├── vulnerable-pr/       # PR with security issues
│   │   ├── no-tests-pr/         # PR missing tests
│   │   └── diffs/               # Sample unified diff files
│   │       ├── clean.diff
│   │       ├── secrets-leak.diff
│   │       ├── complexity-spike.diff
│   │       └── missing-tests.diff
│   ├── integration_test.go
│   └── helpers.go               # Test utilities
│
├── deploy/                      # Deployment artifacts
│   ├── docker/
│   │   └── Dockerfile           # Multi-stage build
│   ├── helm/
│   │   └── shipsafe/            # Helm chart
│   │       ├── Chart.yaml
│   │       ├── values.yaml
│   │       ├── templates/
│   │       │   ├── deployment.yaml
│   │       │   ├── service.yaml
│   │       │   ├── configmap.yaml
│   │       │   ├── secret.yaml
│   │       │   ├── ingress.yaml
│   │       │   └── serviceaccount.yaml
│   │       └── README.md
│   ├── argocd/
│   │   └── application.yaml     # ArgoCD Application manifest
│   └── ci/
│       ├── forgejo-action.yml   # Forgejo Actions workflow
│       └── github-action.yml    # GitHub Actions workflow
│
├── docs/
│   ├── architecture/
│   │   ├── DESIGN.md            # System architecture document
│   │   └── INTERFACES.md        # Interface contract documentation
│   ├── TASKBOARD.md             # Active task tracking for subagents
│   └── user/
│       ├── QUICKSTART.md        # Getting started guide
│       └── CONFIGURATION.md     # Config reference
│
└── configs/
    └── shipsafe.example.yml     # Example configuration file
```

---

## Subagent Boundaries

**CRITICAL RULE: Each subagent ONLY modifies files within its owned directories. Cross-boundary changes require Architect approval.**

| Agent | Owned Paths | Read Access |
|-------|------------|-------------|
| **Architect** | `docs/architecture/`, `pkg/interfaces/`, `CLAUDE.md`, `docs/TASKBOARD.md` | Everything |
| **CLI Agent** | `cmd/`, `pkg/cli/`, `main.go` | `pkg/interfaces/` |
| **Analyzer Agent** | `pkg/analyzer/` | `pkg/interfaces/` |
| **AI Agent** | `pkg/ai/` | `pkg/interfaces/` |
| **Security Agent** | `pkg/security/` | `pkg/interfaces/` |
| **Scorer Agent** | `pkg/scorer/`, `pkg/report/` | `pkg/interfaces/` |
| **VCS Agent** | `pkg/vcs/` | `pkg/interfaces/` |
| **Infra Agent** | `deploy/`, `.forgejo/`, `configs/` | `cmd/`, `pkg/interfaces/` |
| **Test Agent** | `tests/`, `*_test.go` anywhere | Everything (read), test files (write) |
| **Dashboard Agent** | `web/` (Phase 4+) | `pkg/interfaces/`, `pkg/server/` |

### Boundary Enforcement

- If an agent needs a new method on an interface, it MUST document the request in `docs/TASKBOARD.md` and wait for the Architect to update `pkg/interfaces/`.
- If an agent discovers a bug in another agent's code, it documents it in `docs/TASKBOARD.md` — it does NOT fix it directly.
- The Architect is the only agent that modifies `pkg/interfaces/`.
- The Test Agent may create `*_test.go` files in any package directory.

---

## Code Conventions

### Go Conventions

```go
// Package comments: every package has a doc comment
// Package analyzer provides static analysis checks for code diffs.
package analyzer

// Errors: use sentinel errors with package prefix
var (
    ErrAnalysisFailed = errors.New("analyzer: analysis failed")
    ErrInvalidDiff    = errors.New("analyzer: invalid diff format")
)

// Error wrapping: always wrap with context
return fmt.Errorf("analyzing complexity for %s: %w", filename, err)

// Constructors: New* pattern, return interface when possible
func NewComplexityAnalyzer(opts ...Option) interfaces.Analyzer {
    // ...
}

// Options: functional options pattern for configuration
type Option func(*complexityAnalyzer)

func WithThreshold(t int) Option {
    return func(a *complexityAnalyzer) {
        a.threshold = t
    }
}

// Context: all public methods that do I/O accept context.Context as first param
func (a *complexityAnalyzer) Analyze(ctx context.Context, diff *interfaces.Diff) (*interfaces.AnalysisResult, error) {
    // ...
}
```

### Naming

- Interfaces: verb-noun (`Analyzer`, `Scorer`, `Reporter`) — live in `pkg/interfaces/`
- Implementations: descriptive private structs (`complexityAnalyzer`, `weightedScorer`)
- Files: lowercase, single-word or hyphenated (`complexity.go`, `trust-score.go`)
- Test files: `*_test.go` in the same package
- Constants: `CamelCase` for exported, `camelCase` for unexported
- No abbreviations except universally known ones (HTTP, URL, ID, API, PR, CI, VCS)

### Error Handling

- Never ignore errors. If you intentionally discard, comment why: `_ = file.Close() // best-effort cleanup`
- Return errors up the chain with wrapping. Let the CLI layer decide how to present them.
- Use structured logging (slog) for operational messages, NOT fmt.Println
- Analyzer checks should return findings, not errors, for expected code issues. Errors are for infrastructure failures.

### Logging

```go
// Use slog throughout
import "log/slog"

slog.Info("analysis complete", "file", filename, "findings", len(findings), "duration", elapsed)
slog.Error("failed to parse diff", "error", err, "path", path)
```

### Testing

- Table-driven tests for all analyzers
- Use `testdata/` or `tests/fixtures/` for test inputs
- Test against interfaces, not concrete types
- Minimum: every analyzer has at least one positive and one negative test case
- Name tests: `TestAnalyzerName_Scenario_ExpectedBehavior`

```go
func TestComplexityAnalyzer_HighComplexityFunction_ReturnsWarning(t *testing.T) {
    // ...
}
```

### Dependencies

- Minimize external dependencies. Stdlib first.
- Approved dependencies:
  - `github.com/spf13/cobra` — CLI framework
  - `github.com/go-chi/chi/v5` — HTTP router (server mode)
  - `github.com/stretchr/testify` — Test assertions
  - `github.com/smacker/go-tree-sitter` — AST parsing (if needed)
  - `golang.org/x/tools` — Go analysis tools
- Any new dependency must be justified in the PR/commit message

---

## Configuration

ShipSafe uses a `.shipsafe.yml` config file:

```yaml
# .shipsafe.yml
version: "1"

# Trust score thresholds
thresholds:
  green: 80    # Score >= 80: safe to merge
  yellow: 50   # Score 50-79: review recommended
  # Score < 50: RED — block merge

# Analysis modules to enable
analyzers:
  complexity:
    enabled: true
    threshold: 15          # Max cyclomatic complexity per function
  coverage:
    enabled: true
    min_delta: 0           # Minimum test coverage change (0 = no decrease allowed)
  secrets:
    enabled: true
  imports:
    enabled: true
  patterns:
    enabled: true

# AI-powered review (requires LLM provider config)
ai:
  enabled: false
  provider: "openai-compatible"   # openai-compatible | anthropic
  endpoint: "http://ollama:11434/v1"  # Self-hosted Ollama example
  model: "codellama:13b"
  # api_key sourced from SHIPSAFE_AI_API_KEY env var

# Security scanning
security:
  enabled: true
  sbom: true
  compliance:
    nis2: false
    gdpr: false

# CI integration
ci:
  fail_on: "red"              # red | yellow — when to return non-zero exit
  comment: true               # Post results as PR comment
  comment_format: "markdown"  # markdown | summary

# Output
output:
  format: "terminal"          # terminal | json | markdown | html
  verbose: false
```

---

## Interface Contract Rules

1. All cross-module communication goes through interfaces defined in `pkg/interfaces/`
2. No package imports another package's internal types directly
3. The `interfaces` package has ZERO dependencies on any other `pkg/` package
4. Interfaces are kept minimal — prefer many small interfaces over few large ones
5. New interface methods require Architect approval and must be backward-compatible

---

## Git Conventions

- Branch naming: `feature/phase-N/description` (e.g., `feature/phase-1/complexity-analyzer`)
- Commit messages: `[agent] verb: description` (e.g., `[analyzer] add: cyclomatic complexity check`)
- One logical change per commit
- All commits must pass `go vet`, `go build`, and existing tests

---

## Build & Run Commands

```bash
# Build
go build -o shipsafe .

# Run scan on a diff file
./shipsafe scan --diff ./path/to/file.diff

# Run scan on a directory (compares against git HEAD)
./shipsafe scan ./path/to/repo

# Run in CI mode (auto-detects environment, posts comments)
./shipsafe ci

# Run as server (Phase 4+)
./shipsafe server --port 8080

# Run tests
go test ./...

# Run specific package tests
go test ./pkg/analyzer/...
```

---

## Task Tracking

All work is tracked in `docs/TASKBOARD.md`. Format:

```markdown
## Phase N: Title

### TODO
- [ ] AGENT: Task description

### IN PROGRESS
- [ ] AGENT: Task description (started YYYY-MM-DD)

### DONE
- [x] AGENT: Task description (completed YYYY-MM-DD)
```

Agents check `TASKBOARD.md` at the start of each session to understand current priorities.

---

## Quality Gates (Dogfooding)

Once Phase 3 is complete, ShipSafe must pass its own analysis on every PR. This is non-negotiable. If ShipSafe can't verify itself, it can't verify anyone else's code.
