package cli

import (
	"fmt"
	"strings"

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
	root.PersistentFlags().String("project", "", "Start project discovery from this path")

	root.AddCommand(newVersionCommand(dependencies.Application, dependencies.Renderer))
	root.AddCommand(newDoctorCommand(dependencies.Application, dependencies.Renderer))

	return root
}

func newDoctorCommand(application *app.Application, renderer output.Renderer) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Inspect the current project",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			projectPath, err := command.Flags().GetString("project")
			if err != nil {
				return model.WrapError(
					model.ErrorInternal,
					"Could not read the project path flag.",
					err,
				)
			}

			facts, err := application.Doctor(command.Context(), projectPath)
			if err != nil {
				return err
			}
			if err := renderer.Fact(output.Fact{
				Payload: facts,
				Text:    formatDoctorFacts(facts),
			}); err != nil {
				return model.WrapError(
					model.ErrorInternal,
					"Could not write the project discovery facts.",
					err,
				)
			}

			return nil
		},
	}
}

func formatDoctorFacts(facts model.ProjectFacts) string {
	lines := []string{
		fmt.Sprintf("Composer root: %s", facts.Identity.ComposerRoot),
		fmt.Sprintf("Workspace root: %s", facts.Identity.WorkspaceRoot),
	}
	if facts.Identity.RepositoryRoot != "" {
		lines = append(
			lines,
			fmt.Sprintf("Repository root: %s", facts.Identity.RepositoryRoot),
		)
	}
	lines = append(lines, fmt.Sprintf("Identity: %s", facts.Identity.IdentityKey))

	return strings.Join(lines, "\n")
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
