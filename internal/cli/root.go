package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/elefantephp/elefante/internal/app"
	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/output"
	"github.com/spf13/cobra"
)

type Dependencies struct {
	Application  *app.Application
	Renderer     output.Renderer
	AllowPrompts bool
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
	root.PersistentFlags().Bool("non-interactive", false, "Prohibit prompts")
	root.PersistentFlags().Bool("yes", false, "Approve the freshly computed plan")
	root.PersistentFlags().String(
		"approve-plan",
		"",
		"Approve only an exact reviewed plan digest",
	)
	root.MarkFlagsMutuallyExclusive("yes", "approve-plan")

	root.AddCommand(newVersionCommand(dependencies.Application, dependencies.Renderer))
	root.AddCommand(newDoctorCommand(dependencies.Application, dependencies.Renderer))
	root.AddCommand(newPlanCommand(dependencies.Application, dependencies.Renderer))
	root.AddCommand(newSyncCommand(
		dependencies.Application,
		dependencies.Renderer,
		dependencies.AllowPrompts,
	))

	return root
}

func newSyncCommand(
	application *app.Application,
	renderer output.Renderer,
	allowPrompts bool,
) *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Apply an explicitly approved synchronization plan",
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
			nonInteractive, err := command.Flags().GetBool("non-interactive")
			if err != nil {
				return model.WrapError(
					model.ErrorInternal,
					"Could not read the noninteractive flag.",
					err,
				)
			}
			yes, err := command.Flags().GetBool("yes")
			if err != nil {
				return model.WrapError(
					model.ErrorInternal,
					"Could not read the approval flag.",
					err,
				)
			}
			approvedPlan, err := command.Flags().GetString("approve-plan")
			if err != nil {
				return model.WrapError(
					model.ErrorInternal,
					"Could not read the approved plan digest.",
					err,
				)
			}

			request := app.SyncRequest{
				ProjectPath:    projectPath,
				ConfigPath:     configPath,
				Provider:       provider,
				Offline:        offline,
				Frozen:         frozen,
				NonInteractive: nonInteractive,
				Yes:            yes,
				ApprovedPlan:   approvedPlan,
			}
			analysis, syncErr := application.Sync(
				command.Context(),
				request,
			)
			if analysis.Plan.SchemaVersion != "" {
				if err := renderAnalysis(renderer, analysis); err != nil {
					return err
				}
			}
			if !isCommandError(syncErr, model.ErrorApprovalRequired) {
				return syncErr
			}
			if err := renderApprovalRequired(renderer, analysis.Plan); err != nil {
				return err
			}
			if !allowPrompts || nonInteractive {
				return syncErr
			}

			confirmed, err := promptForApproval(
				command.InOrStdin(),
				command.ErrOrStderr(),
			)
			if err != nil {
				return model.WrapError(
					model.ErrorApprovalRequired,
					"Could not read synchronization approval.",
					err,
				)
			}
			if !confirmed {
				return syncErr
			}

			request.ApprovedPlan = analysis.Plan.Digest
			request.Yes = false
			revalidated, syncErr := application.Sync(command.Context(), request)
			if revalidated.Plan.Digest != "" &&
				revalidated.Plan.Digest != analysis.Plan.Digest {
				if err := renderAnalysis(renderer, revalidated); err != nil {
					return err
				}
			}

			return syncErr
		},
	}
}

func renderApprovalRequired(
	renderer output.Renderer,
	builtPlan model.Plan,
) error {
	effectSet := make(map[model.EffectClass]struct{})
	for _, action := range builtPlan.Actions {
		if action.Effect != model.EffectRead {
			effectSet[action.Effect] = struct{}{}
		}
	}
	effects := make([]model.EffectClass, 0, len(effectSet))
	for effect := range effectSet {
		effects = append(effects, effect)
	}
	sort.Slice(effects, func(left int, right int) bool {
		return effects[left] < effects[right]
	})

	if err := renderer.ApprovalRequired(output.ApprovalRequired{
		Payload: model.ApprovalRequiredPayload{
			PlanDigest: builtPlan.Digest,
			Effects:    effects,
			Trust: append(
				[]model.TrustRequirement(nil),
				builtPlan.Trust...,
			),
		},
		Text: "Approval required for plan " + builtPlan.Digest + ".",
	}); err != nil {
		return model.WrapError(
			model.ErrorInternal,
			"Could not write the approval requirement.",
			err,
		)
	}

	return nil
}

func promptForApproval(input io.Reader, writer io.Writer) (bool, error) {
	if _, err := fmt.Fprint(writer, "Apply this exact plan? [y/N]: "); err != nil {
		return false, err
	}
	line, err := bufio.NewReader(input).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	answer := strings.ToLower(strings.TrimSpace(line))

	return answer == "y" || answer == "yes", nil
}

func isCommandError(err error, code model.ErrorCode) bool {
	var commandError *model.Error

	return errors.As(err, &commandError) && commandError.Code == code
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

			return app.BlockingPlanError(analysis.Plan)
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

			return app.BlockingPlanError(analysis.Plan)
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
		fmt.Sprintf("Available: %s", yesNo(observation.Available)),
	}
	if observation.Version != "" {
		lines = append(lines, fmt.Sprintf("Version: %s", observation.Version))
	}
	if observation.State != "" {
		lines = append(lines, fmt.Sprintf("State: %s", observation.State))
	}
	if observation.Platform != "" || observation.Architecture != "" {
		lines = append(
			lines,
			fmt.Sprintf(
				"Platform: %s/%s",
				observation.Platform,
				observation.Architecture,
			),
		)
	}
	for _, engine := range observation.Engines {
		description := strings.TrimSpace(engine.Name + " " + engine.Version)
		if engine.Platform != "" {
			description += ", " + engine.Platform
		}
		lines = append(lines, "Engine: "+description)
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

func yesNo(value bool) string {
	if value {
		return "yes"
	}

	return "no"
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
