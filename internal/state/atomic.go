package state

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/elefantephp/elefante/internal/security"
)

type AtomicWriter struct {
	BeforeRename func(temporaryPath string, targetPath string) error
	Redactor     security.Redactor
}

func WriteJSON(path string, value any) error {
	return (AtomicWriter{}).WriteJSON(path, value)
}

func (writer AtomicWriter) WriteJSON(path string, value any) error {
	encoded, err := writer.Redactor.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, encoded, "", "  "); err != nil {
		return fmt.Errorf("format state: %w", err)
	}
	formatted.WriteByte('\n')

	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return fmt.Errorf("secure state directory: %w", err)
	}

	temporary, err := os.CreateTemp(directory, ".elefante-state-*")
	if err != nil {
		return fmt.Errorf("create temporary state file: %w", err)
	}
	temporaryPath := temporary.Name()
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()

	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()

		return fmt.Errorf("secure temporary state file: %w", err)
	}
	if _, err := temporary.Write(formatted.Bytes()); err != nil {
		_ = temporary.Close()

		return fmt.Errorf("write temporary state file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()

		return fmt.Errorf("flush temporary state file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary state file: %w", err)
	}
	if writer.BeforeRename != nil {
		if err := writer.BeforeRename(temporaryPath, path); err != nil {
			return fmt.Errorf("commit state write: %w", err)
		}
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace state file: %w", err)
	}
	removeTemporary = false

	if err := flushDirectory(directory); err != nil {
		return fmt.Errorf("flush state directory: %w", err)
	}

	return nil
}

func flushDirectory(directory string) error {
	directoryHandle, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer directoryHandle.Close()
	if err := directoryHandle.Sync(); err != nil {
		return err
	}

	return nil
}
