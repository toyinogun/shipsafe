// Package analyzer provides static analysis checks for code diffs.
package analyzer

import (
	"context"
	"fmt"
	"sync"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

// Analyzer is the interface that individual analysis checks implement.
// Defined here at the consumer site until the Architect adds it to pkg/interfaces/.
type Analyzer interface {
	// Name returns the unique identifier for this analyzer.
	Name() string

	// Analyze runs the analysis against a parsed diff and returns results.
	Analyze(ctx context.Context, diff *interfaces.Diff) (*interfaces.AnalysisResult, error)
}

// Registry manages a collection of analyzers and tracks which are enabled.
type Registry struct {
	mu        sync.RWMutex
	analyzers map[string]Analyzer
	enabled   map[string]bool
}

// NewRegistry creates an empty analyzer registry.
func NewRegistry() *Registry {
	return &Registry{
		analyzers: make(map[string]Analyzer),
		enabled:   make(map[string]bool),
	}
}

// Register adds an analyzer to the registry. It is enabled by default.
// Returns an error if an analyzer with the same name is already registered.
func (r *Registry) Register(a Analyzer) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := a.Name()
	if _, exists := r.analyzers[name]; exists {
		return fmt.Errorf("analyzer: %q is already registered", name)
	}

	r.analyzers[name] = a
	r.enabled[name] = true
	return nil
}

// Get returns an analyzer by name. Returns nil if not found.
func (r *Registry) Get(name string) Analyzer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.analyzers[name]
}

// List returns the names of all registered analyzers.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.analyzers))
	for name := range r.analyzers {
		names = append(names, name)
	}
	return names
}

// SetEnabled enables or disables an analyzer by name.
// Returns an error if the analyzer is not registered.
func (r *Registry) SetEnabled(name string, enabled bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.analyzers[name]; !exists {
		return fmt.Errorf("analyzer: %q is not registered", name)
	}
	r.enabled[name] = enabled
	return nil
}

// IsEnabled reports whether the named analyzer is enabled.
func (r *Registry) IsEnabled(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.enabled[name]
}

// EnabledAnalyzers returns all analyzers that are currently enabled.
func (r *Registry) EnabledAnalyzers() []Analyzer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Analyzer
	for name, a := range r.analyzers {
		if r.enabled[name] {
			result = append(result, a)
		}
	}
	return result
}
