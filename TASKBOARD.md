# ShipSafe Task Board

> Agents: Check this file at the start of each session. Pick up TODO items for your role. Move to IN PROGRESS when starting, DONE when complete.

---

## Phase 0: Foundation

### TODO
- [ ] CLI Agent: Initialize `go.mod` with module `github.com/toyinlola/shipsafe`, set up `main.go` with cobra root command
- [ ] CLI Agent: Implement `cmd/root.go` with global flags (--config, --verbose, --format)
- [ ] CLI Agent: Implement `cmd/version.go`
- [ ] Analyzer Agent: Create `pkg/analyzer/registry.go` — analyzer plugin registry (register/list/get analyzers)
- [ ] Analyzer Agent: Create `pkg/analyzer/engine.go` — analysis orchestrator that runs registered analyzers in parallel
- [ ] Scorer Agent: Create `pkg/scorer/calculator.go` — initial weighted scoring implementation
- [ ] Scorer Agent: Create `pkg/scorer/weights.go` — default severity/category weights from DESIGN.md
- [ ] Scorer Agent: Create `pkg/scorer/thresholds.go` — GREEN/YELLOW/RED threshold logic
- [ ] VCS Agent: Create `pkg/vcs/diff.go` — unified diff parser (parse raw diff text into `interfaces.Diff`)
- [ ] Test Agent: Create `tests/fixtures/diffs/clean.diff` — a sample clean diff with tests
- [ ] Test Agent: Create `tests/fixtures/diffs/secrets-leak.diff` — diff containing hardcoded API key
- [ ] Test Agent: Create `tests/fixtures/diffs/complexity-spike.diff` — diff with high-complexity function
- [ ] Test Agent: Create `tests/fixtures/diffs/missing-tests.diff` — diff with new code but no tests

### IN PROGRESS

### DONE
- [x] Architect: Create CLAUDE.md
- [x] Architect: Create docs/architecture/DESIGN.md
- [x] Architect: Create docs/TASKBOARD.md
- [x] Architect: Create pkg/interfaces/types.go
- [x] Architect: Create pkg/interfaces/analyzer.go
- [x] Architect: Create pkg/interfaces/scorer.go
- [x] Architect: Create pkg/interfaces/reporter.go
- [x] Architect: Create pkg/interfaces/ai.go
- [x] Architect: Create pkg/interfaces/vcs.go

---

## Phase 1: MVP CLI + Basic Analysis

### TODO
- [ ] CLI Agent: Implement `cmd/scan.go` — accepts --diff, --format, --config flags; wires up pipeline
- [ ] Analyzer Agent: Implement `pkg/analyzer/complexity.go` — cyclomatic complexity checker
- [ ] Analyzer Agent: Implement `pkg/analyzer/secrets.go` — secret/key detection via regex + entropy
- [ ] Analyzer Agent: Implement `pkg/analyzer/coverage.go` — test file heuristic checker
- [ ] Analyzer Agent: Implement `pkg/analyzer/imports.go` — dependency file change detector
- [ ] Analyzer Agent: Implement `pkg/analyzer/patterns.go` — anti-pattern detection (SQL concat, empty catch, etc.)
- [ ] Scorer Agent: Create `pkg/report/generator.go` — report orchestrator
- [ ] Scorer Agent: Create `pkg/report/markdown.go` — markdown report format
- [ ] Scorer Agent: Create `pkg/report/json.go` — JSON report format
- [ ] Scorer Agent: Create `pkg/report/terminal.go` — terminal pretty-print with color
- [ ] CLI Agent: Create `pkg/cli/config.go` — .shipsafe.yml config file loader
- [ ] CLI Agent: Create `pkg/cli/output.go` — terminal output helpers
- [ ] Test Agent: Write integration test — scan clean.diff → GREEN score
- [ ] Test Agent: Write integration test — scan secrets-leak.diff → RED score
- [ ] Test Agent: Write integration test — scan complexity-spike.diff → YELLOW score
- [ ] Test Agent: Write unit tests for each analyzer
- [ ] Infra Agent: Create `configs/shipsafe.example.yml`

### IN PROGRESS

### DONE

---

## Phase 2: AI-Powered Review

### TODO
- [ ] AI Agent: Implement `pkg/ai/provider.go` — LLM provider abstraction
- [ ] AI Agent: Implement `pkg/ai/providers/openai.go` — OpenAI-compatible provider (covers Ollama, vLLM)
- [ ] AI Agent: Implement `pkg/ai/reviewer.go` — orchestrates semantic/logic/convention review
- [ ] AI Agent: Create `pkg/ai/prompts/semantic.go` — semantic diff analysis prompts
- [ ] AI Agent: Create `pkg/ai/prompts/logic.go` — logic error detection prompts
- [ ] AI Agent: Create `pkg/ai/prompts/convention.go` — convention compliance prompts
- [ ] AI Agent: Implement `pkg/ai/context.go` — codebase context builder
- [ ] Test Agent: Write tests for AI reviewer with mock LLM responses
- [ ] Architect: Update interfaces if AI agent needs changes

### IN PROGRESS

### DONE

---

## Phase 3: CI/CD Integration

### TODO
- [ ] CLI Agent: Implement `cmd/ci.go` — auto-detect CI env, post comments, exit codes
- [ ] VCS Agent: Implement `pkg/vcs/forgejo.go` — Forgejo API (PR comments, status checks)
- [ ] VCS Agent: Implement `pkg/vcs/github.go` — GitHub API (PR comments, status checks)
- [ ] VCS Agent: Implement `pkg/vcs/gitlab.go` — GitLab API (MR comments, status checks)
- [ ] VCS Agent: Implement `pkg/vcs/git.go` — local git operations (diff from HEAD, branch comparison)
- [ ] Infra Agent: Create `deploy/ci/forgejo-action.yml`
- [ ] Infra Agent: Create `deploy/ci/github-action.yml`
- [ ] Infra Agent: Create `deploy/docker/Dockerfile`
- [ ] Test Agent: Integration test — full CI flow with mock VCS
- [ ] Infra Agent: Dogfood — add ShipSafe to its own CI pipeline

### IN PROGRESS

### DONE

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
