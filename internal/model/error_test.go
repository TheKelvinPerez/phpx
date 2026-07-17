package model_test

import (
	"errors"
	"testing"

	"github.com/elefantephp/elefante/internal/model"
)

func TestErrorCodesMapToStableCategoriesAndExitCodes(t *testing.T) {
	tests := []struct {
		code     model.ErrorCode
		category model.ErrorCategory
		exitCode int
	}{
		{model.ErrorUsage, model.CategoryUsage, 2},
		{model.ErrorDiscovery, model.CategoryDiscovery, 3},
		{model.ErrorRequirements, model.CategoryRequirements, 4},
		{model.ErrorProvider, model.CategoryProvider, 5},
		{model.ErrorApprovalRequired, model.CategoryApprovalRequired, 6},
		{model.ErrorPlanMismatch, model.CategoryPlanMismatch, 7},
		{model.ErrorNetwork, model.CategoryNetwork, 8},
		{model.ErrorTrust, model.CategoryTrust, 9},
		{model.ErrorSync, model.CategorySync, 10},
		{model.ErrorArtifact, model.CategoryArtifact, 11},
		{model.ErrorState, model.CategoryState, 12},
		{model.ErrorInternal, model.CategoryInternal, 70},
	}

	for _, test := range tests {
		t.Run(string(test.code), func(t *testing.T) {
			commandError := model.NewError(test.code, "test error")

			if commandError.Category != test.category {
				t.Fatalf("expected category %q, got %q", test.category, commandError.Category)
			}
			if got := model.ExitCode(commandError); got != test.exitCode {
				t.Fatalf("expected exit %d, got %d", test.exitCode, got)
			}
		})
	}
}

func TestWrappedErrorRetainsCauseAndTypedExit(t *testing.T) {
	cause := errors.New("network unavailable")
	commandError := model.WrapError(model.ErrorNetwork, "Could not reach the registry.", cause)

	if !errors.Is(commandError, cause) {
		t.Fatal("expected wrapped cause to remain available")
	}
	if got := model.ExitCode(commandError); got != 8 {
		t.Fatalf("expected network exit 8, got %d", got)
	}
	if got := model.ExitCode(errors.New("untyped")); got != 70 {
		t.Fatalf("expected untyped errors to use internal exit 70, got %d", got)
	}
}
