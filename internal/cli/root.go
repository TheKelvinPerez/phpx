package cli

import (
	"fmt"

	"github.com/elefantephp/elefante/internal/app"
	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/output"
	"github.com/spf13/cobra"
)

type Dependencies struct {
	Application *app.Application
	Renderer    output.Renderer
}

func NewRootCommand(dependencies Dependencies) *cobra.Command {
	root := &cobra.Command{
		Use:           "elefante",
		Short:         "The local development runtime for PHP.",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(command *cobra.Command, _ []string) error {
			return command.Help()
		},
	}
	root.PersistentFlags().Bool("json", false, "Emit newline delimited JSON events")

	root.AddCommand(newVersionCommand(dependencies.Application, dependencies.Renderer))

	return root
}

func newVersionCommand(application *app.Application, renderer output.Renderer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the Elefante version",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			build := application.Version(command.Context())
			if err := renderer.Result(output.Result{
				Payload: build,
				Text:    fmt.Sprintf("elefante %s", build.Version),
			}); err != nil {
				return model.WrapError(model.ErrorInternal, "Could not write the command result.", err)
			}

			return nil
		},
	}
}
