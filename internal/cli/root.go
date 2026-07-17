package cli

import (
	"fmt"

	"github.com/elefantephp/elefante/internal/app"
	"github.com/spf13/cobra"
)

type Dependencies struct {
	Application *app.Application
}

func NewRootCommand(dependencies Dependencies) *cobra.Command {
	root := &cobra.Command{
		Use:   "elefante",
		Short: "The local development runtime for PHP.",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return command.Help()
		},
	}

	root.AddCommand(newVersionCommand(dependencies.Application))

	return root
}

func newVersionCommand(application *app.Application) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the Elefante version",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			build := application.Version(command.Context())
			_, err := fmt.Fprintf(command.OutOrStdout(), "elefante %s\n", build.Version)
			return err
		},
	}
}
