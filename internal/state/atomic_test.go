package state_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/elefantephp/elefante/internal/state"
)

func TestAtomicWriterPreservesPreviousStateWhenCommitIsInterrupted(t *testing.T) {
	t.Parallel()

	target := filepath.Join(t.TempDir(), "state", "environment.json")
	if err := state.WriteJSON(target, map[string]string{
		"value": "previous",
	}); err != nil {
		t.Fatalf("write initial state: %v", err)
	}

	interrupted := state.AtomicWriter{
		BeforeRename: func(string, string) error {
			return errors.New("simulated interruption")
		},
	}
	err := interrupted.WriteJSON(target, map[string]string{
		"value": "replacement",
	})
	if err == nil {
		t.Fatal("expected interrupted write to fail")
	}

	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read preserved state: %v", err)
	}
	if string(content) != "{\n  \"value\": \"previous\"\n}\n" {
		t.Fatalf("previous state was not preserved: %s", content)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("inspect state permissions: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected state mode 0600, got %04o", info.Mode().Perm())
	}
	directoryInfo, err := os.Stat(filepath.Dir(target))
	if err != nil {
		t.Fatalf("inspect state directory permissions: %v", err)
	}
	if directoryInfo.Mode().Perm() != 0o700 {
		t.Fatalf(
			"expected state directory mode 0700, got %04o",
			directoryInfo.Mode().Perm(),
		)
	}
}
