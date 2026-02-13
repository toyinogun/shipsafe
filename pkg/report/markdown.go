package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

// MarkdownFormatter writes a report as Markdown suitable for PR comments.
type MarkdownFormatter struct{}

// NewMarkdownFormatter creates a Markdown report formatter.
func NewMarkdownFormatter() *MarkdownFormatter {
	return &MarkdownFormatter{}
}

// Format writes the report as Markdown to the given writer.
func (f *MarkdownFormatter) Format(w io.Writer, report *interfaces.Report) error {
	f.writeHeader(w, report)
	f.writeSummaryTable(w, report)
	f.writeFindings(w, report)
	f.writeFooter(w, report)
	return nil
}

func (f *MarkdownFormatter) writeHeader(w io.Writer, report *interfaces.Report) {
	badge := scoreBadge(report.TrustScore)
	fmt.Fprintf(w, "# ShipSafe Verification Report %s\n\n", badge)
}

func (f *MarkdownFormatter) writeSummaryTable(w io.Writer, report *interfaces.Report) {
	score := report.TrustScore
	meta := report.DiffMeta

	fmt.Fprintln(w, "| Metric | Value |")
	fmt.Fprintln(w, "|--------|-------|")
	fmt.Fprintf(w, "| **Trust Score** | %d/100 %s |\n", score.Score, scoreBadge(score))
	fmt.Fprintf(w, "| **Rating** | %s |\n", score.Rating)
	fmt.Fprintf(w, "| **Total Findings** | %d |\n", len(report.Findings))
	fmt.Fprintf(w, "| **Files Changed** | %d |\n", meta.FilesChanged)
	fmt.Fprintf(w, "| **Lines** | +%d / -%d |\n", meta.Additions, meta.Deletions)

	if len(score.FindingCount) > 0 {
		parts := formatFindingCounts(score.FindingCount)
		fmt.Fprintf(w, "| **Breakdown** | %s |\n", parts)
	}

	fmt.Fprintln(w)
}

func (f *MarkdownFormatter) writeFindings(w io.Writer, report *interfaces.Report) {
	if len(report.Findings) == 0 {
		fmt.Fprintln(w, "> No findings — clean diff!")
		fmt.Fprintln(w)
		return
	}

	// Group by category.
	grouped := groupByCategory(report.Findings)

	for _, cat := range []interfaces.Category{
		interfaces.CategorySecrets,
		interfaces.CategorySecurity,
		interfaces.CategoryLogic,
		interfaces.CategoryComplexity,
		interfaces.CategoryCoverage,
		interfaces.CategoryPattern,
		interfaces.CategoryImport,
		interfaces.CategoryConvention,
	} {
		findings, ok := grouped[cat]
		if !ok {
			continue
		}

		fmt.Fprintf(w, "## %s (%d)\n\n", categoryTitle(cat), len(findings))

		for _, finding := range findings {
			location := finding.File
			if finding.StartLine > 0 {
				location = fmt.Sprintf("%s:%d", finding.File, finding.StartLine)
			}

			fmt.Fprintf(w, "<details>\n")
			fmt.Fprintf(w, "<summary><strong>%s</strong> [%s] — <code>%s</code></summary>\n\n",
				finding.Title, strings.ToUpper(string(finding.Severity)), location)

			if finding.Description != "" {
				fmt.Fprintf(w, "%s\n\n", finding.Description)
			}
			if finding.Suggestion != "" {
				fmt.Fprintf(w, "**Suggestion:** %s\n\n", finding.Suggestion)
			}
			fmt.Fprintf(w, "*Source: %s | Confidence: %.0f%%*\n\n", finding.Source, finding.Confidence*100)
			fmt.Fprintln(w, "</details>")
			fmt.Fprintln(w)
		}
	}
}

func (f *MarkdownFormatter) writeFooter(w io.Writer, report *interfaces.Report) {
	fmt.Fprintln(w, "---")
	fmt.Fprintf(w, "*Report ID: %s | Generated: %s*\n",
		report.ID, report.Timestamp.Format("2006-01-02 15:04:05"))
}

// scoreBadge returns a text badge based on the rating.
func scoreBadge(score interfaces.TrustScore) string {
	switch score.Rating {
	case interfaces.RatingGreen:
		return "🟢"
	case interfaces.RatingYellow:
		return "🟡"
	case interfaces.RatingRed:
		return "🔴"
	default:
		return "⚪"
	}
}

// groupByCategory groups findings by their category.
func groupByCategory(findings []interfaces.Finding) map[interfaces.Category][]interfaces.Finding {
	grouped := make(map[interfaces.Category][]interfaces.Finding)
	for _, f := range findings {
		grouped[f.Category] = append(grouped[f.Category], f)
	}
	return grouped
}

// categoryTitle returns a human-readable title for a category.
func categoryTitle(c interfaces.Category) string {
	switch c {
	case interfaces.CategorySecurity:
		return "Security"
	case interfaces.CategorySecrets:
		return "Secrets"
	case interfaces.CategoryLogic:
		return "Logic"
	case interfaces.CategoryComplexity:
		return "Complexity"
	case interfaces.CategoryCoverage:
		return "Coverage"
	case interfaces.CategoryPattern:
		return "Patterns"
	case interfaces.CategoryImport:
		return "Imports"
	case interfaces.CategoryConvention:
		return "Conventions"
	default:
		return string(c)
	}
}
