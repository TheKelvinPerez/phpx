package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/elefantephp/elefante/internal/app"
	"github.com/elefantephp/elefante/internal/cli"
	"github.com/elefantephp/elefante/internal/version"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	application := app.New(app.Dependencies{
		Build: version.Current(),
	})
	root := cli.NewRootCommand(cli.Dependencies{
		Application: application,
	})

	if err := root.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
