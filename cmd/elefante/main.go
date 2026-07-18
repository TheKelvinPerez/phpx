package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/elefantephp/elefante/internal/app"
	"github.com/elefantephp/elefante/internal/cli"
	"github.com/elefantephp/elefante/internal/composer"
	"github.com/elefantephp/elefante/internal/executor"
	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/paths"
	"github.com/elefantephp/elefante/internal/providers"
	"github.com/elefantephp/elefante/internal/providers/ddev"
	"github.com/elefantephp/elefante/internal/providers/native"
	"github.com/elefantephp/elefante/internal/security"
	"github.com/elefantephp/elefante/internal/state"
	"github.com/elefantephp/elefante/internal/version"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	executablePath, _ := os.Executable()
	registeredProviders := []providers.Provider{
		ddev.New(),
		native.New(native.Dependencies{
			ProviderPath: executablePath,
		}),
	}
	dependencies := app.Dependencies{
		Build:     version.Current(),
		Providers: registeredProviders,
	}
	userPaths, pathErr := paths.CurrentUserPaths()
	if pathErr != nil {
		dependencies.ApplySynchronization = func(
			context.Context,
			app.SyncExecution,
		) (app.SyncResult, error) {
			return app.SyncResult{}, model.WrapError(
				model.ErrorState,
				"Could not initialize Elefante user paths.",
				pathErr,
			)
		}
	} else {
		managedComposer := composer.NewManager(composer.ManagerOptions{
			CacheRoot: userPaths.CacheRoot,
		})
		redactor := security.NewEnvironmentRedactor(os.Environ())
		store := state.NewStore(userPaths, redactor)
		lockManager := state.NewLockManager(userPaths)
		actionService := app.NewSyncActionService(
			app.SyncActionServiceDependencies{
				Providers:       registeredProviders,
				Runner:          executor.OSRunner{},
				AcquireComposer: managedComposer.Acquire,
			},
		)
		engine := app.NewSyncEngine(app.SyncEngineDependencies{
			State: store,
			AcquireLock: func(identity string) (app.SyncLock, error) {
				return lockManager.Acquire(identity)
			},
			ExecuteAction:    actionService.Execute,
			CompensateAction: actionService.Compensate,
		})
		dependencies.ManagedComposer = managedComposer
		dependencies.ApplySynchronization = engine.Apply
	}
	application := app.New(dependencies)

	exitCode := cli.Execute(ctx, cli.Dependencies{
		Application: application,
	}, cli.Execution{
		Arguments:   os.Args[1:],
		Environment: os.Environ(),
		Input:       os.Stdin,
		Output:      os.Stdout,
		Error:       os.Stderr,
	})
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
