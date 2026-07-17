package discovery

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/elefantephp/elefante/internal/model"
)

type gitMetadata struct {
	repositoryRoot string
	workspaceRoot  string
	commonDir      string
	branch         string
	headCommit     string
	remoteIdentity string
}

func inspectGit(ctx context.Context, directory string) (*gitMetadata, error) {
	if err := ctx.Err(); err != nil {
		return nil, wrapGitError("inspect the Git workspace", err)
	}

	insideWorktree, err := runGitProbe(ctx, directory, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		if gitIsUnavailable(err) {
			return nil, nil
		}

		return nil, wrapGitError("inspect the Git workspace", err)
	}
	if insideWorktree != "true" {
		return nil, nil
	}

	workspaceRoot, err := runGitRequired(
		ctx,
		directory,
		"read the Git workspace root",
		"rev-parse",
		"--show-toplevel",
	)
	if err != nil {
		return nil, err
	}
	workspaceRoot, err = filepath.EvalSymlinks(filepath.Clean(workspaceRoot))
	if err != nil {
		return nil, wrapGitError("resolve the Git workspace root", err)
	}

	commonDir, err := runGitRequired(
		ctx,
		directory,
		"read the Git common directory",
		"rev-parse",
		"--path-format=absolute",
		"--git-common-dir",
	)
	if err != nil {
		return nil, err
	}
	commonDir, err = filepath.EvalSymlinks(filepath.Clean(commonDir))
	if err != nil {
		return nil, wrapGitError("resolve the Git common directory", err)
	}

	branch, err := runGitOptional(ctx, directory, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return nil, wrapGitError("read the Git branch", err)
	}
	headCommit, err := runGitOptional(ctx, directory, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return nil, wrapGitError("read the Git HEAD commit", err)
	}
	remote, err := runGitOptional(ctx, directory, "config", "--get", "remote.origin.url")
	if err != nil {
		return nil, wrapGitError("read the Git origin URL", err)
	}

	repositoryRoot := workspaceRoot
	if filepath.Base(commonDir) == ".git" {
		repositoryRoot = filepath.Dir(commonDir)
	}

	return &gitMetadata{
		repositoryRoot: repositoryRoot,
		workspaceRoot:  workspaceRoot,
		commonDir:      commonDir,
		branch:         branch,
		headCommit:     headCommit,
		remoteIdentity: normalizeRemoteURL(remote),
	}, nil
}

func gitIsUnavailable(err error) bool {
	if errors.Is(err, exec.ErrNotFound) {
		return true
	}

	var exitError *exec.ExitError
	return errors.As(err, &exitError) &&
		(exitError.ExitCode() == 1 || exitError.ExitCode() == 128)
}

func runGitProbe(ctx context.Context, directory string, arguments ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", append([]string{"-C", directory}, arguments...)...)
	output, err := command.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func runGitRequired(
	ctx context.Context,
	directory string,
	action string,
	arguments ...string,
) (string, error) {
	output, err := runGitProbe(ctx, directory, arguments...)
	if err != nil {
		return "", wrapGitError(action, err)
	}
	if output == "" {
		return "", model.NewError(
			model.ErrorDiscovery,
			fmt.Sprintf("Could not %s.", action),
		)
	}

	return output, nil
}

func runGitOptional(ctx context.Context, directory string, arguments ...string) (string, error) {
	output, err := runGitProbe(ctx, directory, arguments...)
	if err == nil {
		return output, nil
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
		return "", nil
	}
	if arguments[0] == "rev-parse" &&
		errors.As(err, &exitError) &&
		exitError.ExitCode() == 128 {
		return "", nil
	}

	return "", err
}

func wrapGitError(action string, err error) *model.Error {
	return model.WrapError(
		model.ErrorDiscovery,
		fmt.Sprintf("Could not %s.", action),
		err,
	)
}

func normalizeRemoteURL(remote string) string {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return ""
	}

	if parsed, err := url.Parse(remote); err == nil && parsed.Scheme != "" {
		parsed.User = nil

		return parsed.String()
	}

	if at := strings.LastIndex(remote, "@"); at >= 0 {
		return remote[at+1:]
	}

	return remote
}
