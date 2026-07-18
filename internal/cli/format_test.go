package cli

import (
	"strings"
	"testing"

	"github.com/elefantephp/elefante/internal/model"
)

func TestFormatProviderObservationIncludesAvailabilityStateAndEngine(t *testing.T) {
	output := formatProviderObservation(model.ProviderObservation{
		Provider:     "ddev",
		Available:    true,
		Version:      "1.24.8",
		Platform:     "darwin",
		Architecture: "arm64",
		State:        model.ProviderStateRunning,
		Engines: []model.EngineObservation{
			{
				Name:     "docker",
				Version:  "29.4.0",
				Platform: "orbstack",
			},
		},
		Capabilities: []model.Capability{},
		Fingerprint:  "sha256:ddev",
	})

	for _, expected := range []string{
		"Provider observation: ddev",
		"Available: yes",
		"Version: 1.24.8",
		"State: running",
		"Platform: darwin/arm64",
		"Engine: docker 29.4.0, orbstack",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, output)
		}
	}
}
