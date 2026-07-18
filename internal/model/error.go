package model

import "errors"

type ErrorCode string

const (
	ErrorUsage                   ErrorCode = "ELEFANTE_USAGE"
	ErrorDiscovery               ErrorCode = "ELEFANTE_DISCOVERY"
	ErrorDiscoveryAmbiguousRoots ErrorCode = "ELEFANTE_DISCOVERY_AMBIGUOUS_ROOTS"
	ErrorRequirements            ErrorCode = "ELEFANTE_REQUIREMENTS"
	ErrorProvider                ErrorCode = "ELEFANTE_PROVIDER"
	ErrorApprovalRequired        ErrorCode = "ELEFANTE_APPROVAL_REQUIRED"
	ErrorPlanMismatch            ErrorCode = "ELEFANTE_PLAN_MISMATCH"
	ErrorNetwork                 ErrorCode = "ELEFANTE_NETWORK"
	ErrorTrust                   ErrorCode = "ELEFANTE_TRUST"
	ErrorSync                    ErrorCode = "ELEFANTE_SYNC"
	ErrorArtifact                ErrorCode = "ELEFANTE_ARTIFACT"
	ErrorState                   ErrorCode = "ELEFANTE_STATE"
	ErrorInternal                ErrorCode = "ELEFANTE_INTERNAL"
)

type ErrorCategory string

const (
	CategoryUsage            ErrorCategory = "usage"
	CategoryDiscovery        ErrorCategory = "discovery"
	CategoryRequirements     ErrorCategory = "requirements"
	CategoryProvider         ErrorCategory = "provider"
	CategoryApprovalRequired ErrorCategory = "approval_required"
	CategoryPlanMismatch     ErrorCategory = "plan_mismatch"
	CategoryNetwork          ErrorCategory = "network"
	CategoryTrust            ErrorCategory = "trust"
	CategorySync             ErrorCategory = "sync"
	CategoryArtifact         ErrorCategory = "artifact"
	CategoryState            ErrorCategory = "state"
	CategoryInternal         ErrorCategory = "internal"
)

type SourceReference struct {
	Path  string `json:"path"`
	Kind  string `json:"kind,omitempty"`
	Field string `json:"field,omitempty"`
	Line  int    `json:"line,omitempty"`
}

type ErrorDetail struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Error struct {
	Code      ErrorCode         `json:"code"`
	Category  ErrorCategory     `json:"category"`
	Message   string            `json:"message"`
	Detail    string            `json:"detail,omitempty"`
	Hint      string            `json:"hint,omitempty"`
	Sources   []SourceReference `json:"sources,omitempty"`
	Provider  string            `json:"provider,omitempty"`
	Retryable bool              `json:"retryable"`
	Details   []ErrorDetail     `json:"details,omitempty"`
	cause     error
}

func NewError(code ErrorCode, message string) *Error {
	category, _ := errorDefinition(code)

	return &Error{
		Code:     code,
		Category: category,
		Message:  message,
	}
}

func WrapError(code ErrorCode, message string, cause error) *Error {
	commandError := NewError(code, message)
	commandError.cause = cause

	return commandError
}

func (commandError *Error) Error() string {
	return commandError.Message
}

func (commandError *Error) Unwrap() error {
	return commandError.cause
}

func (commandError *Error) WithHint(hint string) *Error {
	commandError.Hint = hint

	return commandError
}

func (commandError *Error) WithRetryable(retryable bool) *Error {
	commandError.Retryable = retryable

	return commandError
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}

	var commandError *Error
	if !errors.As(err, &commandError) {
		return 70
	}

	_, exitCode := errorDefinition(commandError.Code)

	return exitCode
}

func ExitForError(err error) Exit {
	return Exit{
		Origin: ExitOriginElefante,
		Code:   ExitCode(err),
	}
}

func errorDefinition(code ErrorCode) (ErrorCategory, int) {
	switch code {
	case ErrorUsage:
		return CategoryUsage, 2
	case ErrorDiscovery, ErrorDiscoveryAmbiguousRoots:
		return CategoryDiscovery, 3
	case ErrorRequirements:
		return CategoryRequirements, 4
	case ErrorProvider:
		return CategoryProvider, 5
	case ErrorApprovalRequired:
		return CategoryApprovalRequired, 6
	case ErrorPlanMismatch:
		return CategoryPlanMismatch, 7
	case ErrorNetwork:
		return CategoryNetwork, 8
	case ErrorTrust:
		return CategoryTrust, 9
	case ErrorSync:
		return CategorySync, 10
	case ErrorArtifact:
		return CategoryArtifact, 11
	case ErrorState:
		return CategoryState, 12
	case ErrorInternal:
		return CategoryInternal, 70
	default:
		return CategoryInternal, 70
	}
}
