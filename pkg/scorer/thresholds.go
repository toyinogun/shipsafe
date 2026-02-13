package scorer

import "github.com/toyinlola/shipsafe/pkg/interfaces"

// Default threshold values.
const (
	DefaultGreenThreshold  = 80
	DefaultYellowThreshold = 50
)

// RatingFromScore returns the trust rating for a given score based on thresholds.
// GREEN: score >= greenThreshold
// YELLOW: score >= yellowThreshold
// RED: score < yellowThreshold
func RatingFromScore(score int, greenThreshold int, yellowThreshold int) interfaces.Rating {
	switch {
	case score >= greenThreshold:
		return interfaces.RatingGreen
	case score >= yellowThreshold:
		return interfaces.RatingYellow
	default:
		return interfaces.RatingRed
	}
}
