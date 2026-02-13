// Package report generates verification reports from analysis results and trust scores.
package report

import (
	"crypto/rand"
	"fmt"
	"sort"
	"time"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

// severityOrder defines the sort priority for findings (critical first).
var severityOrder = map[interfaces.Severity]int{
	interfaces.SeverityCritical: 0,
	interfaces.SeverityHigh:     1,
	interfaces.SeverityMedium:   2,
	interfaces.SeverityLow:      3,
	interfaces.SeverityInfo:     4,
}

// Generator builds reports from analysis results and trust scores.
type Generator struct{}

// NewGenerator creates a report generator.
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate produces a Report from analysis results, a trust score, and the original diff.
func (g *Generator) Generate(results []*interfaces.AnalysisResult, score *interfaces.TrustScore, diff *interfaces.Diff) *interfaces.Report {
	start := time.Now()

	findings := collectFindings(results)
	sortFindingsBySeverity(findings)

	meta := buildDiffMetadata(diff)
	summary := buildSummary(score, findings)

	return &interfaces.Report{
		ID:         generateID(),
		Timestamp:  time.Now(),
		TrustScore: *score,
		Findings:   findings,
		Summary:    summary,
		DiffMeta:   meta,
		Duration:   time.Since(start),
	}
}

// collectFindings merges findings from all analysis results.
func collectFindings(results []*interfaces.AnalysisResult) []interfaces.Finding {
	var all []interfaces.Finding
	for _, r := range results {
		if r == nil || r.Error != nil {
			continue
		}
		all = append(all, r.Findings...)
	}
	return all
}

// sortFindingsBySeverity sorts findings with critical first, info last.
func sortFindingsBySeverity(findings []interfaces.Finding) {
	sort.Slice(findings, func(i, j int) bool {
		oi := severityOrder[findings[i].Severity]
		oj := severityOrder[findings[j].Severity]
		if oi != oj {
			return oi < oj
		}
		return findings[i].File < findings[j].File
	})
}

// buildDiffMetadata calculates summary stats from the diff.
func buildDiffMetadata(diff *interfaces.Diff) interfaces.DiffMetadata {
	if diff == nil {
		return interfaces.DiffMetadata{}
	}

	meta := interfaces.DiffMetadata{
		FilesChanged: len(diff.Files),
		BaseSHA:      diff.BaseSHA,
		HeadSHA:      diff.HeadSHA,
	}

	for _, f := range diff.Files {
		for _, h := range f.Hunks {
			meta.Additions += len(h.AddedLines)
			meta.Deletions += len(h.RemovedLines)
		}
	}

	return meta
}

// buildSummary creates a one-line summary of the trust score and findings.
func buildSummary(score *interfaces.TrustScore, findings []interfaces.Finding) string {
	total := len(findings)
	if total == 0 {
		return fmt.Sprintf("Trust Score: %d/100 [%s] — no findings", score.Score, score.Rating)
	}

	parts := ""
	for _, sev := range []interfaces.Severity{
		interfaces.SeverityCritical,
		interfaces.SeverityHigh,
		interfaces.SeverityMedium,
		interfaces.SeverityLow,
		interfaces.SeverityInfo,
	} {
		count := score.FindingCount[sev]
		if count > 0 {
			if parts != "" {
				parts += ", "
			}
			parts += fmt.Sprintf("%d %s", count, sev)
		}
	}

	return fmt.Sprintf("Trust Score: %d/100 [%s] — %d findings (%s)", score.Score, score.Rating, total, parts)
}

// generateID creates a unique report identifier.
func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b) // best-effort; crypto/rand is reliable
	return fmt.Sprintf("rpt-%x", b)
}
