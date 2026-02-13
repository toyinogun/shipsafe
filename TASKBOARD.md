# ShipSafe Task Board

> Agents: Check this file at the start of each session. Pick up TODO items for your role. Move to IN PROGRESS when starting, DONE when complete.

---

## Phase 0: Foundation

### DONE
- [x] Architect: Create CLAUDE.md
- [x] Architect: Create DESIGN.md
- [x] Architect: Create TASKBOARD.md
- [x] Architect: Create pkg/interfaces/types.go — all core types and interfaces
- [x] Architect: Create pkg/interfaces/vcs.go — VCS-specific types
- [x] CLI Agent: Initialize `go.mod`, set up `main.go` with cobra root command
- [x] CLI Agent: Implement `cmd/root.go` with global flags (--config, --verbose, --format)
- [x] CLI Agent: Implement `cmd/version.go`
- [x] Analyzer Agent: Create `pkg/analyzer/registry.go` — analyzer plugin registry
- [x] Analyzer Agent: Create `pkg/analyzer/engine.go` — analysis orchestrator
- [x] Scorer Agent: Create `pkg/scorer/calculator.go` — weighted scoring implementation
- [x] Scorer Agent: Create `pkg/scorer/weights.go` — default severity/category weights
- [x] Scorer Agent: Create `pkg/scorer/thresholds.go` — GREEN/YELLOW/RED threshold logic
- [x] VCS Agent: Create `pkg/vcs/diff.go` — unified diff parser with tests
- [x] Test Agent: Create `tests/fixtures/diffs/clean.diff`
- [x] Test Agent: Create `tests/fixtures/diffs/secrets-leak.diff`
- [x] Test Agent: Create `tests/fixtures/diffs/complexity-spike.diff`
- [x] Test Agent: Create `tests/fixtures/diffs/missing-tests.diff`

---

## Phase 1: MVP CLI + Basic Analysis

### DONE
- [x] CLI Agent: Implement `cmd/scan.go` — accepts --diff, --format, --config flags; wires up pipeline
- [x] Analyzer Agent: Implement `pkg/analyzer/complexity.go` — cyclomatic complexity checker
- [x] Analyzer Agent: Implement `pkg/analyzer/secrets.go` — secret/key detection via regex + entropy
- [x] Analyzer Agent: Implement `pkg/analyzer/coverage.go` — test file heuristic checker
- [x] Analyzer Agent: Implement `pkg/analyzer/imports.go` — dependency file change detector
- [x] Analyzer Agent: Implement `pkg/analyzer/patterns.go` — anti-pattern detection
- [x] Scorer Agent: Create `pkg/report/generator.go` — report orchestrator
- [x] Scorer Agent: Create `pkg/report/markdown.go` — markdown report format
- [x] Scorer Agent: Create `pkg/report/json.go` — JSON report format
- [x] Scorer Agent: Create `pkg/report/terminal.go` — terminal pretty-print with color
- [x] CLI Agent: Create `pkg/cli/config.go` — .shipsafe.yml config file loader
- [x] CLI Agent: Wire all modules in scan command, working end-to-end
- [x] Test Agent: Write integration tests (clean, secrets-leak, complexity-spike, mixed-issues)
- [x] Test Agent: Write unit tests for each analyzer
- [x] Infra Agent: Create `shipsafe.example.yml`
- [x] Analyzer Agent: Fix false positives in secrets, complexity, patterns, and imports analyzers

---

## Phase 2: AI-Powered Review

