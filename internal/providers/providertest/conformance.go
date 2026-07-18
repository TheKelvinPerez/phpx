package providertest

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/providers"
)

type Suite struct {
	Provider         providers.Provider
	InspectRequest   providers.InspectRequest
	ExecutionRequest providers.ExecutionRequest
}

func Run(t *testing.T, suite Suite) {
	t.Helper()

	if suite.Provider == nil {
		t.Fatal("provider conformance requires a provider")
	}
	if strings.TrimSpace(suite.Provider.Name()) == "" {
		t.Fatal("provider name must not be empty")
	}

	t.Run("stable inspection", func(t *testing.T) {
		first, err := suite.Provider.Inspect(t.Context(), suite.InspectRequest)
		if err != nil {
			t.Fatalf("inspect provider: %v", err)
		}
		second, err := suite.Provider.Inspect(t.Context(), suite.InspectRequest)
		if err != nil {
			t.Fatalf("inspect provider again: %v", err)
		}
		if !reflect.DeepEqual(first, second) {
			t.Fatalf(
				"equivalent inspections differ\nfirst:  %#v\nsecond: %#v",
				first,
				second,
			)
		}
		if first.Provider != suite.Provider.Name() {
			t.Fatalf(
				"observation provider %q does not match %q",
				first.Provider,
				suite.Provider.Name(),
			)
		}
		if !first.Available {
			t.Fatalf("conformance fixture provider is unavailable: %#v", first)
		}
		if !strings.HasPrefix(first.Fingerprint, "sha256:") {
			t.Fatalf("provider fingerprint is not content addressed: %q", first.Fingerprint)
		}
		if !sort.SliceIsSorted(
			first.Capabilities,
			func(left int, right int) bool {
				return first.Capabilities[left] < first.Capabilities[right]
			},
		) {
			t.Fatalf("provider capabilities are not sorted: %#v", first.Capabilities)
		}
		if hasCapability(
			first.Capabilities,
			model.CapabilityInspectRuntime,
		) {
			if len(first.Runtimes) == 0 {
				t.Fatal("runtime inspection capability reported no runtimes")
			}
			for _, runtime := range first.Runtimes {
				if strings.TrimSpace(runtime.Name) == "" ||
					strings.TrimSpace(runtime.Version) == "" ||
					strings.TrimSpace(runtime.Source.Path) == "" {
					t.Fatalf("runtime identity is incomplete: %#v", runtime)
				}
			}
		}
		if hasCapability(
			first.Capabilities,
			model.CapabilityInspectExtensions,
		) {
			if len(first.Extensions) == 0 {
				t.Fatal("extension inspection capability reported no extensions")
			}
			for _, extension := range first.Extensions {
				if strings.TrimSpace(extension.Name) == "" ||
					strings.TrimSpace(extension.Source.Path) == "" {
					t.Fatalf(
						"extension identity is incomplete: %#v",
						extension,
					)
				}
			}
		}
		if hasCapability(
			first.Capabilities,
			model.CapabilityInspectComposer,
		) {
			if len(first.Composer) == 0 {
				t.Fatal("Composer inspection capability reported no executable")
			}
			for _, composer := range first.Composer {
				if strings.TrimSpace(composer.Version) == "" ||
					strings.TrimSpace(composer.Identity) == "" ||
					strings.TrimSpace(composer.Reference.Path) == "" {
					t.Fatalf("Composer identity is incomplete: %#v", composer)
				}
			}
		}
	})

	t.Run("deterministic offline plan", func(t *testing.T) {
		observation, err := suite.Provider.Inspect(
			t.Context(),
			suite.InspectRequest,
		)
		if err != nil {
			t.Fatalf("inspect provider: %v", err)
		}
		request := providers.ProviderPlanRequest{
			Facts: model.ProjectFacts{
				Identity: suite.InspectRequest.Project,
			},
			Observation: observation,
			Policy: model.PlanPolicy{
				Offline: true,
			},
		}
		first, err := suite.Provider.Plan(t.Context(), request)
		if err != nil {
			t.Fatalf("plan provider: %v", err)
		}
		second, err := suite.Provider.Plan(t.Context(), request)
		if err != nil {
			t.Fatalf("plan provider again: %v", err)
		}
		if !reflect.DeepEqual(first, second) {
			t.Fatalf(
				"equivalent provider plans differ\nfirst:  %#v\nsecond: %#v",
				first,
				second,
			)
		}
		for _, action := range first.Actions {
			if action.Network != model.NetworkNone {
				t.Fatalf(
					"offline provider plan contains network action %#v",
					action,
				)
			}
		}
	})

	t.Run("argument safe execution", func(t *testing.T) {
		first, err := suite.Provider.ExecutionSpec(
			t.Context(),
			suite.ExecutionRequest,
		)
		if err != nil {
			t.Fatalf("build execution spec: %v", err)
		}
		second, err := suite.Provider.ExecutionSpec(
			t.Context(),
			suite.ExecutionRequest,
		)
		if err != nil {
			t.Fatalf("build execution spec again: %v", err)
		}
		if !reflect.DeepEqual(first, second) {
			t.Fatalf(
				"equivalent execution specs differ\nfirst:  %#v\nsecond: %#v",
				first,
				second,
			)
		}
		if !reflect.DeepEqual(first.Arguments, suite.ExecutionRequest.Arguments) {
			t.Fatalf(
				"execution arguments changed\nexpected: %#v\ngot:      %#v",
				suite.ExecutionRequest.Arguments,
				first.Arguments,
			)
		}
		if strings.TrimSpace(first.Executable) == "" {
			t.Fatal("execution spec did not resolve an executable")
		}
		if first.WorkingDirectory != suite.ExecutionRequest.WorkingDirectory {
			t.Fatalf(
				"working directory changed from %q to %q",
				suite.ExecutionRequest.WorkingDirectory,
				first.WorkingDirectory,
			)
		}
		if !reflect.DeepEqual(
			first.Environment,
			suite.ExecutionRequest.Environment,
		) {
			t.Fatalf(
				"environment overlay changed\nexpected: %#v\ngot:      %#v",
				suite.ExecutionRequest.Environment,
				first.Environment,
			)
		}
		if first.InputMode == "" || first.OutputMode == "" {
			t.Fatalf("execution stream modes are incomplete: %#v", first)
		}
	})
}

func hasCapability(
	capabilities []model.Capability,
	expected model.Capability,
) bool {
	for _, capability := range capabilities {
		if capability == expected {
			return true
		}
	}

	return false
}
