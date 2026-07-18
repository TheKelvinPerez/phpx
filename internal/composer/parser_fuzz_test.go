package composer_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/elefantephp/elefante/internal/composer"
	"github.com/elefantephp/elefante/internal/model"
)

func FuzzParseProjectManifest(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`{}`),
		[]byte(`{"name":"acme/example","require":{"PHP":"^8.4"}}`),
		[]byte(`{"name":"acme/example","require":null}`),
		[]byte(`{"name":"first","name":"second"}`),
		[]byte(`{"scripts":{"test":["@php vendor/bin/phpunit"]}}`),
		[]byte(`{"name":"synthetic-fuzz-secret"`),
		{0xff, 0xfe, 0xfd},
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, content []byte) {
		input := composer.ProjectInput{
			Manifest: composer.Document{
				Path:    "/workspace/composer.json",
				Content: content,
			},
		}

		first, firstErr := composer.ParseProject(input)
		second, secondErr := composer.ParseProject(input)
		assertDeterministicParse(t, first, firstErr, second, secondErr)

		if firstErr != nil && bytes.Contains(content, []byte("synthetic-fuzz-secret")) {
			encoded, err := json.Marshal(firstErr)
			if err != nil {
				t.Fatalf("marshal parser error: %v", err)
			}
			if bytes.Contains(encoded, []byte("synthetic-fuzz-secret")) {
				t.Fatalf("parser error leaked malformed manifest content: %s", encoded)
			}
		}
	})
}

func FuzzParseProjectLock(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`{
			"content-hash":"d41d8cd98f00b204e9800998ecf8427e",
			"packages":[],
			"packages-dev":[]
		}`),
		[]byte(`{"packages":null}`),
		[]byte(`{"content-hash":"synthetic-fuzz-secret","packages":[`),
		[]byte(`{
			"content-hash":"d41d8cd98f00b204e9800998ecf8427e",
			"packages":[{"name":"acme/example","version":"1.0.0"}],
			"packages-dev":[{"name":"ACME/EXAMPLE","version":"1.1.0"}]
		}`),
		{0xff, 0xfe, 0xfd},
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, content []byte) {
		input := composer.ProjectInput{
			Manifest: composer.Document{
				Path:    "/workspace/composer.json",
				Content: []byte(`{"name":"acme/fuzz"}`),
			},
			Lock: &composer.Document{
				Path:    "/workspace/composer.lock",
				Content: content,
			},
		}

		first, firstErr := composer.ParseProject(input)
		second, secondErr := composer.ParseProject(input)
		assertDeterministicParse(t, first, firstErr, second, secondErr)

		if first.Facts.Lock.Status == model.ComposerLockInvalid &&
			bytes.Contains(content, []byte("synthetic-fuzz-secret")) {
			encoded, err := json.Marshal(first)
			if err != nil {
				t.Fatalf("marshal invalid lock result: %v", err)
			}
			if bytes.Contains(encoded, []byte("synthetic-fuzz-secret")) {
				t.Fatalf("invalid lock result leaked malformed content: %s", encoded)
			}
		}
	})
}

func assertDeterministicParse(
	t *testing.T,
	first composer.ParseResult,
	firstErr error,
	second composer.ParseResult,
	secondErr error,
) {
	t.Helper()

	if (firstErr == nil) != (secondErr == nil) {
		t.Fatalf("parser error state changed between equivalent inputs")
	}
	if firstErr != nil {
		if firstErr.Error() != secondErr.Error() {
			t.Fatalf(
				"parser error changed between equivalent inputs, first %q, second %q",
				firstErr.Error(),
				secondErr.Error(),
			)
		}

		return
	}

	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first parser result: %v", err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second parser result: %v", err)
	}
	if !bytes.Equal(firstJSON, secondJSON) {
		t.Fatalf(
			"parser result changed between equivalent inputs\nfirst: %s\nsecond: %s",
			firstJSON,
			secondJSON,
		)
	}
}
