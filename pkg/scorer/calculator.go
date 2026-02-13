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
// Score is clamped to [0, 100].
func (c *Calculator) Score(results []*interfaces.AnalysisResult) *interfaces.TrustScore {
	breakdown := make(map[interfaces.Category]int)
	findingCount := make(map[interfaces.Severity]int)

	var totalPenalty float64

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
			totalPenalty += penalty

			breakdown[f.Category] += int(math.Round(penalty))
			findingCount[f.Severity]++
		}
	}

	score := 100 - int(math.Round(totalPenalty))
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	rating := RatingFromScore(score, c.greenThreshold, c.yellowThreshold)

	return &interfaces.TrustScore{
		Score:        score,
		Rating:       rating,
		Breakdown:    breakdown,
		FindingCount: findingCount,
	}
}
