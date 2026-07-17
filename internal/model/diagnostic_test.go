package model_test

import (
	"encoding/json"
	"testing"

	"github.com/elefantephp/elefante/internal/model"
)

func TestDiagnosticMarshalsStablePublicFields(t *testing.T) {
	diagnostic := model.Diagnostic{
		Code:      "ELEFANTE_CONFIG_CONFLICT",
		Severity:  model.SeverityWarning,
		Message:   "Conflicting project configuration",
		Detail:    "composer.json and .php-version require different PHP versions",
		Hint:      "Align the declared PHP requirements.",
		Sources:   []model.SourceReference{{Path: "composer.json", Kind: "composer"}},
		Provider:  "native",
		Retryable: false,
	}

	encoded, err := json.Marshal(diagnostic)
	if err != nil {
		t.Fatalf("marshal diagnostic: %v", err)
	}

	expected := `{"code":"ELEFANTE_CONFIG_CONFLICT","severity":"warning","message":"Conflicting project configuration","detail":"composer.json and .php-version require different PHP versions","hint":"Align the declared PHP requirements.","sources":[{"path":"composer.json","kind":"composer"}],"provider":"native","retryable":false}`
	if string(encoded) != expected {
		t.Fatalf("expected stable diagnostic JSON\nexpected: %s\ngot:      %s", expected, encoded)
	}
}