### DONE
- [x] AI Agent: Implement `pkg/ai/provider.go` — LLM provider abstraction
- [x] AI Agent: Implement `pkg/ai/providers/openai.go` — OpenAI-compatible provider (covers Ollama, vLLM)
- [x] AI Agent: Implement `pkg/ai/reviewer.go` — orchestrates semantic/logic/convention review
- [x] AI Agent: Create `pkg/ai/prompts/semantic.go` — semantic diff analysis prompts
- [x] AI Agent: Create `pkg/ai/prompts/logic.go` — logic error detection prompts
- [x] AI Agent: Create `pkg/ai/prompts/convention.go` — convention compliance prompts
- [x] AI Agent: Implement `pkg/ai/context.go` — codebase context builder
- [x] Test Agent: Write tests for AI reviewer with mock LLM responses
- [x] CLI Agent: Integrate AI reviewer into scan pipeline
- [x] Milestone: v0.3.0 — full pipeline with AI review validated

---

## Phase 3: CI/CD Integration

### TODO
- [ ] VCS Agent: Implement `pkg/vcs/gitlab.go` — GitLab API (MR comments, status checks)
- [ ] VCS Agent: Implement `pkg/vcs/git.go` — local git operations (diff from HEAD, branch comparison)
- [ ] Test Agent: Integration test — full CI flow with mock VCS
- [ ] Infra Agent: Dogfood — add ShipSafe to its own CI pipeline

### DONE
- [x] CLI Agent: Implement `cmd/ci.go` — auto-detect CI env, post comments, exit codes
- [x] VCS Agent: Implement `pkg/vcs/forgejo.go` — Forgejo API (PR comments, status checks) with tests
- [x] VCS Agent: Implement `pkg/vcs/github.go` — GitHub API (PR comments, status checks) with tests
- [x] Infra Agent: Create `.forgejo/workflows/ci.yml`
- [x] Infra Agent: Create `deploy/ci/github-action.yml`
- [x] Infra Agent: Create `deploy/docker/Dockerfile`
- [x] Scorer Agent: Fix per-category caps and severity floors for balanced scoring
- [x] Analyzer Agent: Fix skip URLs in entropy detector
- [x] Analyzer Agent: Fix skip text content entropy, smart coverage for frontend

---

## Phase 4: Server + Dashboard (Not Started)

### TODO
- [ ] CLI Agent: Implement `cmd/server.go` — HTTP server command
- [ ] Server Agent: Implement `pkg/server/server.go` — HTTP server
- [ ] Server Agent: Implement `pkg/server/routes.go` — API routes
- [ ] Server Agent: Implement `pkg/server/webhook.go` — Webhook handlers
- [ ] Server Agent: Implement `pkg/server/middleware.go` — Auth, logging middleware
- [ ] Dashboard Agent: Set up `web/` — React + TypeScript + Tailwind dashboard
- [ ] Infra Agent: Create Helm chart (`deploy/helm/shipsafe/`)
- [ ] Infra Agent: Create ArgoCD application manifest (`deploy/argocd/`)

---

## Phase 5: Compliance (Not Started)

### TODO
- [ ] Security Agent: Implement `pkg/security/scanner.go` — Security scan orchestrator
- [ ] Security Agent: Implement `pkg/security/deps.go` — Dependency vulnerability checking
- [ ] Security Agent: Implement `pkg/security/sbom.go` — SBOM generation (CycloneDX)
- [ ] Security Agent: Implement `pkg/security/injection.go` — SQL/command injection detection
- [ ] Security Agent: Implement `pkg/security/compliance/nis2.go` — NIS2 mapping
- [ ] Security Agent: Implement `pkg/security/compliance/gdpr.go` — GDPR data handling checks
- [ ] AI Agent: Implement `pkg/ai/providers/anthropic.go` — Anthropic Claude API provider
- [ ] Scorer Agent: Implement `pkg/report/html.go` — HTML report format

---

## Interface Change Requests

> Agents: If you need a new method or type in `pkg/interfaces/`, document it here. Architect will review and implement.

| Requesting Agent | Interface/Type | Proposed Change | Status |
|-----------------|---------------|----------------|--------|
| (none yet) | | | |

---

## Bugs & Issues

> Agents: If you find a bug in another agent's code, log it here. Do NOT fix it directly.

| Reporter | Affected Package | Description | Status |
|----------|-----------------|-------------|--------|
| (none yet) | | | |
