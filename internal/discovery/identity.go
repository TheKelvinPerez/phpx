package discovery

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"

	"github.com/elefantephp/elefante/internal/model"
)

func identityKey(identity model.ProjectIdentity, remoteIdentity string) string {
	repositoryIdentity := remoteIdentity
	if repositoryIdentity == "" {
		repositoryIdentity = identity.GitCommonDir
	}
	if repositoryIdentity == "" {
		repositoryIdentity = identity.ComposerRoot
	}

	composerRelativePath := "."
	if identity.WorkspaceRoot != "" {
		if relative, err := filepath.Rel(identity.WorkspaceRoot, identity.ComposerRoot); err == nil {
			composerRelativePath = filepath.ToSlash(relative)
		}
	}

	components := []string{
		repositoryIdentity,
		composerRelativePath,
		identity.WorkspaceRoot,
	}
	sum := sha256.Sum256([]byte(strings.Join(components, "\x00")))

	return "sha256:" + hex.EncodeToString(sum[:])
}
