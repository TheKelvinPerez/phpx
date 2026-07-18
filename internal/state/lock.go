package state

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"syscall"

	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/paths"
)

const environmentLockSchemaVersion = "elefante.environment-lock/v1"

type LockManager struct {
	Paths        paths.UserPaths
	PID          int
	ProcessAlive func(pid int) bool
}

type EnvironmentLock struct {
	path     string
	metadata lockMetadata
	released atomic.Bool
}

type lockMetadata struct {
	SchemaVersion   string `json:"schema_version"`
	ProjectIdentity string `json:"project_identity"`
	PID             int    `json:"pid"`
	Nonce           string `json:"nonce"`
}

func NewLockManager(userPaths paths.UserPaths) LockManager {
	return LockManager{
		Paths: userPaths,
		PID:   os.Getpid(),
	}
}

func (manager LockManager) Acquire(identity string) (*EnvironmentLock, error) {
	projectPaths, err := manager.Paths.Project(identity)
	if err != nil {
		return nil, stateError(
			"Could not resolve the environment lock path.",
			"",
			err,
		)
	}
	if manager.PID <= 0 {
		manager.PID = os.Getpid()
	}
	if manager.ProcessAlive == nil {
		manager.ProcessAlive = processAlive
	}
	if err := os.MkdirAll(manager.Paths.LocksRoot, 0o700); err != nil {
		return nil, stateError(
			"Could not create the environment lock directory.",
			manager.Paths.LocksRoot,
			err,
		)
	}
	if err := os.Chmod(manager.Paths.LocksRoot, 0o700); err != nil {
		return nil, stateError(
			"Could not secure the environment lock directory.",
			manager.Paths.LocksRoot,
			err,
		)
	}

	for range 8 {
		metadata, err := newLockMetadata(identity, manager.PID)
		if err != nil {
			return nil, stateError(
				"Could not create environment lock ownership metadata.",
				projectPaths.Lock,
				err,
			)
		}
		created, err := createLockFile(projectPaths.Lock, metadata)
		if err != nil {
			return nil, stateError(
				"Could not create the environment lock.",
				projectPaths.Lock,
				err,
			)
		}
		if created {
			return &EnvironmentLock{
				path:     projectPaths.Lock,
				metadata: metadata,
			}, nil
		}

		recovered, err := recoverStaleLock(
			projectPaths.Lock,
			identity,
			manager.ProcessAlive,
		)
		if err != nil {
			return nil, err
		}
		if recovered {
			continue
		}

		return nil, lockHeldError(projectPaths.Lock)
	}

	return nil, stateError(
		"Could not acquire the environment lock after stale recovery.",
		projectPaths.Lock,
		nil,
	)
}

func (lock *EnvironmentLock) OwnerPID() int {
	return lock.metadata.PID
}

func (lock *EnvironmentLock) Release() error {
	if lock == nil || lock.released.Load() {
		return nil
	}
	content, err := os.ReadFile(lock.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return stateError(
				"The environment lock disappeared before release.",
				lock.path,
				err,
			)
		}

		return stateError("Could not read the environment lock.", lock.path, err)
	}
	var current lockMetadata
	if err := json.Unmarshal(content, &current); err != nil {
		return stateError("Environment lock metadata is invalid.", lock.path, err)
	}
	if current != lock.metadata {
		return stateError(
			"Environment lock ownership changed before release.",
			lock.path,
			nil,
		)
	}
	if err := os.Remove(lock.path); err != nil {
		return stateError("Could not release the environment lock.", lock.path, err)
	}
	lock.released.Store(true)
	if err := flushDirectory(filepath.Dir(lock.path)); err != nil {
		return stateError(
			"Could not flush the released environment lock.",
			lock.path,
			err,
		)
	}
	return nil
}

func newLockMetadata(identity string, pid int) (lockMetadata, error) {
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return lockMetadata{}, fmt.Errorf("generate lock nonce: %w", err)
	}

	return lockMetadata{
		SchemaVersion:   environmentLockSchemaVersion,
		ProjectIdentity: identity,
		PID:             pid,
		Nonce:           hex.EncodeToString(nonceBytes),
	}, nil
}

func createLockFile(path string, metadata lockMetadata) (bool, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	keep := false
	defer func() {
		_ = file.Close()
		if !keep {
			_ = os.Remove(path)
		}
	}()

	encoded, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return false, fmt.Errorf("encode lock metadata: %w", err)
	}
	encoded = append(encoded, '\n')
	if _, err := file.Write(encoded); err != nil {
		return false, fmt.Errorf("write lock metadata: %w", err)
	}
	if err := file.Sync(); err != nil {
		return false, fmt.Errorf("flush lock metadata: %w", err)
	}
	if err := file.Close(); err != nil {
		return false, fmt.Errorf("close lock metadata: %w", err)
	}
	if err := flushDirectory(filepath.Dir(path)); err != nil {
		return false, fmt.Errorf("flush lock directory: %w", err)
	}
	keep = true

	return true, nil
}

func recoverStaleLock(
	path string,
	identity string,
	alive func(pid int) bool,
) (bool, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, stateError("Could not inspect the environment lock.", path, err)
	}
	var owner lockMetadata
	if err := json.Unmarshal(content, &owner); err != nil {
		return false, stateError("Environment lock metadata is invalid.", path, err)
	}
	if owner.SchemaVersion != environmentLockSchemaVersion ||
		owner.ProjectIdentity != identity ||
		owner.PID <= 0 ||
		owner.Nonce == "" {
		return false, stateError(
			"Environment lock metadata is not valid for this project.",
			path,
			nil,
		)
	}
	if alive(owner.PID) {
		return false, lockHeldByOwnerError(path, owner.PID)
	}

	current, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, stateError("Could not verify the stale environment lock.", path, err)
	}
	if !bytes.Equal(content, current) {
		return true, nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, stateError("Could not remove the stale environment lock.", path, err)
	}
	if err := flushDirectory(filepath.Dir(path)); err != nil {
		return false, stateError("Could not flush stale lock recovery.", path, err)
	}

	return true, nil
}

func lockHeldError(path string) *model.Error {
	return stateError(
		"Another mutating operation owns the environment lock.",
		path,
		nil,
	)
}

func lockHeldByOwnerError(path string, pid int) *model.Error {
	commandError := lockHeldError(path)
	commandError.Details = []model.ErrorDetail{
		{Name: "owner_pid", Value: strconv.Itoa(pid)},
	}

	return commandError
}

func processAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))

	return err == nil || errors.Is(err, syscall.EPERM)
}
