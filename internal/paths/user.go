package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
)

type UserPathOptions struct {
	GOOS       string
	Home       string
	ConfigHome string
	CacheHome  string
	StateHome  string
}

type UserPaths struct {
	ConfigRoot string
	CacheRoot  string
	LogRoot    string
	ConfigFile string
	StateRoot  string
	LocksRoot  string
}

type ProjectPaths struct {
	Root        string
	Environment string
	Trust       string
	Journal     string
	Lock        string
}

func CurrentUserPaths() (UserPaths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return UserPaths{}, fmt.Errorf("resolve user home directory: %w", err)
	}

	return ResolveUserPaths(UserPathOptions{
		GOOS:       runtime.GOOS,
		Home:       home,
		ConfigHome: os.Getenv("XDG_CONFIG_HOME"),
		CacheHome:  os.Getenv("XDG_CACHE_HOME"),
		StateHome:  os.Getenv("XDG_STATE_HOME"),
	})
}

func ResolveUserPaths(options UserPathOptions) (UserPaths, error) {
	home := filepath.Clean(strings.TrimSpace(options.Home))
	if home == "" || home == "." || !filepath.IsAbs(home) {
		return UserPaths{}, fmt.Errorf("user home directory must be absolute")
	}

	var configRoot string
	var cacheRoot string
	var logRoot string
	switch options.GOOS {
	case "darwin":
		configRoot = filepath.Join(home, "Library", "Application Support", "Elefante")
		cacheRoot = filepath.Join(home, "Library", "Caches", "Elefante")
		logRoot = filepath.Join(home, "Library", "Logs", "Elefante")
	case "linux":
		configHome := defaultPath(options.ConfigHome, filepath.Join(home, ".config"))
		cacheHome := defaultPath(options.CacheHome, filepath.Join(home, ".cache"))
		stateHome := defaultPath(
			options.StateHome,
			filepath.Join(home, ".local", "state"),
		)
		configRoot = filepath.Join(configHome, "elefante")
		cacheRoot = filepath.Join(cacheHome, "elefante")
		logRoot = filepath.Join(stateHome, "elefante", "logs")
	default:
		configRoot = filepath.Join(home, ".elefante")
		cacheRoot = filepath.Join(configRoot, "cache")
		logRoot = filepath.Join(configRoot, "logs")
	}

	stateRoot := filepath.Join(configRoot, "state")

	return UserPaths{
		ConfigRoot: configRoot,
		CacheRoot:  cacheRoot,
		LogRoot:    logRoot,
		ConfigFile: filepath.Join(configRoot, "config.toml"),
		StateRoot:  stateRoot,
		LocksRoot:  filepath.Join(configRoot, "locks"),
	}, nil
}

func (paths UserPaths) Project(identity string) (ProjectPaths, error) {
	safeIdentity, err := safeIdentity(identity)
	if err != nil {
		return ProjectPaths{}, err
	}
	root := filepath.Join(paths.StateRoot, "projects", safeIdentity)

	return ProjectPaths{
		Root:        root,
		Environment: filepath.Join(root, "environment.json"),
		Trust:       filepath.Join(root, "trust.json"),
		Journal:     filepath.Join(root, "journal.json"),
		Lock:        filepath.Join(paths.LocksRoot, safeIdentity+".lock"),
	}, nil
}

func defaultPath(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}

	return filepath.Clean(value)
}

func safeIdentity(identity string) (string, error) {
	identity = strings.TrimSpace(identity)
	if identity == "" {
		return "", fmt.Errorf("project identity cannot be empty")
	}

	var builder strings.Builder
	for _, character := range identity {
		if unicode.IsLetter(character) ||
			unicode.IsDigit(character) ||
			character == '.' ||
			character == '_' ||
			character == '-' {
			builder.WriteRune(character)
			continue
		}
		builder.WriteByte('_')
	}
	safe := strings.Trim(builder.String(), ".")
	if safe == "" || safe == "." || safe == ".." {
		return "", fmt.Errorf("project identity does not contain a safe path component")
	}

	return safe, nil
}
