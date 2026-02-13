# ShipSafe — System Architecture

## 1. Vision

ShipSafe is a self-hosted AI code verification gateway that produces trust scores for code changes. It answers one question: **"Is this code safe to merge?"**

It runs as a CLI in CI/CD pipelines (primary mode) or as a persistent server receiving webhooks (server mode). All analysis happens on-premises. No code leaves the user's infrastructure.

---

## 2. Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        CLI / Server                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────┐  │
│  │  scan     │  │  ci      │  │  server  │  │  version   │  │
│  │  command  │  │  command  │  │  command │  │  command   │  │
│  └─────┬────┘  └─────┬────┘  └─────┬────┘  └────────────┘  │
│        │             │             │                         │
│        └──────┬──────┘             │                         │
│               ▼                    ▼                         │
│  ┌─────────────────────────────────────────────────┐        │
│  │              Pipeline Orchestrator               │        │
│  │   (Coordinates analysis flow, collects results)  │        │
│  └──────┬───────────┬──────────┬──────────┬────────┘        │
│         │           │          │          │                  │
│         ▼           ▼          ▼          ▼                  │
│  ┌───────────┐ ┌─────────┐ ┌────────┐ ┌──────────┐         │
│  │ Analyzers │ │   AI    │ │Security│ │  VCS     │         │
│  │ (Static)  │ │ Review  │ │ Scan   │ │ Provider │         │
│  └─────┬─────┘ └────┬────┘ └───┬────┘ └──────────┘         │
│        │            │          │                             │
│        └─────┬──────┘──────────┘                            │
│              ▼                                               │
│  ┌──────────────────┐    ┌────────────────────┐             │
│  │  Trust Scorer     │───▶│  Report Generator  │             │
│  │  (Score 0-100)    │    │  (MD/JSON/HTML)    │             │
│  └──────────────────┘    └────────────────────┘             │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. Core Data Flow

### 3.1 CLI Scan Mode (Primary)

```
Input (diff/path/PR)
    │
    ▼
┌──────────────┐
│  VCS Layer   │  Parse diff, extract changed files, hunks, metadata
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Pipeline    │  Fan out to analyzers in parallel
│  Orchestrator│
└──────┬───────┘
       │
       ├──▶ Analyzer: Complexity      ──▶ []Finding
       ├──▶ Analyzer: Coverage         ──▶ []Finding
       ├──▶ Analyzer: Secrets          ──▶ []Finding
       ├──▶ Analyzer: Imports          ──▶ []Finding
       ├──▶ Analyzer: Patterns         ──▶ []Finding
       ├──▶ AI Reviewer (if enabled)   ──▶ []Finding
       └──▶ Security Scanner           ──▶ []Finding
       │
       ▼
┌──────────────┐
│  Scorer      │  Aggregate findings → weighted trust score (0-100)
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Reporter    │  Generate report in requested format
└──────┬───────┘
       │
       ▼
Output (terminal / file / PR comment)
```

### 3.2 CI Mode

Same as scan mode but additionally:
1. Auto-detects CI environment (GitHub Actions, Forgejo Actions, GitLab CI)
2. Extracts PR/MR metadata from environment variables
3. Posts report as PR comment via VCS provider
4. Returns exit code based on threshold config (`fail_on: red|yellow`)

### 3.3 Server Mode (Phase 4+)

```
Webhook (PR opened/updated)
    │
    ▼
┌──────────────┐
│  Webhook     │  Validate signature, parse event
│  Handler     │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Job Queue   │  Enqueue analysis job (NATS JetStream)
│  (NATS)      │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Worker      │  Execute pipeline (same as CLI scan)
└──────┬───────┘
       │
       ├──▶ Store results in PostgreSQL
       └──▶ Post PR comment via VCS provider
```

---

## 4. Module Specifications

### 4.1 Interfaces Package (`pkg/interfaces/`)

This is the contract layer. Every module depends on these interfaces but NEVER on each other's concrete implementations.

#### Core Types

