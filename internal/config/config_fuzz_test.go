package config_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/elefantephp/elefante/internal/config"
)

func FuzzConfigurationParser(f *testing.F) {
	f.Add([]byte("schema_version = 1\n"))
	f.Add([]byte("schema_version = 2\n"))
	f.Add([]byte("schema_version = 1\nunknown = true\n"))
	f.Add([]byte(`
schema_version = 1
[tasks.test]
command = ["php", "artisan", "test"]
`))
	f.Add([]byte("not valid TOML"))

	repositoryRoot := f.TempDir()
	configPath := filepath.Join(repositoryRoot, "elefante.toml")
	f.Fuzz(func(t *testing.T, content []byte) {
		document := config.Document{
			Path:           configPath,
			RepositoryRoot: repositoryRoot,
			Content:        content,
		}
		first := config.Parse(document)
		second := config.Parse(document)

		if !reflect.DeepEqual(first, second) {
			t.Fatalf(
				"configuration parsing is not deterministic\nfirst: %#v\nsecond: %#v",
				first,
				second,
			)
		}
	})
}
