package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

type StartPath struct {
	Supplied  string
	Absolute  string
	Resolved  string
	Directory string
}

func NormalizeStart(path string) (StartPath, error) {
	supplied := path
	if path == "" {
		workingDirectory, err := os.Getwd()
		if err != nil {
			return StartPath{}, fmt.Errorf("get working directory: %w", err)
		}
		path = workingDirectory
		supplied = workingDirectory
	}

	absolute, err := filepath.Abs(path)
	if err != nil {
		return StartPath{}, fmt.Errorf("make start path absolute: %w", err)
	}
	absolute = filepath.Clean(absolute)

	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return StartPath{}, fmt.Errorf("resolve start path: %w", err)
	}
	resolved = filepath.Clean(resolved)

	info, err := os.Stat(resolved)
	if err != nil {
		return StartPath{}, fmt.Errorf("inspect start path: %w", err)
	}

	directory := resolved
	if !info.IsDir() {
		directory = filepath.Dir(resolved)
	}

	return StartPath{
		Supplied:  supplied,
		Absolute:  absolute,
		Resolved:  resolved,
		Directory: directory,
	}, nil
}

func Contains(root string, path string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	if err != nil {
		return false
	}

	return relative == "." ||
		(relative != ".." && !filepath.IsAbs(relative) &&
			!startsWithParent(relative))
}

func startsWithParent(relative string) bool {
	return len(relative) > 3 &&
		relative[:3] == ".."+string(filepath.Separator)
}