```go
// Severity levels for findings
type Severity string

const (
    SeverityCritical Severity = "critical"  // Must fix before merge
    SeverityHigh     Severity = "high"      // Strongly recommended fix
    SeverityMedium   Severity = "medium"    // Should fix
    SeverityLow      Severity = "low"       // Informational
    SeverityInfo     Severity = "info"      // FYI only
)

// Category classifies what type of issue was found
type Category string

const (
    CategoryComplexity  Category = "complexity"
    CategoryCoverage    Category = "coverage"
    CategorySecurity    Category = "security"
    CategorySecrets     Category = "secrets"
    CategoryPattern     Category = "pattern"
    CategoryImport      Category = "import"
    CategoryLogic       Category = "logic"
    CategoryConvention  Category = "convention"
)

// Finding represents a single issue found during analysis
type Finding struct {
    ID          string   `json:"id"`           // Unique finding ID
    Category    Category `json:"category"`      // What type of issue
    Severity    Severity `json:"severity"`      // How serious
    File        string   `json:"file"`          // File path
    StartLine   int      `json:"start_line"`    // Line number (0 if not applicable)
    EndLine     int      `json:"end_line"`      // End line (0 if single line)
    Title       string   `json:"title"`         // Short description
    Description string   `json:"description"`   // Detailed explanation
    Suggestion  string   `json:"suggestion"`    // How to fix (optional)
    Source      string   `json:"source"`        // Which analyzer produced this
    Confidence  float64  `json:"confidence"`    // 0.0-1.0 confidence in finding
    Metadata    map[string]any `json:"metadata"` // Analyzer-specific extra data
}

// TrustScore is the final calculated trust score
type TrustScore struct {
    Score       int               `json:"score"`       // 0-100
    Rating      Rating            `json:"rating"`      // GREEN/YELLOW/RED
    Breakdown   map[Category]int  `json:"breakdown"`   // Per-category scores
    FindingCount map[Severity]int `json:"finding_count"` // Findings by severity
}

type Rating string

const (
    RatingGreen  Rating = "GREEN"   // Safe to merge
    RatingYellow Rating = "YELLOW"  // Review recommended
    RatingRed    Rating = "RED"     // Do not merge
)

// Diff represents a parsed code diff
type Diff struct {
    BaseSHA     string      `json:"base_sha"`
    HeadSHA     string      `json:"head_sha"`
    Files       []FileDiff  `json:"files"`
    PRTitle     string      `json:"pr_title"`      // If available
    PRBody      string      `json:"pr_body"`       // If available
    Author      string      `json:"author"`        // If available
}

type FileDiff struct {
    Path        string      `json:"path"`
    OldPath     string      `json:"old_path"`      // For renames
    Status      FileStatus  `json:"status"`        // Added, Modified, Deleted, Renamed
    Hunks       []Hunk      `json:"hunks"`
    Language    string      `json:"language"`       // Detected language
    IsBinary    bool        `json:"is_binary"`
}

type FileStatus string

const (
    FileAdded    FileStatus = "added"
    FileModified FileStatus = "modified"
    FileDeleted  FileStatus = "deleted"
    FileRenamed  FileStatus = "renamed"
)

type Hunk struct {
    OldStart    int    `json:"old_start"`
    OldLines    int    `json:"old_lines"`
    NewStart    int    `json:"new_start"`
    NewLines    int    `json:"new_lines"`
    Content     string `json:"content"`       // Raw hunk content
    AddedLines  []Line `json:"added_lines"`
    RemovedLines []Line `json:"removed_lines"`
}

type Line struct {
    Number  int    `json:"number"`
    Content string `json:"content"`
}

// AnalysisResult is what each analyzer returns
type AnalysisResult struct {
    AnalyzerName string    `json:"analyzer_name"`
    Findings     []Finding `json:"findings"`
    Duration     time.Duration `json:"duration"`
    Error        error     `json:"-"`             // Non-nil if analyzer itself failed
    Metadata     map[string]any `json:"metadata"`  // Analyzer-specific summary data
}

// Report is the final output
type Report struct {
    ID          string          `json:"id"`
    Timestamp   time.Time       `json:"timestamp"`
    TrustScore  TrustScore      `json:"trust_score"`
    Findings    []Finding       `json:"findings"`
    Summary     string          `json:"summary"`     // Human-readable summary
    DiffMeta    DiffMetadata    `json:"diff_metadata"`
    Duration    time.Duration   `json:"duration"`
    Config      map[string]any  `json:"config"`      // Active config snapshot
}

type DiffMetadata struct {
    FilesChanged  int `json:"files_changed"`
    Additions     int `json:"additions"`
    Deletions     int `json:"deletions"`
    BaseSHA       string `json:"base_sha"`
    HeadSHA       string `json:"head_sha"`
}
```

