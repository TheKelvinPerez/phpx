package state_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/elefantephp/elefante/internal/state"
)

func TestEnvironmentLockAllowsOnlyOneConcurrentOwner(t *testing.T) {
	t.Parallel()

	manager := state.NewLockManager(testUserPaths(t))
	manager.PID = 1001
	manager.ProcessAlive = func(int) bool {
		return true
	}

	const contenders = 24
	start := make(chan struct{})
	release := make(chan struct{})
	attempted := make(chan struct{}, contenders)
	var acquired atomic.Int32
	var attempts sync.WaitGroup
	attempts.Add(contenders)
	for range contenders {
		go func() {
			defer attempts.Done()
			<-start
			lock, err := manager.Acquire("sha256:project")
			if err != nil {
				attempted <- struct{}{}
				return
			}
			acquired.Add(1)
			attempted <- struct{}{}
			<-release
			_ = lock.Release()
		}()
	}

	close(start)
	for range contenders {
		<-attempted
	}
	close(release)
	attempts.Wait()

	if acquired.Load() != 1 {
		t.Fatalf("expected one lock owner, got %d", acquired.Load())
	}
}

func TestEnvironmentLockRecoversOnlyAfterOwnerIsStale(t *testing.T) {
	t.Parallel()

	userPaths := testUserPaths(t)
	first := state.NewLockManager(userPaths)
	first.PID = 1001
	first.ProcessAlive = func(pid int) bool {
		return pid == 1001
	}
	if _, err := first.Acquire("sha256:project"); err != nil {
		t.Fatalf("acquire original lock: %v", err)
	}

	second := state.NewLockManager(userPaths)
	second.PID = 2002
	second.ProcessAlive = func(pid int) bool {
		return pid == 2002
	}
	recovered, err := second.Acquire("sha256:project")
	if err != nil {
		t.Fatalf("recover stale lock: %v", err)
	}
	if recovered.OwnerPID() != 2002 {
		t.Fatalf("expected recovered owner 2002, got %d", recovered.OwnerPID())
	}
	if err := recovered.Release(); err != nil {
		t.Fatalf("release recovered lock: %v", err)
	}
}
