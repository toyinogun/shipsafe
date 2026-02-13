package report

import (
	"encoding/json"
	"io"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

// JSONFormatter writes a report as JSON.
type JSONFormatter struct{}

// NewJSONFormatter creates a JSON report formatter.
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

// Format writes the report as indented JSON to the given writer.
func (f *JSONFormatter) Format(w io.Writer, report *interfaces.Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