#### Interfaces

```go
// Analyzer performs a specific type of code analysis on a diff
type Analyzer interface {
    // Name returns the unique identifier for this analyzer
    Name() string
    // Analyze examines the diff and returns findings
    Analyze(ctx context.Context, diff *Diff) (*AnalysisResult, error)
}

// Scorer calculates a trust score from analysis results
type Scorer interface {
    // Score computes the aggregate trust score
    Score(ctx context.Context, results []*AnalysisResult) (*TrustScore, error)
}

// Reporter generates formatted output from a report
type Reporter interface {
    // Format returns the format name (markdown, json, terminal, html)
    Format() string
    // Generate produces the formatted report
    Generate(ctx context.Context, report *Report) ([]byte, error)
}

// AIReviewer performs LLM-powered code review
type AIReviewer interface {
    // Review performs AI-powered analysis of the diff
    Review(ctx context.Context, diff *Diff, opts *AIReviewOptions) (*AnalysisResult, error)
    // Available returns true if the AI provider is configured and reachable
    Available(ctx context.Context) bool
}

type AIReviewOptions struct {
    ContextFiles  []string   // Additional files for context
    FocusAreas    []string   // Specific areas to review (security, logic, etc.)
    MaxTokens     int        // Max tokens for LLM response
}

// VCSProvider abstracts git platform operations (GitHub, Forgejo, GitLab)
type VCSProvider interface {
    // GetDiff retrieves the diff for a PR/MR
    GetDiff(ctx context.Context, prRef string) (*Diff, error)
    // PostComment posts the report as a PR comment
    PostComment(ctx context.Context, prRef string, body string) error
    // SetStatus sets the commit status check
    SetStatus(ctx context.Context, sha string, status StatusState, description string) error
}

type StatusState string

const (
    StatusPending StatusState = "pending"
    StatusSuccess StatusState = "success"
    StatusFailure StatusState = "failure"
    StatusError   StatusState = "error"
)

// DiffParser parses raw diff content into structured Diff objects
type DiffParser interface {
    // Parse converts raw unified diff content into a Diff struct
    Parse(ctx context.Context, raw []byte) (*Diff, error)
    // ParseFile reads a diff file from disk
    ParseFile(ctx context.Context, path string) (*Diff, error)
}

// Pipeline orchestrates the full analysis workflow
type Pipeline interface {
    // Run executes the full analysis pipeline and returns a report
    Run(ctx context.Context, diff *Diff) (*Report, error)
}
```

---

## 5. Analysis Modules Detail

### 5.1 Complexity Analyzer

**Purpose:** Detect functions with high cyclomatic complexity, especially newly added or significantly changed functions.

**Logic:**
1. Parse added/modified files using AST (tree-sitter or go/ast for Go)
2. Calculate cyclomatic complexity per function
3. Compare against threshold (default: 15)
4. Flag functions that exceed threshold OR increased by > 5 from base

**Finding example:**
```json
{
  "category": "complexity",
  "severity": "medium",
  "file": "pkg/auth/handler.go",
  "start_line": 42,
  "title": "High cyclomatic complexity (23) in function HandleLogin",
  "suggestion": "Consider breaking this function into smaller units"
}
```

### 5.2 Coverage Analyzer

**Purpose:** Detect when new code is added without corresponding tests.

**Logic:**
1. Identify new/modified source files in diff
2. Check if corresponding test files are also modified or added
3. Use heuristics: new `.go` file → expect `_test.go` file; new `.py` file → expect `test_*.py`
4. For modified files, check if test files in the same package were also modified
5. Calculate a "test coverage delta" score

**Note:** This is heuristic-based in Phase 1. Phase 4+ can integrate with actual coverage tools.

### 5.3 Secrets Analyzer

**Purpose:** Detect hardcoded secrets, API keys, tokens, passwords.

**Logic:**
1. Scan added lines only (not removed lines)
2. Regex patterns for: AWS keys, private keys, JWT tokens, connection strings, passwords in config
3. Entropy analysis for high-entropy strings (potential secrets)
4. Check against common false positive patterns (test fixtures, examples)

**Severity:** Always HIGH or CRITICAL.

### 5.4 Import/Dependency Analyzer

**Purpose:** Detect changes to project dependencies that could introduce risk.

