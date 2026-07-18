package model

type ProjectIdentity struct {
	RepositoryRoot  string `json:"repository_root,omitempty"`
	ComposerRoot    string `json:"composer_root"`
	ApplicationRoot string `json:"application_root"`
	WorkspaceRoot   string `json:"workspace_root"`
	GitCommonDir    string `json:"git_common_dir,omitempty"`
	Branch          string `json:"branch,omitempty"`
	HeadCommit      string `json:"head_commit,omitempty"`
	IdentityKey     string `json:"identity_key,omitempty"`
}

type InputFingerprint struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type ProjectPath struct {
	Supplied string `json:"supplied"`
	Absolute string `json:"absolute"`
	Resolved string `json:"resolved"`
}

type ComposerLink struct {
	Name       string          `json:"name"`
	Constraint string          `json:"constraint"`
	Source     SourceReference `json:"source"`
}

type ComposerManifestFacts struct {
	Path                    string         `json:"path"`
	Name                    string         `json:"name,omitempty"`
	Type                    string         `json:"type,omitempty"`
	Requirements            []ComposerLink `json:"requirements,omitempty"`
	DevelopmentRequirements []ComposerLink `json:"development_requirements,omitempty"`
	Conflicts               []ComposerLink `json:"conflicts,omitempty"`
}

type PlatformOverride struct {
	Name     string          `json:"name"`
	Kind     RequirementKind `json:"kind"`
	Version  string          `json:"version,omitempty"`
	Disabled bool            `json:"disabled"`
	Source   SourceReference `json:"source"`
}

type ComposerScript struct {
	Name          string          `json:"name"`
	CommandCount  int             `json:"command_count"`
	ContentSHA256 string          `json:"content_sha256"`
	Source        SourceReference `json:"source"`
}

type ComposerPackage struct {
	Name         string          `json:"name"`
	Version      string          `json:"version"`
	Type         string          `json:"type,omitempty"`
	Development  bool            `json:"development"`
	Requirements []ComposerLink  `json:"requirements,omitempty"`
	Conflicts    []ComposerLink  `json:"conflicts,omitempty"`
	Source       SourceReference `json:"source"`
}

type ComposerPlugin struct {
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Development bool            `json:"development"`
	Source      SourceReference `json:"source"`
}

type ComposerLockStatus string

const (
	ComposerLockMissing      ComposerLockStatus = "missing"
	ComposerLockFresh        ComposerLockStatus = "fresh"
	ComposerLockStale        ComposerLockStatus = "stale"
	ComposerLockInvalid      ComposerLockStatus = "invalid"
	ComposerLockInconsistent ComposerLockStatus = "inconsistent"
)

type ComposerLockFacts struct {
	Path                string             `json:"path"`
	Status              ComposerLockStatus `json:"status"`
	ContentHash         string             `json:"content_hash,omitempty"`
	ExpectedContentHash string             `json:"expected_content_hash,omitempty"`
	PluginAPIVersion    string             `json:"plugin_api_version,omitempty"`
	Packages            []ComposerPackage  `json:"packages,omitempty"`
}

type ComposerFacts struct {
	Manifest             ComposerManifestFacts `json:"manifest"`
	Lock                 ComposerLockFacts     `json:"lock"`
	PlatformRequirements []Requirement         `json:"platform_requirements,omitempty"`
	PlatformEmulation    []PlatformOverride    `json:"platform_emulation,omitempty"`
	Plugins              []ComposerPlugin      `json:"plugins,omitempty"`
	Scripts              []ComposerScript      `json:"scripts,omitempty"`
}

type ProjectFacts struct {
	StartingPath      ProjectPath        `json:"starting_path"`
	Identity          ProjectIdentity    `json:"identity"`
	Composer          ComposerFacts      `json:"composer"`
	Diagnostics       []Diagnostic       `json:"diagnostics,omitempty"`
	InputFingerprints []InputFingerprint `json:"input_fingerprints,omitempty"`
}
