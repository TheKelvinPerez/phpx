package security_test

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/elefantephp/elefante/internal/security"
)

func FuzzRedactorNeverEmitsRegisteredSecret(f *testing.F) {
	f.Add("synthetic-token", "prefix")
	f.Add("composer-auth-secret", "Authorization: Bearer")
	f.Add("url-password", "https://packages.example.test")

	f.Fuzz(func(t *testing.T, secretMaterial string, context string) {
		sum := sha256.Sum256([]byte(secretMaterial))
		secret := "synthetic-" + hex.EncodeToString(sum[:])

		redactor := security.NewRedactor(secret)
		text := redactor.Text(context + " " + secret)
		if strings.Contains(text, secret) {
			t.Fatalf("text redaction leaked %q in %q", secret, text)
		}

		encoded, err := redactor.Marshal(map[string]any{
			"message":       context + " " + secret,
			"authorization": "Bearer " + secret,
		})
		if err != nil {
			t.Fatalf("marshal redacted payload: %v", err)
		}
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("structured redaction leaked %q in %s", secret, encoded)
		}
	})
}
