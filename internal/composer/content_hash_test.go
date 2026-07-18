package composer

import "testing"

func TestComposerContentHashMatchesComposerEncoding(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "numeric and unicode extra values",
			content: `{
				"extra": {
					"ratio": 1.0,
					"threshold": 1e6,
					"tiny": 1e-7,
					"url": "https://example.invalid/é/🐘"
				},
				"name": "acme/example",
				"config": {
					"platform": {
						"php": "8.4.10"
					}
				}
			}`,
			expected: "aa58661d82495c9e809d284cca63a181",
		},
		{
			name:     "null relevant fields are omitted",
			content:  `{"extra":null}`,
			expected: "99914b932bd37a50b983c5e7c90ae93b",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := composerContentHash([]byte(test.content))
			if err != nil {
				t.Fatalf("compute Composer content hash: %v", err)
			}
			if actual != test.expected {
				t.Fatalf("expected content hash %q, got %q", test.expected, actual)
			}
		})
	}
}

func TestEncodePHPJSONNumberMatchesComposerRuntime(t *testing.T) {
	tests := map[string]string{
		"0.1":                "0.1",
		"0.0001":             "0.0001",
		"0.00001":            "1.0e-5",
		"1e16":               "10000000000000000",
		"1e17":               "1.0e+17",
		"-0.0":               "-0",
		"1.2345678901234567": "1.2345678901234567",
		"1e-6":               "1.0e-6",
		"1e20":               "1.0e+20",
		"9007199254740993.0": "9007199254740992",
		"1e-400":             "0",
		"-1e-400":            "-0",
	}

	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			actual, err := encodePHPJSONNumber(input)
			if err != nil {
				t.Fatalf("encode PHP JSON number %q: %v", input, err)
			}
			if actual != expected {
				t.Fatalf("expected %q, got %q", expected, actual)
			}
		})
	}
}

func TestEncodePHPJSONNumberRejectsOverflow(t *testing.T) {
	for _, input := range []string{"1e400", "-1e400"} {
		if _, err := encodePHPJSONNumber(input); err == nil {
			t.Fatalf("expected %q to exceed finite JSON number range", input)
		}
	}
}
