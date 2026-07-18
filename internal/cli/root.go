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
	root.PersistentFlags().String("config", "", "Use this Elefante configuration file")
	root.PersistentFlags().String("provider", "", "Select an environment provider")
	root.PersistentFlags().Bool("offline", false, "Prohibit network access")
	root.PersistentFlags().Bool("frozen", false, "Prohibit project file changes")

	root.AddCommand(newVersionCommand(dependencies.Application, dependencies.Renderer))
	root.AddCommand(newDoctorCommand(dependencies.Application, dependencies.Renderer))
	root.AddCommand(newPlanCommand(dependencies.Application, dependencies.Renderer))

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
			configPath, err := command.Flags().GetString("config")
			if err != nil {
				return model.WrapError(
					model.ErrorInternal,
					"Could not read the configuration path flag.",
					err,
				)
			}

			provider, err := command.Flags().GetString("provider")
			if err != nil {
				return model.WrapError(
					model.ErrorInternal,
					"Could not read the provider flag.",
					err,
				)
			}

			analysis, err := application.Doctor(command.Context(), app.DoctorRequest{
				ProjectPath: projectPath,
				ConfigPath:  configPath,
				Provider:    provider,
			})
			if err != nil {
				return err
			}
			if err := renderAnalysis(renderer, analysis); err != nil {
				return err
			}

			return blockingPlanError(analysis.Plan)
		},
	}
}

func newPlanCommand(
	application *app.Application,
	renderer output.Renderer,
) *cobra.Command {
	return &cobra.Command{
		Use:   "plan",
		Short: "Build the read only synchronization plan",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			projectPath, configPath, provider, err := analysisPaths(command)
			if err != nil {
				return err
			}
			offline, err := command.Flags().GetBool("offline")
			if err != nil {
				return model.WrapError(
					model.ErrorInternal,
					"Could not read the offline flag.",
					err,
				)
			}
			frozen, err := command.Flags().GetBool("frozen")
			if err != nil {
				return model.WrapError(
					model.ErrorInternal,
					"Could not read the frozen flag.",
					err,
				)
			}

			analysis, err := application.Plan(command.Context(), app.PlanRequest{
				ProjectPath: projectPath,
				ConfigPath:  configPath,
				Provider:    provider,
				Offline:     offline,
				Frozen:      frozen,
			})
			if err != nil {
				return err
			}
			if err := renderAnalysis(renderer, analysis); err != nil {
				return err
			}

			return blockingPlanError(analysis.Plan)
		},
	}
}

func analysisPaths(command *cobra.Command) (string, string, string, error) {
	projectPath, err := command.Flags().GetString("project")
	if err != nil {
		return "", "", "", model.WrapError(
			model.ErrorInternal,
			"Could not read the project path flag.",
			err,
		)
	}
	configPath, err := command.Flags().GetString("config")
	if err != nil {
		return "", "", "", model.WrapError(
			model.ErrorInternal,
			"Could not read the configuration path flag.",
			err,
		)
	}
	provider, err := command.Flags().GetString("provider")
	if err != nil {
		return "", "", "", model.WrapError(
			model.ErrorInternal,
			"Could not read the provider flag.",
			err,
		)
	}

	return projectPath, configPath, provider, nil
}

func renderAnalysis(
	renderer output.Renderer,
	analysis app.Analysis,
) error {
	if err := renderer.Fact(output.Fact{
		Payload: analysis.Facts,
		Text:    formatProjectFacts(analysis),
	}); err != nil {
		return model.WrapError(
			model.ErrorInternal,
			"Could not write the project discovery facts.",
			err,
		)
	}
	for _, observation := range analysis.Observations {
		if err := renderer.Fact(output.Fact{
			Payload: observation,
			Text:    formatProviderObservation(observation),
		}); err != nil {
			return model.WrapError(
				model.ErrorInternal,
				"Could not write a provider observation.",
				err,
			)
		}
	}
	for _, diagnostic := range analysis.Plan.Diagnostics {
		if err := renderer.Diagnostic(output.Diagnostic{
			Payload: diagnostic,
			Text:    formatDiagnostic(diagnostic),
		}); err != nil {
			return model.WrapError(
				model.ErrorInternal,
				"Could not write a plan diagnostic.",
				err,
			)
		}
	}
	if err := renderer.Plan(output.Plan{
		Payload: analysis.Plan,
		Text:    formatPlan(analysis.Plan),
	}); err != nil {
		return model.WrapError(
			model.ErrorInternal,
			"Could not write the resolved plan.",
			err,
		)
	}

	return nil
}

