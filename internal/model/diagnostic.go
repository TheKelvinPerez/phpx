package model

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

type Diagnostic struct {
	Code      string            `json:"code"`
	Severity  Severity          `json:"severity"`
	Message   string            `json:"message"`
	Detail    string            `json:"detail,omitempty"`
	Hint      string            `json:"hint,omitempty"`
	Sources   []SourceReference `json:"sources,omitempty"`
	Provider  string            `json:"provider,omitempty"`
	Retryable bool              `json:"retryable"`
}
