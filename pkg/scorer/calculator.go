package scorer

import (
	"math"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

// Calculator computes trust scores from analysis findings.
type Calculator struct {
	severityWeights     SeverityWeights
	categoryMultipliers CategoryMultipliers
	greenThreshold      int
	yellowThreshold     int
}

// Option configures the Calculator.
type Option func(*Calculator)

// WithSeverityWeights overrides the default severity weights.
func WithSeverityWeights(w SeverityWeights) Option {
	return func(c *Calculator) {
		c.severityWeights = w
	}
}

// WithCategoryMultipliers overrides the default category multipliers.
func WithCategoryMultipliers(m CategoryMultipliers) Option {
	return func(c *Calculator) {
		c.categoryMultipliers = m
	}
}

// WithThresholds overrides the default GREEN/YELLOW thresholds.
func WithThresholds(green, yellow int) Option {
	return func(c *Calculator) {
		c.greenThreshold = green
		c.yellowThreshold = yellow
	}
}

// NewCalculator creates a scorer with optional configuration.
func NewCalculator(opts ...Option) *Calculator {
	c := &Calculator{
		severityWeights:     DefaultSeverityWeights(),
		categoryMultipliers: DefaultCategoryMultipliers(),
		greenThreshold:      DefaultGreenThreshold,
		yellowThreshold:     DefaultYellowThreshold,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Score computes a TrustScore from analysis results.
// Formula: start at 100, subtract penalty per finding.
// penalty = severity_weight * category_multiplier * confidence
// Per-category penalties are capped to prevent one noisy category from dominating.
// Severity floors ensure the score reflects actual risk level.
// Score is clamped to [0, 100].
func (c *Calculator) Score(results []*interfaces.AnalysisResult) *interfaces.TrustScore {
	type categoryInfo struct {
		penalty     float64
		hasCritical bool
		hasHigh     bool
	}

	categories := make(map[interfaces.Category]*categoryInfo)
	findingCount := make(map[interfaces.Severity]int)

	var hasCritical, hasHigh bool

	for _, result := range results {
		if result == nil || result.Error != nil {
			continue
		}
		for _, f := range result.Findings {
			weight := c.severityWeights.SeverityWeight(f.Severity)
			multiplier := c.categoryMultipliers.Multiplier(f.Category)
			confidence := f.Confidence
			if confidence <= 0 {
				confidence = 1.0
			}

			penalty := float64(weight) * multiplier * confidence

			ci, ok := categories[f.Category]
			if !ok {
				ci = &categoryInfo{}
				categories[f.Category] = ci
			}
			ci.penalty += penalty

			if f.Severity == interfaces.SeverityCritical {
				ci.hasCritical = true
				hasCritical = true
			}
			if f.Severity == interfaces.SeverityHigh {
				ci.hasHigh = true
				hasHigh = true
			}

			findingCount[f.Severity]++
		}
	}

	// Apply per-category caps and compute total penalty.
	var totalPenalty float64
	breakdown := make(map[interfaces.Category]int)

	for cat, ci := range categories {
		cap := float64(CategoryPenaltyCap)
		if (cat == interfaces.CategorySecurity || cat == interfaces.CategorySecrets) &&
			(ci.hasCritical || ci.hasHigh) {
			cap = float64(CriticalCategoryPenaltyCap)
		}

		capped := ci.penalty
		if capped > cap {
			capped = cap
		}

		totalPenalty += capped
		breakdown[cat] = int(math.Round(capped))
	}

	score := 100 - int(math.Round(totalPenalty))
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	// Apply severity floors.
	if !hasCritical && !hasHigh && score < MinScoreNoCriticalNoHigh {
		score = MinScoreNoCriticalNoHigh
	} else if !hasCritical && score < MinScoreNoCritical {
		score = MinScoreNoCritical
	}

	rating := RatingFromScore(score, c.greenThreshold, c.yellowThreshold)

	return &interfaces.TrustScore{
		Score:        score,
		Rating:       rating,
		Breakdown:    breakdown,
		FindingCount: findingCount,
	}
}
