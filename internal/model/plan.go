package model

const PlanSchemaVersion = "elefante.plan/v1"

type Operation string

const (
	OperationDoctor Operation = "doctor"
	OperationSync   Operation = "sync"
)

type PlanPolicy struct {
	Offline bool `json:"offline"`
	Frozen  bool `json:"frozen"`
}

type Capability string

const (
	CapabilityInspectRuntime    Capability = "inspect_runtime"
	CapabilityInspectExtensions Capability = "inspect_extensions"
	CapabilityInspectComposer   Capability = "inspect_composer"
	CapabilityInspectPlatform   Capability = "inspect_platform"
	CapabilityInstallRuntime    Capability = "install_runtime"
	CapabilityInstallExtension  Capability = "install_extension"
	CapabilityStartProvider     Capability = "start_provider"
	CapabilityExecuteCommand    Capability = "execute_command"
)

type RuntimeObservation struct {
	Name    string          `json:"name"`
	Version string          `json:"version"`
	SAPI    string          `json:"sapi,omitempty"`
	Source  SourceReference `json:"source"`
}

type ExtensionObservation struct {
	Name      string          `json:"name"`
	Version   string          `json:"version,omitempty"`
	Available bool            `json:"available"`
	Source    SourceReference `json:"source"`
}

type ComposerObservation struct {
	Version         string          `json:"version"`
	Source          string          `json:"source"`
	Path            string          `json:"path,omitempty"`
	Identity        string          `json:"identity"`
	SHA256          string          `json:"sha256,omitempty"`
	DistributionURL string          `json:"distribution_url,omitempty"`
	MetadataURL     string          `json:"metadata_url,omitempty"`
	Cached          bool            `json:"cached,omitempty"`
	PluginAPI       string          `json:"plugin_api,omitempty"`
	RuntimeAPI      string          `json:"runtime_api,omitempty"`
	Reference       SourceReference `json:"reference"`
}

type ProviderState string

const (
	ProviderStateUnavailable  ProviderState = "unavailable"
	ProviderStateUnconfigured ProviderState = "unconfigured"
	ProviderStateStopped      ProviderState = "stopped"
	ProviderStateRunning      ProviderState = "running"
	ProviderStateDegraded     ProviderState = "degraded"
)

type EngineObservation struct {
	Name     string          `json:"name"`
	Version  string          `json:"version"`
	Platform string          `json:"platform,omitempty"`
	Source   SourceReference `json:"source"`
}

type ProviderObservation struct {
	Provider     string                 `json:"provider"`
	Available    bool                   `json:"available"`
	Version      string                 `json:"version,omitempty"`
	Platform     string                 `json:"platform,omitempty"`
	Architecture string                 `json:"architecture,omitempty"`
	State        ProviderState          `json:"state,omitempty"`
	Engines      []EngineObservation    `json:"engines,omitempty"`
	Capabilities []Capability           `json:"capabilities,omitempty"`
	Runtimes     []RuntimeObservation   `json:"runtimes,omitempty"`
	Composer     []ComposerObservation  `json:"composer,omitempty"`
	Extensions   []ExtensionObservation `json:"extensions,omitempty"`
	Diagnostics  []Diagnostic           `json:"diagnostics,omitempty"`
	Fingerprint  string                 `json:"fingerprint"`
}

type ProviderSelection struct {
	Name                   string `json:"name,omitempty"`
	Reason                 string `json:"reason,omitempty"`
	ObservationFingerprint string `json:"observation_fingerprint,omitempty"`
}

type ResolutionStatus string

const (
	ResolutionSatisfied       ResolutionStatus = "satisfied"
	ResolutionActionRequired  ResolutionStatus = "action_required"
	ResolutionBlocked         ResolutionStatus = "blocked"
	ResolutionAmbiguous       ResolutionStatus = "ambiguous"
	ResolutionLegacy          ResolutionStatus = "legacy"
	ResolutionOptionalMissing ResolutionStatus = "optional_missing"
)

type RequirementResolution struct {
	Name          string            `json:"name"`
	Kind          RequirementKind   `json:"kind"`
	Constraint    string            `json:"constraint"`
	Optional      bool              `json:"optional"`
	Status        ResolutionStatus  `json:"status"`
	SelectedValue string            `json:"selected_value,omitempty"`
	Sources       []SourceReference `json:"sources"`
}

type EffectClass string

const (
	EffectRead                 EffectClass = "read"
	EffectCacheMutation        EffectClass = "cache_mutation"
	EffectLocalStateMutation   EffectClass = "local_state_mutation"
	EffectProviderMutation     EffectClass = "provider_mutation"
	EffectMachineMutation      EffectClass = "machine_mutation"
	EffectProjectMutation      EffectClass = "project_mutation"
	EffectProjectCodeExecution EffectClass = "project_code_execution"
)

type NetworkRequirement string

const (
	NetworkNone     NetworkRequirement = "none"
	NetworkRead     NetworkRequirement = "read"
	NetworkRequired NetworkRequirement = "required"
)

type TrustClass string

const (
	TrustNone            TrustClass = "none"
	TrustComposerPlugins TrustClass = "composer_plugins"
	TrustComposerScripts TrustClass = "composer_scripts"
)

type ActionKind string

const (
	ActionPrepareCache        ActionKind = "prepare_cache"
	ActionPrepareRuntime      ActionKind = "prepare_runtime"
	ActionPrepareExtension    ActionKind = "prepare_extension"
	ActionPrepareComposer     ActionKind = "prepare_composer"
	ActionPrepareProvider     ActionKind = "prepare_provider"
	ActionInstallDependencies ActionKind = "install_dependencies"
	ActionVerifyPlatform      ActionKind = "verify_platform"
	ActionRecordState         ActionKind = "record_state"
)

type ActionInput struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ActionOutput struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type PlanAction struct {
	ID              string             `json:"id"`
	Kind            ActionKind         `json:"kind"`
	Summary         string             `json:"summary"`
	Effect          EffectClass        `json:"effect"`
	Network         NetworkRequirement `json:"network"`
	Trust           TrustClass         `json:"trust"`
	Reversible      bool               `json:"reversible"`
	Inputs          []ActionInput      `json:"inputs,omitempty"`
	ExpectedOutputs []ActionOutput     `json:"expected_outputs,omitempty"`
	Dependencies    []string           `json:"dependencies,omitempty"`
}

type TrustRequirement struct {
	Class       TrustClass        `json:"class"`
	Fingerprint string            `json:"fingerprint"`
	Sources     []SourceReference `json:"sources,omitempty"`
}

type Plan struct {
	SchemaVersion string                  `json:"schema_version"`
	Operation     Operation               `json:"operation"`
	Project       ProjectIdentity         `json:"project"`
	Provider      ProviderSelection       `json:"provider"`
	Requirements  []RequirementResolution `json:"requirements"`
	Actions       []PlanAction            `json:"actions"`
	Diagnostics   []Diagnostic            `json:"diagnostics,omitempty"`
	Trust         []TrustRequirement      `json:"trust,omitempty"`
	Inputs        []InputFingerprint      `json:"inputs"`
	Policy        PlanPolicy              `json:"policy"`
	Digest        string                  `json:"digest"`
}
