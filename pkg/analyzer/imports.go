package analyzer

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

// Dependency manifest filenames to watch for.
var dependencyManifests = map[string]string{
	"go.mod":            "Go",
	"go.sum":            "Go",
	"package.json":      "JavaScript/TypeScript",
	"package-lock.json": "JavaScript/TypeScript",
	"yarn.lock":         "JavaScript/TypeScript",
	"pnpm-lock.yaml":    "JavaScript/TypeScript",
	"requirements.txt":  "Python",
	"Pipfile":           "Python",
	"Pipfile.lock":      "Python",
	"pyproject.toml":    "Python",
	"poetry.lock":       "Python",
	"Cargo.toml":        "Rust",
	"Cargo.lock":        "Rust",
	"pom.xml":           "Java",
	"build.gradle":      "Java",
	"build.gradle.kts":  "Kotlin",
	"Gemfile":           "Ruby",
	"Gemfile.lock":      "Ruby",
	"composer.json":     "PHP",
	"composer.lock":     "PHP",
}

// Patterns to detect new dependency additions in various manifest files.
var newDepPatterns = []struct {
	files   []string
	pattern *regexp.Regexp
	desc    string
}{
	{
		files:   []string{"go.mod"},
		pattern: regexp.MustCompile(`^\s*require\s+\S+|^\s+\S+\s+v[\d.]+`),
		desc:    "Go module dependency",
	},
	{
		files:   []string{"package.json"},
		pattern: regexp.MustCompile(`^\s*"[^"]+"\s*:\s*"[\^~>=<]*\d`),
		desc:    "npm package",
	},
	{
		files:   []string{"requirements.txt"},
		pattern: regexp.MustCompile(`^\s*[a-zA-Z][a-zA-Z0-9._-]*\s*[><=!~]+`),
		desc:    "Python package",
	},
	{
		files:   []string{"Cargo.toml"},
		pattern: regexp.MustCompile(`^\s*[a-zA-Z][a-zA-Z0-9_-]*\s*=\s*(?:"[\d.]+"|{)`),
		desc:    "Rust crate",
	},
	{
		files:   []string{"Gemfile"},
		pattern: regexp.MustCompile(`^\s*gem\s+['"]`),
		desc:    "Ruby gem",
	},
	{
		files:   []string{"pom.xml"},
		pattern: regexp.MustCompile(`<dependency>`),
		desc:    "Maven dependency",
	},
}

// Pattern to detect semver major version in added lines.
var majorVersionPattern = regexp.MustCompile(`v?(\d+)\.\d+\.\d+`)

// ImportsAnalyzer detects changes to dependency manifest files.
type ImportsAnalyzer struct{}

// NewImportsAnalyzer creates a new dependency change analyzer.
func NewImportsAnalyzer() *ImportsAnalyzer {
	return &ImportsAnalyzer{}
}

// Name returns the analyzer identifier.
func (im *ImportsAnalyzer) Name() string {
	return "imports"
}

// Analyze scans the diff for dependency manifest changes.
func (im *ImportsAnalyzer) Analyze(ctx context.Context, diff *interfaces.Diff) (*interfaces.AnalysisResult, error) {
	result := &interfaces.AnalysisResult{
		AnalyzerName: im.Name(),
	}

	for i := range diff.Files {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		file := &diff.Files[i]
		if file.IsBinary {
			continue
		}

		filename := fileBaseName(file.Path)
		lang, isManifest := dependencyManifests[filename]
		if !isManifest {
			continue
		}

		findings := im.analyzeManifest(file, filename, lang)
		result.Findings = append(result.Findings, findings...)
	}

	return result, nil
}

