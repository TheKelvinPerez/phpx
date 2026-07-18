package plan_test

import (
	"testing"

	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/plan"
)

func TestComposerTrustFingerprintInvalidatesOnEveryExecutionInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		change func(*plan.Request)
	}{
		{
			name: "manifest digest",
			change: func(request *plan.Request) {
				request.Facts.InputFingerprints[0].SHA256 = "manifest-changed"
			},
		},
		{
			name: "lock digest",
			change: func(request *plan.Request) {
				request.Facts.InputFingerprints[1].SHA256 = "lock-changed"
			},
		},
		{
			name: "relevant configuration digest",
			change: func(request *plan.Request) {
				request.Facts.InputFingerprints[2].SHA256 = "config-changed"
			},
		},
		{
			name: "Composer executable identity",
			change: func(request *plan.Request) {
				request.Observations[0].Composer[0].Identity = "sha256:composer-changed"
			},
		},
		{
			name: "plugin identity",
			change: func(request *plan.Request) {
				request.Facts.Composer.Plugins[0].Version = "2.0.0"
			},
		},
		{
			name: "script definition",
			change: func(request *plan.Request) {
				request.Facts.Composer.Scripts[0].ContentSHA256 = "sha256:script-changed"
			},
		},
	}

	baseline := composerTrustFingerprint(t, trustRequest())
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			request := trustRequest()
			test.change(&request)
			changed := composerTrustFingerprint(t, request)
			if changed == baseline {
				t.Fatalf("%s did not invalidate Composer trust", test.name)
			}
		})
	}
}

func TestComposerTrustFingerprintIgnoresNonexecutionInputs(t *testing.T) {
	t.Parallel()

	first := trustRequest()
	first.Facts.InputFingerprints = append(
		first.Facts.InputFingerprints,
		model.InputFingerprint{
			Path:   "/workspace/.php-version",
			Kind:   "php_version",
			SHA256: "php-version-one",
			Size:   4,
		},
	)
	second := trustRequest()
	second.Facts.InputFingerprints = append(
		second.Facts.InputFingerprints,
		model.InputFingerprint{
			Path:   "/workspace/.php-version",
			Kind:   "php_version",
			SHA256: "php-version-two",
			Size:   4,
		},
	)

	firstFingerprint := composerTrustFingerprint(t, first)
	secondFingerprint := composerTrustFingerprint(t, second)
	if firstFingerprint != secondFingerprint {
		t.Fatal("PHP version input unexpectedly invalidated Composer code trust")
	}
}

func trustRequest() plan.Request {
	request := compatibleRequest()
	request.Facts.InputFingerprints = append(
		request.Facts.InputFingerprints,
		model.InputFingerprint{
			Path:   "/workspace/composer.lock",
			Kind:   "composer_lock",
			SHA256: "lock-digest",
			Size:   256,
		},
		model.InputFingerprint{
			Path:   "/workspace/elefante.toml",
			Kind:   "elefante_config",
			SHA256: "config-digest",
			Size:   64,
		},
	)
	request.Facts.Composer.Plugins = []model.ComposerPlugin{
		{
			Name:    "acme/plugin",
			Version: "1.0.0",
		},
	}
	request.Facts.Composer.Scripts = []model.ComposerScript{
		{
			Name:          "post-install-cmd",
			CommandCount:  1,
			ContentSHA256: "sha256:script",
		},
	}

	return request
}

func composerTrustFingerprint(t *testing.T, request plan.Request) string {
	t.Helper()

	builtPlan, err := plan.Build(request)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if len(builtPlan.Trust) == 0 {
		t.Fatal("expected Composer trust requirements")
	}

	return builtPlan.Trust[0].Fingerprint
}
