package discovery

import "testing"

func FuzzValidateJSONObject(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte("{}"),
		[]byte(`{"name":"acme/example"}`),
		[]byte(`{"config":{"platform":{"php":"8.4"}}}`),
		[]byte(`{"duplicate":1,"duplicate":2}`),
		[]byte("[]"),
		[]byte("{"),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, content []byte) {
		first := validateJSONObject(content)
		second := validateJSONObject(content)

		if (first == nil) != (second == nil) {
			t.Fatalf("validation was not deterministic: first %v, second %v", first, second)
		}
		if first != nil && first.Error() != second.Error() {
			t.Fatalf("validation errors differ: first %q, second %q", first, second)
		}
	})
}
