package analyzer

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

// Engine orchestrates running all enabled analyzers against a diff.
type Engine struct {
	registry *Registry
}

// NewEngine creates an analysis engine backed by the given registry.
func NewEngine(registry *Registry) *Engine {
	return &Engine{registry: registry}
}

// Run executes all enabled analyzers against the diff in parallel.
// A failing analyzer does not stop other analyzers from running.
// Returns results for every enabled analyzer, including those that errored.
// Respects context cancellation.
func (e *Engine) Run(ctx context.Context, diff *interfaces.Diff) ([]*interfaces.AnalysisResult, error) {
	if diff == nil {
		return nil, fmt.Errorf("analyzer: diff must not be nil")
	}

	analyzers := e.registry.EnabledAnalyzers()
	if len(analyzers) == 0 {
		slog.Info("no enabled analyzers to run")
		return nil, nil
	}

	slog.Info("starting analysis", "analyzer_count", len(analyzers))

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results = make([]*interfaces.AnalysisResult, 0, len(analyzers))
	)

	for _, a := range analyzers {
		wg.Add(1)
		go func(a Analyzer) {
			defer wg.Done()

			// Check for context cancellation before starting.
			if ctx.Err() != nil {
				return
			}

			name := a.Name()
			start := time.Now()
			slog.Info("running analyzer", "name", name)

			result, err := a.Analyze(ctx, diff)
			elapsed := time.Since(start)

			if err != nil {
				slog.Error("analyzer failed", "name", name, "error", err, "duration", elapsed)
				result = &interfaces.AnalysisResult{
					AnalyzerName: name,
					Duration:     elapsed,
					Error:        fmt.Errorf("analyzer %s: %w", name, err),
				}
			} else {
				result.Duration = elapsed
				slog.Info("analyzer complete", "name", name, "findings", len(result.Findings), "duration", elapsed)
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(a)
	}

	// Wait for all analyzers or context cancellation.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All analyzers finished.
	case <-ctx.Done():
		slog.Warn("analysis cancelled", "error", ctx.Err())
		// Wait for in-flight goroutines to notice cancellation and finish.
		<-done
		return results, ctx.Err()
	}

	return results, nil
}
