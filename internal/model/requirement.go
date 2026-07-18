package model

type RequirementKind string

const (
	RequirementPHP                RequirementKind = "php"
	RequirementPHPSubtype         RequirementKind = "php_subtype"
	RequirementExtension          RequirementKind = "extension"
	RequirementSystemLibrary      RequirementKind = "system_library"
	RequirementComposer           RequirementKind = "composer"
	RequirementComposerPluginAPI  RequirementKind = "composer_plugin_api"
	RequirementComposerRuntimeAPI RequirementKind = "composer_runtime_api"
)

type RequirementScope string

const (
	RequirementScopeRoot                             RequirementScope = "root"
	RequirementScopeRootDevelopment                  RequirementScope = "root_development"
	RequirementScopeRootConflict                     RequirementScope = "root_conflict"
	RequirementScopeLocked                           RequirementScope = "locked"
	RequirementScopeLockedDevelopment                RequirementScope = "locked_development"
	RequirementScopeLockedPackage                    RequirementScope = "locked_package"
	RequirementScopeLockedPackageConflict            RequirementScope = "locked_package_conflict"
	RequirementScopeLockedDevelopmentPackage         RequirementScope = "locked_development_package"
	RequirementScopeLockedDevelopmentPackageConflict RequirementScope = "locked_development_package_conflict"
)

type Requirement struct {
	Name       string            `json:"name"`
	Kind       RequirementKind   `json:"kind"`
	Constraint string            `json:"constraint"`
	Scope      RequirementScope  `json:"scope"`
	Package    string            `json:"package,omitempty"`
	Optional   bool              `json:"optional"`
	Sources    []SourceReference `json:"sources"`
}
