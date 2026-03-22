package model

import "fmt"

// Severity distinguishes fatal errors from advisories.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Diagnostic is one validation finding.
type Diagnostic struct {
	Severity Severity
	Path     string // e.g. "cases[2].operationId"
	Message  string
	Hint     string // optional corrective guidance
}

func (d Diagnostic) String() string {
	s := fmt.Sprintf("%s: %s: %s", d.Severity, d.Path, d.Message)
	if d.Hint != "" {
		s += "\n  hint: " + d.Hint
	}
	return s
}

// HasErrors returns true if any diagnostic has SeverityError.
func HasErrors(diags []Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}