**Logic:**
1. Detect changes to dependency files: `go.mod`, `package.json`, `requirements.txt`, `Cargo.toml`, etc.
2. Flag new dependencies (new attack surface)
3. Flag major version bumps (breaking changes)
4. Flag removal of dependencies (potential breakage)

### 5.5 Pattern Analyzer

**Purpose:** Detect common anti-patterns and code smells.

**Logic (initial set):**
1. SQL string concatenation (injection risk)
2. Empty catch/except blocks (swallowed errors)
3. TODO/FIXME/HACK in new code
4. Deeply nested conditionals (> 4 levels)
5. Magic numbers in business logic
6. Console.log / print statements (debug leftovers)

This is extensible — the registry pattern lets us add checks without modifying the orchestrator.

---

## 6. Trust Score Calculation

### 6.1 Scoring Model

Each analyzer category has a **base weight** and findings reduce the score:

```
Base Score: 100

For each finding:
  penalty = severity_weight[finding.severity] * category_weight[finding.category] * finding.confidence

Final Score = max(0, 100 - sum(penalties))
```

### 6.2 Default Weights

**Severity weights:**
| Severity | Weight |
|----------|--------|
| Critical | 25 |
| High | 15 |
| Medium | 8 |
| Low | 3 |
| Info | 0 |

**Category multipliers:**
| Category | Multiplier |
|----------|-----------|
| Security | 1.5 |
| Secrets | 2.0 |
| Logic | 1.3 |
| Complexity | 0.8 |
| Coverage | 0.7 |
| Pattern | 0.5 |
| Import | 0.6 |
| Convention | 0.3 |

### 6.3 Thresholds (Configurable)

| Rating | Score Range | CI Exit Code |
|--------|------------|-------------|
| GREEN | 80-100 | 0 |
| YELLOW | 50-79 | 0 (unless `fail_on: yellow`) |
| RED | 0-49 | 1 |

---

## 7. AI Review Module

### 7.1 Provider Abstraction

The AI module supports any OpenAI-compatible API endpoint. This covers:
- **Self-hosted:** Ollama, vLLM, LocalAI, llama.cpp server
- **Cloud (EU):** Mistral API, OpenRouter (EU endpoints)
- **Cloud (global):** OpenAI, Anthropic

The user configures the endpoint and model. ShipSafe never sends code to any external service unless explicitly configured.

### 7.2 Review Strategy

The AI reviewer performs three types of analysis:

**Semantic Diff Review:**
- Input: The diff + PR title/description
- Question: "Does this code change accomplish what the PR description says?"
- Output: Findings about mismatches between intent and implementation

**Logic Error Detection:**
- Input: Changed functions with surrounding context
- Question: "Are there logic errors, edge cases, or off-by-one errors?"
- Output: Findings about potential bugs

**Convention Compliance:**
- Input: Changed code + sample of existing codebase patterns
- Question: "Does this code follow the project's existing conventions?"
- Output: Findings about style/pattern deviations

### 7.3 Context Window Management

To keep LLM calls efficient and within context limits:
1. Send only the relevant diff hunks + surrounding context (configurable, default ±50 lines)
2. For large PRs, chunk by file and aggregate findings
3. Include a "codebase summary" built from: README, directory structure, recent commit messages
4. Max context: configurable, default 8K tokens for input

---

## 8. Deployment Architecture

### 8.1 CLI Mode (Phases 1-3)

```
Developer Machine / CI Runner
┌─────────────────────────┐
│  shipsafe binary (Go)   │
│  ┌───────────────────┐  │
│  │ Analyzers (Go)    │  │
│  │ Scorer (Go)       │  │
│  │ Reporter (Go)     │  │
│  └───────────────────┘  │
│  ┌───────────────────┐  │
│  │ AI Module (opt.)  │──┼──▶ LLM API (Ollama / cloud)
│  └───────────────────┘  │
└─────────────────────────┘
```

Single binary. Zero infrastructure required for basic analysis.
AI module is optional and requires LLM endpoint config.

### 8.2 Server Mode (Phase 4+)

