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

	exitCode := cli.Execute(ctx, cli.Dependencies{
		Application: application,
	}, cli.Execution{
		Arguments: os.Args[1:],
		Input:     os.Stdin,
		Output:    os.Stdout,
		Error:     os.Stderr,
	})
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
