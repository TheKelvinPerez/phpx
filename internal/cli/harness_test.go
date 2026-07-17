package cli_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/elefantephp/elefante/internal/cli"
	"github.com/elefantephp/elefante/internal/output"
	"github.com/spf13/cobra"
)

func executeCommand(
	t *testing.T,
	ctx context.Context,
	input io.Reader,
	dependencies cli.Dependencies,
	arguments ...string,
) (*cobra.Command, string, string, error) {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	dependencies.Renderer = output.NewHumanRenderer(&stdout, &stderr)
	command := cli.NewRootCommand(dependencies)
	command.SetArgs(arguments)
	command.SetIn(input)
	command.SetOut(&stdout)
	command.SetErr(&stderr)

	err := command.ExecuteContext(ctx)

	return command, stdout.String(), stderr.String(), err
}
