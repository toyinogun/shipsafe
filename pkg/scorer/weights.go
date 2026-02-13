// Package scorer calculates trust scores from analysis findings.
package scorer

import "github.com/toyinlola/shipsafe/pkg/interfaces"

// Default severity weights define the base penalty points for each severity level.
const (
	DefaultWeightCritical = 25
	DefaultWeightHigh     = 15
	DefaultWeightMedium   = 8
	DefaultWeightLow      = 3
	DefaultWeightInfo     = 0
)

// Default category multipliers amplify or reduce penalties based on finding category.
const (
	DefaultMultiplierSecurity   = 1.5
	DefaultMultiplierSecrets    = 2.0
	DefaultMultiplierLogic      = 1.3
	DefaultMultiplierComplexity = 0.8
	DefaultMultiplierCoverage   = 0.4
	DefaultMultiplierPattern    = 0.5
	DefaultMultiplierImport     = 0.3
	DefaultMultiplierConvention = 0.3
)

// Per-category penalty caps prevent one noisy category from dominating the score.
const (
	CategoryPenaltyCap         = 25 // Max penalty any single category can contribute
	CriticalCategoryPenaltyCap = 40 // Higher cap for security/secrets with critical/high findings
)

// Severity floors ensure the score reflects actual risk level.
const (
	MinScoreNoCriticalNoHigh = 30 // Floor when no critical and no high findings
	MinScoreNoCritical       = 15 // Floor when no critical findings (but has high)
)

// SeverityWeights maps severity levels to their base penalty points.
type SeverityWeights map[interfaces.Severity]int

// CategoryMultipliers maps categories to their penalty multipliers.
type CategoryMultipliers map[interfaces.Category]float64

// DefaultSeverityWeights returns the default severity weight map.
func DefaultSeverityWeights() SeverityWeights {
	return SeverityWeights{
		interfaces.SeverityCritical: DefaultWeightCritical,
		interfaces.SeverityHigh:     DefaultWeightHigh,
		interfaces.SeverityMedium:   DefaultWeightMedium,
		interfaces.SeverityLow:      DefaultWeightLow,
		interfaces.SeverityInfo:     DefaultWeightInfo,
	}
}

// DefaultCategoryMultipliers returns the default category multiplier map.
func DefaultCategoryMultipliers() CategoryMultipliers {
	return CategoryMultipliers{
		interfaces.CategorySecurity:   DefaultMultiplierSecurity,
		interfaces.CategorySecrets:    DefaultMultiplierSecrets,
		interfaces.CategoryLogic:      DefaultMultiplierLogic,
		interfaces.CategoryComplexity: DefaultMultiplierComplexity,
		interfaces.CategoryCoverage:   DefaultMultiplierCoverage,
		interfaces.CategoryPattern:    DefaultMultiplierPattern,
		interfaces.CategoryImport:     DefaultMultiplierImport,
		interfaces.CategoryConvention: DefaultMultiplierConvention,
	}
}

// SeverityWeight returns the penalty weight for a severity, falling back to 0 for unknown levels.
func (w SeverityWeights) SeverityWeight(s interfaces.Severity) int {
	if v, ok := w[s]; ok {
		return v
	}
	return 0
}

// Multiplier returns the multiplier for a category, falling back to 1.0 for unknown categories.
func (m CategoryMultipliers) Multiplier(c interfaces.Category) float64 {
	if v, ok := m[c]; ok {
		return v
	}
	return 1.0
}