func formatProjectFacts(analysis app.Analysis) string {
	facts := analysis.Facts
	provider := analysis.Plan.Provider.Name
	if provider == "" {
		provider = "none"
	}
	reason := humanToken(analysis.Plan.Provider.Reason)
	if reason == "" {
		reason = "no deterministic selection"
	}
	lines := []string{
		fmt.Sprintf("Project: %s", facts.Identity.ComposerRoot),
		fmt.Sprintf("Provider: %s", provider),
		fmt.Sprintf("Selection reason: %s", reason),
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

func formatProviderObservation(observation model.ProviderObservation) string {
	lines := []string{
		fmt.Sprintf("Provider observation: %s", observation.Provider),
		fmt.Sprintf(
			"Platform: %s/%s",
			observation.Platform,
			observation.Architecture,
		),
	}
	for _, runtime := range observation.Runtimes {
		lines = append(
			lines,
			fmt.Sprintf(
				"PHP: %s, SAPI %s, %s",
				runtime.Version,
				runtime.SAPI,
				runtime.Source.Path,
			),
		)
	}
	for _, composer := range observation.Composer {
		lines = append(
			lines,
			fmt.Sprintf(
				"Composer: %s, %s",
				composer.Version,
				composer.Path,
			),
		)
	}
	capabilities := make([]string, 0, len(observation.Capabilities))
	for _, capability := range observation.Capabilities {
		capabilities = append(capabilities, humanToken(string(capability)))
	}
	lines = append(
		lines,
		fmt.Sprintf("Capabilities: %s", strings.Join(capabilities, ", ")),
		fmt.Sprintf("Observation fingerprint: %s", observation.Fingerprint),
	)

	return strings.Join(lines, "\n")
}

func formatDiagnostic(diagnostic model.Diagnostic) string {
	label := humanToken(string(diagnostic.Severity))
	if label == "" {
		label = "diagnostic"
	}
	label = strings.ToUpper(label[:1]) + label[1:]
	lines := []string{
		fmt.Sprintf("%s [%s]: %s", label, diagnostic.Code, diagnostic.Message),
	}
	if diagnostic.Detail != "" {
		lines = append(lines, "Detail: "+diagnostic.Detail)
	}
	if diagnostic.Hint != "" {
		lines = append(lines, "Hint: "+diagnostic.Hint)
	}

	return strings.Join(lines, "\n")
}

func formatPlan(builtPlan model.Plan) string {
	lines := []string{
		fmt.Sprintf("Operation: %s", builtPlan.Operation),
		"Requirements:",
	}
	if len(builtPlan.Requirements) == 0 {
		lines = append(lines, "None")
	}
	for index, requirement := range builtPlan.Requirements {
		resolution := humanToken(string(requirement.Status))
		if requirement.SelectedValue != "" {
			resolution += ", selected " + requirement.SelectedValue
		}
		lines = append(
			lines,
			fmt.Sprintf(
				"%d. %s %s, %s",
				index+1,
				requirement.Name,
				requirement.Constraint,
				resolution,
			),
		)
	}
	lines = append(lines, "Planned actions:")
	if len(builtPlan.Actions) == 0 {
		lines = append(lines, "None")
	}
	for index, action := range builtPlan.Actions {
		lines = append(
			lines,
			fmt.Sprintf(
				"%d. %s, %s",
				index+1,
				action.Summary,
				humanToken(string(action.Effect)),
			),
		)
	}
	lines = append(lines, "Plan digest: "+builtPlan.Digest)
	if builtPlan.Operation == model.OperationDoctor {
		lines = append(lines, "Next command: elefante plan")
	} else {
		lines = append(lines, "Next command: elefante sync")
	}

	return strings.Join(lines, "\n")
}

func blockingPlanError(builtPlan model.Plan) error {
	for _, diagnostic := range builtPlan.Diagnostics {
		if diagnostic.Severity != model.SeverityError {
			continue
		}
		code := model.ErrorRequirements
		if strings.HasPrefix(diagnostic.Code, "ELEFANTE_PROVIDER") ||
			strings.HasPrefix(diagnostic.Code, "ELEFANTE_NATIVE") {
			code = model.ErrorProvider
		}
		commandError := model.NewError(
			code,
			"Project analysis found a blocking diagnostic.",
		)
		commandError.Detail = diagnostic.Message
		commandError.Hint = diagnostic.Hint
		commandError.Sources = append(
			[]model.SourceReference(nil),
			diagnostic.Sources...,
		)
		commandError.Provider = diagnostic.Provider

		return commandError
	}

	return nil
}

func humanToken(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), "_", " ")
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