// analyzeManifest inspects a dependency manifest file for changes.
func (im *ImportsAnalyzer) analyzeManifest(file *interfaces.FileDiff, filename, lang string) []interfaces.Finding {
	var findings []interfaces.Finding

	// Collect added and removed lines.
	var addedLines []interfaces.Line
	var removedLines []interfaces.Line
	for j := range file.Hunks {
		addedLines = append(addedLines, file.Hunks[j].AddedLines...)
		removedLines = append(removedLines, file.Hunks[j].RemovedLines...)
	}

	// Check for new dependencies (added lines matching dep patterns).
	for _, line := range addedLines {
		if isDepLine(filename, line.Content) {
			depName := extractDepName(line.Content)
			// Check if this is a major version bump (same dep in removed lines with different major).
			if majorBump, oldMajor, newMajor := isMajorVersionBump(line.Content, removedLines, depName); majorBump {
				findings = append(findings, interfaces.Finding{
					ID:        fmt.Sprintf("IMP-MAJOR-%d", line.Number),
					Category:  interfaces.CategoryImport,
					Severity:  interfaces.SeverityMedium,
					File:      file.Path,
					StartLine: line.Number,
					EndLine:   line.Number,
					Title:     fmt.Sprintf("Major version bump in %s dependency", lang),
					Description: fmt.Sprintf(
						"Dependency %q was upgraded from major version %s to %s. Major version bumps may include breaking changes.",
						depName, oldMajor, newMajor,
					),
					Suggestion: "Review the dependency changelog for breaking changes and update consuming code accordingly.",
					Source:     "imports",
					Confidence: 0.80,
					Metadata: map[string]any{
						"language":    lang,
						"dependency":  depName,
						"old_major":   oldMajor,
						"new_major":   newMajor,
						"manifest":    filename,
					},
				})
			} else {
				findings = append(findings, interfaces.Finding{
					ID:        fmt.Sprintf("IMP-NEW-%d", line.Number),
					Category:  interfaces.CategoryImport,
					Severity:  interfaces.SeverityLow,
					File:      file.Path,
					StartLine: line.Number,
					EndLine:   line.Number,
					Title:     fmt.Sprintf("New %s dependency added", lang),
					Description: fmt.Sprintf(
						"A new dependency was added to %s: %s. New dependencies increase the supply chain attack surface.",
						filename, strings.TrimSpace(line.Content),
					),
					Suggestion: "Verify the dependency is from a trusted source, is actively maintained, and has no known vulnerabilities.",
					Source:     "imports",
					Confidence: 0.70,
					Metadata: map[string]any{
						"language":   lang,
						"dependency": depName,
						"manifest":   filename,
					},
				})
			}
		}
	}

	// Check for removed dependencies.
	for _, line := range removedLines {
		if isDepLine(filename, line.Content) {
			depName := extractDepName(line.Content)
			// Only flag if the dep wasn't re-added (i.e., not a version change).
			if !depExistsInLines(depName, addedLines) {
				findings = append(findings, interfaces.Finding{
					ID:        fmt.Sprintf("IMP-REMOVED-%d", line.Number),
					Category:  interfaces.CategoryImport,
					Severity:  interfaces.SeverityInfo,
					File:      file.Path,
					StartLine: line.Number,
					EndLine:   line.Number,
					Title:     fmt.Sprintf("%s dependency removed", lang),
					Description: fmt.Sprintf(
						"Dependency %q was removed from %s.",
						depName, filename,
					),
					Suggestion: "Ensure no code still references the removed dependency.",
					Source:     "imports",
					Confidence: 0.75,
				})
			}
		}
	}

	return findings
}

// isDepLine checks if a line looks like a dependency declaration for the given manifest.
func isDepLine(filename, content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
		return false
	}

	for _, p := range newDepPatterns {
		for _, f := range p.files {
			if f == filename && p.pattern.MatchString(content) {
				return true
			}
		}
	}

	// Generic fallback: if the manifest is known but no specific pattern, check for version-like strings.
	if _, ok := dependencyManifests[filename]; ok {
		return majorVersionPattern.MatchString(content)
	}

	return false
}

// extractDepName tries to extract the dependency name from a manifest line.
func extractDepName(content string) string {
	trimmed := strings.TrimSpace(content)

	// Go module: github.com/foo/bar v1.2.3
	goModRe := regexp.MustCompile(`^\s*(?:require\s+)?(\S+)\s+v`)
	if m := goModRe.FindStringSubmatch(trimmed); len(m) > 1 {
		return m[1]
	}

	// package.json: "name": "^1.2.3"
	npmRe := regexp.MustCompile(`^\s*"([^"]+)"\s*:`)
	if m := npmRe.FindStringSubmatch(trimmed); len(m) > 1 {
		return m[1]
	}

	// requirements.txt: name==1.2.3 or name>=1.2.3
	pipRe := regexp.MustCompile(`^\s*([a-zA-Z][a-zA-Z0-9._-]*)\s*[><=!~]`)
	if m := pipRe.FindStringSubmatch(trimmed); len(m) > 1 {
		return m[1]
	}

	// Gemfile: gem 'name'
	gemRe := regexp.MustCompile(`gem\s+['"]([^'"]+)['"]`)
	if m := gemRe.FindStringSubmatch(trimmed); len(m) > 1 {
		return m[1]
	}

	// Cargo.toml: name = "1.2.3" or name = {version = ...}
	cargoRe := regexp.MustCompile(`^\s*([a-zA-Z][a-zA-Z0-9_-]*)\s*=`)
	if m := cargoRe.FindStringSubmatch(trimmed); len(m) > 1 {
		return m[1]
	}

	// Fallback: return the first word-like token.
	fields := strings.Fields(trimmed)
	if len(fields) > 0 {
		return strings.Trim(fields[0], `"'<>/`)
	}

	return "unknown"
}

// isMajorVersionBump checks if the added line represents a major version bump
// compared to a removed line with the same dependency name.
func isMajorVersionBump(addedContent string, removedLines []interfaces.Line, depName string) (bool, string, string) {
	newMajor := extractMajorVersion(addedContent)
	if newMajor == "" {
		return false, "", ""
	}

	for _, removed := range removedLines {
		removedDep := extractDepName(removed.Content)
		if removedDep == depName {
			oldMajor := extractMajorVersion(removed.Content)
			if oldMajor != "" && oldMajor != newMajor {
				return true, oldMajor, newMajor
			}
		}
	}

	return false, "", ""
}

// extractMajorVersion extracts the major version number from a line containing a semver.
func extractMajorVersion(content string) string {
	m := majorVersionPattern.FindStringSubmatch(content)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

// depExistsInLines checks if a dependency name appears in any of the given lines.
func depExistsInLines(depName string, lines []interfaces.Line) bool {
	for _, line := range lines {
		if strings.Contains(line.Content, depName) {
			return true
		}
	}
	return false
}

// fileBaseName extracts the filename from a path (last component).
func fileBaseName(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}
