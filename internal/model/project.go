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

type ProjectFacts struct {
	StartingPath      ProjectPath        `json:"starting_path"`
	Identity          ProjectIdentity    `json:"identity"`
	InputFingerprints []InputFingerprint `json:"input_fingerprints,omitempty"`
}
