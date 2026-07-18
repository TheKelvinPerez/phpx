package plan_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/plan"
)

func TestBuildMatchesPlanGoldens(t *testing.T) {
	cases := []struct {
		name    string
		request func() plan.Request
	}{
		{name: "compatible", request: compatibleRequest},
		{
			name: "incompatible",
			request: func() plan.Request {
				request := compatibleRequest()
				request.Observations[0].Runtimes[0].Version = "8.3.19"

				return request
			},
		},
		{
			name: "ambiguous",
			request: func() plan.Request {
				request := compatibleRequest()
				request.Facts.Configuration.Providers.Allowed = nil
				ddev := request.Observations[0]
				ddev.Provider = "ddev"
				ddev.Fingerprint = "sha256:ddev"
				request.Observations = append(request.Observations, ddev)

				return request
			},
		},
		{
			name: "legacy",
			request: func() plan.Request {
				request := compatibleRequest()
				request.Facts.Composer.PlatformRequirements[0].Constraint = "^8.2"
				request.Observations[0].Runtimes[0].Version = "8.2.29"

				return request
			},
		},
		{
			name: "optional-extension",
			request: func() plan.Request {
				request := compatibleRequest()
				request.Facts.Configuration.Path = "/workspace/elefante.toml"
				request.Facts.Configuration.Extensions.Optional = []string{
					"ext-xdebug",
				}

				return request
			},
		},
		{
			name: "offline",
			request: func() plan.Request {
				request := compatibleRequest()
				request.Policy.Offline = true

				return request
			},
		},
		{
			name: "frozen",
			request: func() plan.Request {
				request := compatibleRequest()
				request.Policy.Frozen = true
				request.Facts.Composer.Lock = model.ComposerLockFacts{
					Status: model.ComposerLockMissing,
				}

				return request
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			first := buildGoldenPlan(t, testCase.request())
			second := buildGoldenPlan(t, testCase.request())
			if !bytes.Equal(first, second) {
				t.Fatalf(
					"plan golden %q was not byte identical across builds\nfirst:\n%s\nsecond:\n%s",
					testCase.name,
					first,
					second,
				)
			}

			path := filepath.Join(
				"..",
				"..",
				"testdata",
				"golden",
				"plan",
				testCase.name+".json",
			)
			expected, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf(
					"read plan golden %s: %v\nactual:\n%s",
					path,
					err,
					first,
				)
			}
			if !bytes.Equal(first, expected) {
				t.Fatalf(
					"plan golden %q changed\nexpected:\n%s\nactual:\n%s",
					testCase.name,
					expected,
					first,
				)
			}
		})
	}
}

func buildGoldenPlan(t *testing.T, request plan.Request) []byte {
	t.Helper()

	result, err := plan.Build(request)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("encode plan: %v", err)
	}

	return append(encoded, '\n')
}
