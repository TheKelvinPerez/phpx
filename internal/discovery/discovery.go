package discovery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/elefantephp/elefante/internal/model"
	projectpaths "github.com/elefantephp/elefante/internal/paths"
)

type Request struct {
	StartPath       string
	MaxMetadataSize int64
}

func Discover(ctx context.Context, request Request) (model.ProjectFacts, error) {
	start, err := projectpaths.NormalizeStart(request.StartPath)
	if err != nil {
		return model.ProjectFacts{}, model.WrapError(
			model.ErrorDiscovery,
			"Could not resolve the project path.",
			err,
		)
	}

	git, err := inspectGit(ctx, start.Directory)
	if err != nil {
		return model.ProjectFacts{}, err
	}

	ancestorBoundary := filesystemRoot(start.Directory)
	candidateSearchRoot := start.Directory
	if git != nil {
		ancestorBoundary = git.workspaceRoot
		candidateSearchRoot = git.workspaceRoot
	}

	composerRoot, found, err := nearestComposerRoot(start.Directory, ancestorBoundary)
	if err != nil {
		return model.ProjectFacts{}, err
	}
	if !found {
		candidates, err := composerRootsUnder(candidateSearchRoot)
		if err != nil {
			return model.ProjectFacts{}, err
		}

		switch len(candidates) {
		case 0:
			return model.ProjectFacts{}, model.NewError(
				model.ErrorDiscovery,
				"Could not find composer.json from the project path.",
			)
		case 1:
			composerRoot = candidates[0]
		default:
			return model.ProjectFacts{}, ambiguousComposerRoots(candidates)
		}
	}

	identity := model.ProjectIdentity{
		ComposerRoot:    composerRoot,
		ApplicationRoot: composerRoot,
		WorkspaceRoot:   composerRoot,
	}
	remoteIdentity := ""

	if git != nil {
		identity.RepositoryRoot = git.repositoryRoot
		identity.WorkspaceRoot = git.workspaceRoot
		identity.GitCommonDir = git.commonDir
		identity.Branch = git.branch
		identity.HeadCommit = git.headCommit
		remoteIdentity = git.remoteIdentity
	}
	identity.IdentityKey = identityKey(identity, remoteIdentity)

	metadataBoundary := identity.ComposerRoot
	if git != nil {
		metadataBoundary = identity.WorkspaceRoot
	}
	composerFingerprint, err := readComposerMetadata(
		identity.ComposerRoot,
		metadataBoundary,
		request.MaxMetadataSize,
	)
	if err != nil {
		return model.ProjectFacts{}, err
	}

	return model.ProjectFacts{
		StartingPath: model.ProjectPath{
			Supplied: start.Supplied,
			Absolute: start.Absolute,
			Resolved: start.Resolved,
		},
		Identity:          identity,
		InputFingerprints: []model.InputFingerprint{composerFingerprint},
	}, nil
}

func nearestComposerRoot(startDirectory string, boundary string) (string, bool, error) {
	directory := startDirectory
	for {
		composerPath := filepath.Join(directory, "composer.json")
		info, err := os.Stat(composerPath)
		switch {
		case err == nil && !info.IsDir():
			return directory, true, nil
		case err == nil:
			return "", false, model.NewError(
				model.ErrorDiscovery,
				fmt.Sprintf("%s is not a file.", composerPath),
			)
		case !errors.Is(err, os.ErrNotExist):
			return "", false, model.WrapError(
				model.ErrorDiscovery,
				"Could not inspect Composer project metadata.",
				err,
			)
		}

		if directory == boundary {
			break
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}

	return "", false, nil
}

func composerRootsUnder(root string) ([]string, error) {
	var candidates []string

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && path != root && ignoredDiscoveryDirectory(entry.Name()) {
			return filepath.SkipDir
		}
		if entry.IsDir() || entry.Name() != "composer.json" {
			return nil
		}

		candidates = append(candidates, filepath.Dir(path))

		return nil
	})
	if err != nil {
		return nil, model.WrapError(
			model.ErrorDiscovery,
			"Could not search for Composer project roots.",
			err,
		)
	}

	sort.Strings(candidates)

	return candidates, nil
}

func ignoredDiscoveryDirectory(name string) bool {
	switch name {
	case ".git", ".elefante", "node_modules", "vendor":
		return true
	default:
		return false
	}
}

func ambiguousComposerRoots(candidates []string) *model.Error {
	commandError := model.NewError(
		model.ErrorDiscoveryAmbiguousRoots,
		"Several Composer projects were found. Select one with --project.",
	)
	commandError.Hint = "Pass --project with one of the candidate paths."

	for _, candidate := range candidates {
		commandError.Details = append(commandError.Details, model.ErrorDetail{
			Name:  "candidate",
			Value: candidate,
		})
		commandError.Sources = append(commandError.Sources, model.SourceReference{
			Path: filepath.Join(candidate, "composer.json"),
			Kind: "composer_manifest",
		})
	}

	return commandError
}

func filesystemRoot(path string) string {
	root := filepath.Clean(path)
	for {
		parent := filepath.Dir(root)
		if parent == root {
			return root
		}
		root = parent
	}
}