```
Kubernetes Cluster
┌─────────────────────────────────────────┐
│  Namespace: shipsafe                     │
│  ┌──────────────┐  ┌─────────────────┐  │
│  │  ShipSafe    │  │  PostgreSQL     │  │
│  │  Server      │  │  (CloudNativePG)│  │
│  │  (Deployment)│  │  (Cluster)      │  │
│  └──────┬───────┘  └────────▲────────┘  │
│         │                    │           │
│         ├────────────────────┘           │
│         │                                │
│  ┌──────▼───────┐  ┌─────────────────┐  │
│  │  NATS        │  │  ShipSafe       │  │
│  │  JetStream   │  │  Dashboard      │  │
│  │  (StatefulSet)│  │  (Deployment)  │  │
│  └──────────────┘  └─────────────────┘  │
│                                          │
│  ┌──────────────────────────────────┐   │
│  │  Ingress (Traefik / nginx)       │   │
│  └──────────────────────────────────┘   │
└─────────────────────────────────────────┘
          │
          ▼
   Forgejo / GitHub (webhooks)
```

### 8.3 Helm Chart Values (Key Config)

```yaml
replicaCount: 1

image:
  repository: registry.example.com/shipsafe
  tag: latest

server:
  port: 8080
  metricsPort: 9090

ai:
  enabled: false
  provider: "openai-compatible"
  endpoint: ""
  model: ""
  apiKeySecret: "shipsafe-ai-key"

database:
  enabled: true   # Set false to use external PG
  # CloudNativePG cluster spec
  instances: 1
  storage:
    size: 5Gi

nats:
  enabled: true

ingress:
  enabled: false
  className: "traefik"
  host: "shipsafe.example.com"

vcs:
  provider: "forgejo"  # forgejo | github | gitlab
  webhookSecret: ""
  # API token sourced from secret
```

---

## 9. Configuration Precedence

Configuration is resolved in this order (later overrides earlier):

1. Built-in defaults (hardcoded in Go)
2. `.shipsafe.yml` in repository root
3. `~/.config/shipsafe/config.yml` (user-level)
4. Environment variables (`SHIPSAFE_*` prefix)
5. CLI flags

---

## 10. Security Considerations

### 10.1 Data Sovereignty

- **Default:** All analysis runs locally. No network calls unless AI module is enabled.
- **AI Module:** Code is sent to the configured LLM endpoint. When using self-hosted LLMs (Ollama), code never leaves the network.
- **Server Mode:** All data stored in user's PostgreSQL. No telemetry, no phone-home.
- **Audit Trail:** Every analysis is logged with timestamp, config used, and findings (server mode).

### 10.2 Secret Handling

- API keys for LLM providers: environment variables or Kubernetes secrets. Never in config files.
- VCS tokens: same — env vars or K8s secrets.
- Webhook secrets: validated on every incoming request (server mode).

### 10.3 Supply Chain

- SBOM generated for ShipSafe itself (CycloneDX)
- Container images signed (cosign)
- Reproducible builds from tagged releases

---

## 11. Metrics & Observability

Server mode exposes Prometheus metrics:

```
shipsafe_analyses_total{rating="GREEN|YELLOW|RED"}    # Counter
shipsafe_analysis_duration_seconds                      # Histogram
shipsafe_trust_score                                    # Histogram
shipsafe_findings_total{category,severity}             # Counter
shipsafe_ai_review_duration_seconds                    # Histogram
shipsafe_ai_review_errors_total                        # Counter
shipsafe_webhook_requests_total{provider,status}       # Counter
```

---

## 12. Roadmap Alignment

| Phase | Focus | Deliverable |
|-------|-------|-------------|
| 0 | Foundation | Skeleton, interfaces, CLAUDE.md |
| 1 | MVP CLI | `shipsafe scan` with 5 basic analyzers |
| 2 | AI Review | LLM-powered semantic/logic/convention review |
| 3 | CI Integration | `shipsafe ci` + Forgejo/GitHub integration |
| 4 | Server + Dashboard | Persistent server, webhook, dashboard |
| 5 | Compliance | NIS2/GDPR reporting, SBOM, audit trail |

---

## 13. Future Considerations (Not in Scope Yet)

- **MCP Server:** Expose ShipSafe as an MCP server so AI coding tools can query trust scores
- **IDE Extension:** VS Code extension showing trust score before commit
- **Multi-repo Analysis:** Cross-repository dependency impact analysis
- **Custom Analyzers:** Plugin system for user-defined checks (WASM or Go plugins)
- **Historical Trends:** Trust score trends per team/repo over time
- **LLM Fine-tuning:** Fine-tune a small model on the user's codebase for better convention detection
