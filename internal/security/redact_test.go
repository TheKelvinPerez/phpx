package security_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/elefantephp/elefante/internal/security"
)

func TestRedactorRemovesSecretsFromStructuredOutput(t *testing.T) {
	t.Parallel()

	const secret = "synthetic-secret-value"
	redactor := security.NewRedactor(secret)
	payload := map[string]any{
		"authorization": "Bearer " + secret,
		"cookie":        "session=" + secret,
		"environment": []string{
			"APP_ENV=local",
			"API_TOKEN=" + secret,
			`COMPOSER_AUTH={"github-oauth":{"github.com":"` + secret + `"}}`,
		},
		"message": "Authorization: Bearer " + secret,
		"url":     "https://user:" + secret + "@packages.example.test/archive.zip?token=" + secret + "&channel=stable",
	}

	sanitized, err := redactor.Value(payload)
	if err != nil {
		t.Fatalf("redact payload: %v", err)
	}
	encoded, err := json.Marshal(sanitized)
	if err != nil {
		t.Fatalf("encode redacted payload: %v", err)
	}
	output := string(encoded)

	if strings.Contains(output, secret) {
		t.Fatalf("raw secret survived redaction: %s", output)
	}
	for _, expected := range []string{
		`"authorization":"[REDACTED]"`,
		`"cookie":"[REDACTED]"`,
		`"APP_ENV=local"`,
		`"API_TOKEN=[REDACTED]"`,
		`"COMPOSER_AUTH=[REDACTED]"`,
		`"message":"Authorization: Bearer [REDACTED]"`,
		`channel=stable`,
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("expected redacted output to contain %q, got %s", expected, output)
		}
	}
}

func TestEnvironmentRedactorFindsSensitiveAndComposerAuthValues(t *testing.T) {
	t.Parallel()

	const token = "environment-token"
	const composerToken = "composer-auth-token"
	redactor := security.NewEnvironmentRedactor([]string{
		"APP_ENV=local",
		"API_TOKEN=" + token,
		`COMPOSER_AUTH={"github-oauth":{"github.com":"` + composerToken + `"}}`,
	})

	output := redactor.Text(
		"tokens: " + token + ", " + composerToken + ", environment: local",
	)
	if strings.Contains(output, token) || strings.Contains(output, composerToken) {
		t.Fatalf("environment secret survived redaction: %s", output)
	}
	if !strings.Contains(output, "environment: local") {
		t.Fatalf("safe environment content was removed: %s", output)
	}
}

func TestRedactorRecognizesSensitivePairsHeadersAndEmbeddedURLs(t *testing.T) {
	t.Parallel()

	const secret = "unregistered-sensitive-value"
	redactor := security.NewRedactor()
	payload := map[string]any{
		"inputs": []map[string]string{
			{
				"name":  "PRIVATE_TOKEN",
				"value": secret,
			},
		},
		"message": "Fetch https://user:" + secret +
			"@packages.example.test/archive?api_key=" + secret +
			" failed\nCookie: session=" + secret + "; preference=safe",
	}

	encoded, err := redactor.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal redacted surfaces: %v", err)
	}
	output := string(encoded)

	if strings.Contains(output, secret) {
		t.Fatalf("unregistered sensitive value survived redaction: %s", output)
	}
	if !strings.Contains(output, `"name":"PRIVATE_TOKEN","value":"[REDACTED]"`) {
		t.Fatalf("sensitive name and value pair was not preserved safely: %s", output)
	}
	if !strings.Contains(output, `Cookie: [REDACTED]`) {
		t.Fatalf("Cookie header was not redacted: %s", output)
	}
}
