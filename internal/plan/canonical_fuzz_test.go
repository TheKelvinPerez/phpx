package plan_test

import (
	"encoding/json"
	"testing"

	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/plan"
)

func FuzzBuildCanonicalization(f *testing.F) {
	f.Add(uint8(0))
	f.Add(uint8(0xff))
	f.Add(uint8(0x55))

	f.Fuzz(func(t *testing.T, permutation uint8) {
		canonical := canonicalizationRequest()
		permuted := clonePlanRequest(t, canonical)

		if permutation&1 != 0 {
			reverse(permuted.Facts.Composer.PlatformRequirements)
			reverse(permuted.Facts.InputFingerprints)
		}
		if permutation&2 != 0 {
			reverse(permuted.Observations)
		}
		if permutation&4 != 0 {
			reverse(permuted.Facts.Composer.Plugins)
			reverse(permuted.Facts.Composer.Scripts)
		}
		if permutation&8 != 0 {
			reverse(permuted.Facts.Diagnostics)
			for index := range permuted.Facts.Diagnostics {
				reverse(permuted.Facts.Diagnostics[index].Sources)
			}
		}
		for index := range permuted.Observations {
			if permutation&16 != 0 {
				reverse(permuted.Observations[index].Capabilities)
				reverse(permuted.Observations[index].Runtimes)
			}
			if permutation&32 != 0 {
				reverse(permuted.Observations[index].Extensions)
				reverse(permuted.Observations[index].Composer)
			}
			if permutation&64 != 0 {
				reverse(permuted.Observations[index].Diagnostics)
			}
		}

		first, err := plan.Build(canonical)
		if err != nil {
			t.Fatalf("build canonical plan: %v", err)
		}
		second, err := plan.Build(permuted)
		if err != nil {
			t.Fatalf("build permuted plan: %v", err)
		}
		firstJSON, err := json.Marshal(first)
		if err != nil {
			t.Fatalf("encode canonical plan: %v", err)
		}
		secondJSON, err := json.Marshal(second)
		if err != nil {
			t.Fatalf("encode permuted plan: %v", err)
		}
		if string(firstJSON) != string(secondJSON) {
			t.Fatalf(
				"equivalent inputs produced different plans\ncanonical: %s\npermuted:  %s",
				firstJSON,
				secondJSON,
			)
		}
	})
}

func canonicalizationRequest() plan.Request {
	request := compatibleRequest()
	request.Provider = "native"
	request.Facts.InputFingerprints = append(
		request.Facts.InputFingerprints,
		model.InputFingerprint{
			Path:   "/workspace/composer.lock",
			Kind:   "composer_lock",
			SHA256: "lock-digest",
			Size:   256,
		},
	)
	request.Facts.Composer.Plugins = []model.ComposerPlugin{
		{Name: "z/plugin", Version: "1.0.0"},
		{Name: "a/plugin", Version: "2.0.0"},
	}
	request.Facts.Composer.Scripts = []model.ComposerScript{
		{Name: "post-install-cmd", CommandCount: 1, ContentSHA256: "sha256:z"},
		{Name: "pre-install-cmd", CommandCount: 2, ContentSHA256: "sha256:a"},
	}
	request.Facts.Diagnostics = []model.Diagnostic{
		{
			Code:     "ELEFANTE_FIXTURE",
			Severity: model.SeverityWarning,
			Message:  "First.",
			Sources: []model.SourceReference{
				{Path: "/workspace/z", Kind: "fixture"},
				{Path: "/workspace/a", Kind: "fixture"},
			},
		},
		{
			Code:     "ELEFANTE_FIXTURE",
			Severity: model.SeverityWarning,
			Message:  "Second.",
			Sources: []model.SourceReference{
				{Path: "/workspace/b", Kind: "fixture"},
			},
		},
	}
	request.Observations[0].Capabilities = []model.Capability{
		model.CapabilityInspectRuntime,
		model.CapabilityInspectComposer,
	}
	request.Observations[0].Composer = append(
		request.Observations[0].Composer,
		model.ComposerObservation{
			Version:  "2.8.8",
			Source:   "managed",
			Identity: "sha256:managed-composer",
		},
	)
	ddev := request.Observations[0]
	ddev.Provider = "ddev"
	ddev.Fingerprint = "sha256:ddev"
	request.Observations = append(request.Observations, ddev)

	return request
}

func clonePlanRequest(t *testing.T, request plan.Request) plan.Request {
	t.Helper()

	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("encode plan request: %v", err)
	}
	var clone plan.Request
	if err := json.Unmarshal(encoded, &clone); err != nil {
		t.Fatalf("decode plan request: %v", err)
	}

	return clone
}

func reverse[T any](values []T) {
	for left, right := 0, len(values)-1; left < right; left, right =
		left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}
