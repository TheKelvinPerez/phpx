package paths_test

import (
	"os"
	"path/filepath"
	"testing"

	projectpaths "github.com/elefantephp/elefante/internal/paths"
)

func TestNormalizeStartResolvesSymlinkAndRetainsSuppliedPath(t *testing.T) {
	target := t.TempDir()
	alias := filepath.Join(t.TempDir(), "project")
	if err := os.Symlink(target, alias); err != nil {
		t.Fatalf("create path symlink: %v", err)
	}

	normalized, err := projectpaths.NormalizeStart(alias)
	if err != nil {
		t.Fatalf("normalize path: %v", err)
	}
	resolvedTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatalf("resolve target: %v", err)
	}
	absoluteAlias, err := filepath.Abs(alias)
	if err != nil {
		t.Fatalf("make alias absolute: %v", err)
	}

	if normalized.Supplied != alias {
		t.Errorf("expected supplied path %q, got %q", alias, normalized.Supplied)
	}
	if normalized.Absolute != filepath.Clean(absoluteAlias) {
		t.Errorf("expected absolute path %q, got %q", absoluteAlias, normalized.Absolute)
	}
	if normalized.Resolved != resolvedTarget {
		t.Errorf("expected resolved path %q, got %q", resolvedTarget, normalized.Resolved)
	}
	if normalized.Directory != resolvedTarget {
		t.Errorf("expected directory %q, got %q", resolvedTarget, normalized.Directory)
	}
}

func TestNormalizeStartUsesContainingDirectoryForFile(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "composer.json")
	if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	normalized, err := projectpaths.NormalizeStart(path)
	if err != nil {
		t.Fatalf("normalize file path: %v", err)
	}
	resolvedDirectory, err := filepath.EvalSymlinks(directory)
	if err != nil {
		t.Fatalf("resolve directory: %v", err)
	}

	if normalized.Directory != resolvedDirectory {
		t.Errorf("expected containing directory %q, got %q", resolvedDirectory, normalized.Directory)
	}
}

func TestContainsUsesPathBoundaries(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "projects", "elefante")

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"root", root, true},
		{"descendant", filepath.Join(root, "src", "Domain"), true},
		{"parent", filepath.Dir(root), false},
		{"sibling with common prefix", root + "-copy", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := projectpaths.Contains(root, test.path); got != test.expected {
				t.Errorf("expected Contains(%q, %q) to be %t, got %t", root, test.path, test.expected, got)
			}
		})
	}
}
