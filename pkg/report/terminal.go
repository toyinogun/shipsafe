package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

// ANSI color codes for terminal output.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

// TerminalFormatter writes a color-coded report to a terminal.
type TerminalFormatter struct{}

// NewTerminalFormatter creates a terminal report formatter.
func NewTerminalFormatter() *TerminalFormatter {
	return &TerminalFormatter{}
}

// Format writes the report to the given writer using ANSI colors.
func (f *TerminalFormatter) Format(w io.Writer, report *interfaces.Report) error {
	f.writeHeader(w, report)
	f.writeSummary(w, report)
	f.writeFindings(w, report)
	f.writeFooter(w, report)
	return nil
}

func (f *TerminalFormatter) writeHeader(w io.Writer, report *interfaces.Report) {
	fmt.Fprintf(w, "\n%s%s══════════════════════════════════════════%s\n", colorBold, colorCyan, colorReset)
	fmt.Fprintf(w, "%s%s  ShipSafe Verification Report%s\n", colorBold, colorCyan, colorReset)
	fmt.Fprintf(w, "%s%s══════════════════════════════════════════%s\n\n", colorBold, colorCyan, colorReset)
}

func (f *TerminalFormatter) writeSummary(w io.Writer, report *interfaces.Report) {
	score := report.TrustScore
	color := ratingColor(score.Rating)

	fmt.Fprintf(w, "  %s%sTrust Score: %d/100 [%s]%s\n\n",
		colorBold, color, score.Score, score.Rating, colorReset)

	total := len(report.Findings)
	if total == 0 {
		fmt.Fprintf(w, "  %sNo findings — clean diff!%s\n\n", colorGreen, colorReset)
		return
	}

	parts := formatFindingCounts(score.FindingCount)
	fmt.Fprintf(w, "  %d findings (%s)\n\n", total, parts)
}

func (f *TerminalFormatter) writeFindings(w io.Writer, report *interfaces.Report) {
	if len(report.Findings) == 0 {
		return
	}

	// Group findings by severity.
	grouped := groupBySeverity(report.Findings)

	for _, sev := range []interfaces.Severity{
		interfaces.SeverityCritical,
		interfaces.SeverityHigh,
		interfaces.SeverityMedium,
		interfaces.SeverityLow,
		interfaces.SeverityInfo,
	} {
		findings, ok := grouped[sev]
		if !ok {
			continue
		}

		color := severityColor(sev)
		label := strings.ToUpper(string(sev))
		fmt.Fprintf(w, "  %s%s── %s (%d) ──%s\n", colorBold, color, label, len(findings), colorReset)

		for _, finding := range findings {
			location := finding.File
			if finding.StartLine > 0 {
				location = fmt.Sprintf("%s:%d", finding.File, finding.StartLine)
			}
			fmt.Fprintf(w, "    %s[%s]%s %s\n", color, finding.ID, colorReset, finding.Title)
			fmt.Fprintf(w, "      %s%s%s\n", colorDim, location, colorReset)
			if finding.Description != "" {
				fmt.Fprintf(w, "      %s\n", finding.Description)
			}
			if finding.Suggestion != "" {
				fmt.Fprintf(w, "      %s→ %s%s\n", colorCyan, finding.Suggestion, colorReset)
			}
			fmt.Fprintln(w)
		}
	}
}

func (f *TerminalFormatter) writeFooter(w io.Writer, report *interfaces.Report) {
	meta := report.DiffMeta
	fmt.Fprintf(w, "  %s%s──────────────────────────────────────────%s\n", colorDim, colorCyan, colorReset)
	fmt.Fprintf(w, "  %sFiles: %d | +%d/-%d | Report: %s%s\n",
		colorDim, meta.FilesChanged, meta.Additions, meta.Deletions, report.ID, colorReset)
	fmt.Fprintf(w, "  %sGenerated: %s%s\n\n",
		colorDim, report.Timestamp.Format("2006-01-02 15:04:05"), colorReset)
}

// ratingColor returns the ANSI color for a rating.
func ratingColor(r interfaces.Rating) string {
	switch r {
	case interfaces.RatingGreen:
		return colorGreen
	case interfaces.RatingYellow:
		return colorYellow
	case interfaces.RatingRed:
		return colorRed
	default:
		return colorReset
	}
}

// severityColor returns the ANSI color for a severity level.
func severityColor(s interfaces.Severity) string {
	switch s {
	case interfaces.SeverityCritical, interfaces.SeverityHigh:
		return colorRed
	case interfaces.SeverityMedium:
		return colorYellow
	case interfaces.SeverityLow, interfaces.SeverityInfo:
		return colorDim
	default:
		return colorReset
	}
}

// groupBySeverity groups findings by their severity.
func groupBySeverity(findings []interfaces.Finding) map[interfaces.Severity][]interfaces.Finding {
	grouped := make(map[interfaces.Severity][]interfaces.Finding)
	for _, f := range findings {
		grouped[f.Severity] = append(grouped[f.Severity], f)
	}
	return grouped
}

// formatFindingCounts produces a summary like "1 high, 4 medium, 7 low".
func formatFindingCounts(counts map[interfaces.Severity]int) string {
	var parts []string
	for _, sev := range []interfaces.Severity{
		interfaces.SeverityCritical,
		interfaces.SeverityHigh,
		interfaces.SeverityMedium,
		interfaces.SeverityLow,
		interfaces.SeverityInfo,
	} {
		if c := counts[sev]; c > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", c, sev))
		}
	}
	return strings.Join(parts, ", ")
}
