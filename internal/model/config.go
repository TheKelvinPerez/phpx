package model

type ConfigProjectPolicy struct {
	ComposerRoot string `json:"composer_root,omitempty"`
}

type ConfigProviderPolicy struct {
	Preferred []string `json:"preferred,omitempty"`
	Allowed   []string `json:"allowed,omitempty"`
	Denied    []string `json:"denied,omitempty"`
}

type ConfigComposerPolicy struct {
	Constraint string `json:"constraint,omitempty"`
}

type ConfigExtensionPolicy struct {
	Optional []string `json:"optional,omitempty"`
}

type ConfigTask struct {
	Name             string   `json:"name"`
	Command          []string `json:"command"`
	WorkingDirectory string   `json:"working_directory"`
}

type ConfigCIPolicy struct {
	Provider string `json:"provider,omitempty"`
	Frozen   bool   `json:"frozen"`
}

type ConfigFacts struct {
	Path          string                `json:"path,omitempty"`
	SchemaVersion int                   `json:"schema_version,omitempty"`
	Project       ConfigProjectPolicy   `json:"project"`
	Providers     ConfigProviderPolicy  `json:"providers"`
	Composer      ConfigComposerPolicy  `json:"composer"`
	Extensions    ConfigExtensionPolicy `json:"extensions"`
	Tasks         []ConfigTask          `json:"tasks,omitempty"`
	CI            ConfigCIPolicy        `json:"ci"`
}
