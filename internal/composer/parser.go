package composer

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/elefantephp/elefante/internal/jsonmeta"
	"github.com/elefantephp/elefante/internal/model"
)

type Document struct {
	Path    string
	Content []byte
}

type ProjectInput struct {
	Manifest Document
	Lock     *Document
}

type ParseResult struct {
	Facts       model.ComposerFacts
	Diagnostics []model.Diagnostic
}

type manifestDocument struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Require    map[string]string `json:"require"`
	RequireDev map[string]string `json:"require-dev"`
	Conflict   map[string]string `json:"conflict"`
	Config     struct {
		Platform map[string]json.RawMessage `json:"platform"`
	} `json:"config"`
	Scripts map[string]json.RawMessage `json:"scripts"`
}

type lockDocument struct {
	ContentHash       string                     `json:"content-hash"`
	Packages          []lockPackageDocument      `json:"packages"`
	PackagesDev       []lockPackageDocument      `json:"packages-dev"`
	Platform          map[string]string          `json:"platform"`
	PlatformDev       map[string]string          `json:"platform-dev"`
	PlatformOverrides map[string]json.RawMessage `json:"platform-overrides"`
	PluginAPIVersion  string                     `json:"plugin-api-version"`
}

type lockPackageDocument struct {
	Name     string            `json:"name"`
	Version  string            `json:"version"`
	Type     string            `json:"type"`
	Require  map[string]string `json:"require"`
	Conflict map[string]string `json:"conflict"`
}

func ParseProject(input ProjectInput) (ParseResult, error) {
	manifestRoot, err := jsonmeta.DecodeObject(
		input.Manifest.Content,
		jsonmeta.DefaultMaxDepth,
	)
	if err != nil {
		return ParseResult{}, composerSchemaError(
			model.SourceReference{
				Path: input.Manifest.Path,
				Kind: "composer_manifest",
			},
			"Composer metadata does not match the expected schema.",
			err,
		)
	}
	if invalidField := invalidManifestField(input.Manifest.Path, manifestRoot); invalidField != nil {
		return ParseResult{}, composerSchemaError(
			*invalidField,
			"Composer metadata does not match the expected field types.",
			nil,
		)
	}

	var manifest manifestDocument
	if err := json.Unmarshal(input.Manifest.Content, &manifest); err != nil {
		commandError := model.WrapError(
			model.ErrorRequirements,
			"Composer metadata does not match the expected schema.",
			err,
		)
		commandError.Sources = []model.SourceReference{
			{
				Path: input.Manifest.Path,
				Kind: "composer_manifest",
			},
		}

		return ParseResult{}, commandError
	}

	requirements := composerLinks(
		input.Manifest.Path,
		"composer_manifest",
		"/require",
		manifest.Require,
	)
	developmentRequirements := composerLinks(
		input.Manifest.Path,
		"composer_manifest",
		"/require-dev",
		manifest.RequireDev,
	)
	conflicts := composerLinks(
		input.Manifest.Path,
		"composer_manifest",
		"/conflict",
		manifest.Conflict,
	)

	platformRequirements := collectPlatformRequirements(
		requirements,
		model.RequirementScopeRoot,
	)
	platformRequirements = append(
		platformRequirements,
		collectPlatformRequirements(
			developmentRequirements,
			model.RequirementScopeRootDevelopment,
		)...,
	)
	platformRequirements = append(
		platformRequirements,
		collectPlatformRequirements(conflicts, model.RequirementScopeRootConflict)...,
	)
	platformEmulation, err := parsePlatformOverrides(
		input.Manifest.Path,
		"composer_manifest",
		"/config/platform",
		manifest.Config.Platform,
	)
	if err != nil {
		return ParseResult{}, err
	}
	scripts, err := parseScripts(input.Manifest.Path, manifest.Scripts)
	if err != nil {
		return ParseResult{}, err
	}

	expectedContentHash, err := composerContentHash(input.Manifest.Content)
	if err != nil {
		return ParseResult{}, composerSchemaError(
			model.SourceReference{
				Path: input.Manifest.Path,
				Kind: "composer_manifest",
			},
			"Composer metadata does not match the expected schema.",
			err,
		)
	}

	lockFacts := model.ComposerLockFacts{
		Path:                filepath.Join(filepath.Dir(input.Manifest.Path), "composer.lock"),
		ExpectedContentHash: expectedContentHash,
	}
	var diagnostics []model.Diagnostic
	var lockedRequirements []model.Requirement
	var lockedEmulation []model.PlatformOverride
	var lockedPackages []model.ComposerPackage
	var plugins []model.ComposerPlugin
	if input.Lock == nil {
		lockFacts.Status = model.ComposerLockMissing
		diagnostics = append(diagnostics, model.Diagnostic{
			Code:     "ELEFANTE_COMPOSER_LOCK_MISSING",
			Severity: model.SeverityWarning,
			Message:  "No Composer lock file was found.",
			Hint:     "Run composer update when deterministic dependency identity is required.",
			Sources: []model.SourceReference{
				{
					Path: lockFacts.Path,
					Kind: "composer_lock",
				},
			},
		})
	} else {
		lockFacts.Path = input.Lock.Path
		lock, malformedDiagnostic := decodeLock(*input.Lock)
		if malformedDiagnostic != nil {
			lockFacts.Status = model.ComposerLockInvalid
			diagnostics = append(diagnostics, *malformedDiagnostic)
		} else {
			lockFacts.ContentHash = lock.ContentHash
			lockFacts.PluginAPIVersion = lock.PluginAPIVersion
			consistencySources := lockConsistencySources(input.Lock.Path, lock)
			if len(consistencySources) > 0 {
				lockFacts.Status = model.ComposerLockInconsistent
				diagnostics = append(diagnostics, model.Diagnostic{
					Code:     "ELEFANTE_COMPOSER_LOCK_INCONSISTENT",
					Severity: model.SeverityError,
					Message:  "The Composer lock file is internally inconsistent.",
					Hint:     "Regenerate composer.lock with composer update.",
					Sources:  consistencySources,
				})
			} else {
				if lock.ContentHash == expectedContentHash {
					lockFacts.Status = model.ComposerLockFresh
				} else {
					lockFacts.Status = model.ComposerLockStale
					diagnostics = append(diagnostics, model.Diagnostic{
						Code:     "ELEFANTE_COMPOSER_LOCK_STALE",
						Severity: model.SeverityWarning,
						Message:  "The Composer lock file is not current with composer.json.",
						Hint:     "Run composer update to refresh the lock file.",
						Sources: []model.SourceReference{
							{
								Path:  input.Lock.Path,
								Kind:  "composer_lock",
								Field: "/content-hash",
							},
							{
								Path: input.Manifest.Path,
								Kind: "composer_manifest",
							},
						},
					})
				}

				lockedLinks := composerLinks(
					input.Lock.Path,
					"composer_lock",
					"/platform",
					lock.Platform,
				)
				lockedDevelopmentLinks := composerLinks(
					input.Lock.Path,
					"composer_lock",
					"/platform-dev",
					lock.PlatformDev,
				)
				lockedRequirements = collectPlatformRequirements(
					lockedLinks,
					model.RequirementScopeLocked,
				)
				lockedRequirements = append(
					lockedRequirements,
					collectPlatformRequirements(
						lockedDevelopmentLinks,
						model.RequirementScopeLockedDevelopment,
					)...,
				)

				lockedEmulation, err = parsePlatformOverrides(
					input.Lock.Path,
					"composer_lock",
					"/platform-overrides",
					lock.PlatformOverrides,
				)
				if err != nil {
					return ParseResult{}, err
				}

				var packageRequirements []model.Requirement
				lockedPackages, plugins, packageRequirements = parseLockedPackages(
					input.Lock.Path,
					lock.Packages,
					lock.PackagesDev,
				)
				lockedRequirements = append(lockedRequirements, packageRequirements...)
			}
		}
	}

	platformRequirements = append(platformRequirements, lockedRequirements...)
	platformEmulation = append(platformEmulation, lockedEmulation...)

	return ParseResult{
		Facts: model.ComposerFacts{
			Manifest: model.ComposerManifestFacts{
				Path:                    input.Manifest.Path,
				Name:                    manifest.Name,
				Type:                    manifest.Type,
				Requirements:            requirements,
				DevelopmentRequirements: developmentRequirements,
				Conflicts:               conflicts,
			},
			Lock: model.ComposerLockFacts{
				Path:                lockFacts.Path,
				Status:              lockFacts.Status,
				ContentHash:         lockFacts.ContentHash,
				ExpectedContentHash: lockFacts.ExpectedContentHash,
				PluginAPIVersion:    lockFacts.PluginAPIVersion,
				Packages:            lockedPackages,
			},
			PlatformRequirements: platformRequirements,
			PlatformEmulation:    platformEmulation,
			Plugins:              plugins,
			Scripts:              scripts,
		},
		Diagnostics: diagnostics,
	}, nil
}

func decodeLock(document Document) (lockDocument, *model.Diagnostic) {
	root, err := jsonmeta.DecodeObject(document.Content, jsonmeta.DefaultMaxDepth)
	if err != nil {
		return lockDocument{}, malformedLockDiagnostic(model.SourceReference{
			Path: document.Path,
			Kind: "composer_lock",
		})
	}
	if invalidField := invalidLockField(document.Path, root); invalidField != nil {
		return lockDocument{}, malformedLockDiagnostic(*invalidField)
	}

	var lock lockDocument
	if err := json.Unmarshal(document.Content, &lock); err != nil {
		return lockDocument{}, malformedLockDiagnostic(model.SourceReference{
			Path: document.Path,
			Kind: "composer_lock",
		})
	}

	return lock, nil
}

func malformedLockDiagnostic(source model.SourceReference) *model.Diagnostic {
	return &model.Diagnostic{
		Code:     "ELEFANTE_COMPOSER_LOCK_MALFORMED",
		Severity: model.SeverityError,
		Message:  "The Composer lock file is malformed.",
		Hint:     "Regenerate composer.lock with composer update.",
		Sources:  []model.SourceReference{source},
	}
}

func invalidLockField(path string, root *jsonmeta.Value) *model.SourceReference {
	if contentHash, exists := root.Member("content-hash"); exists &&
		contentHash.Kind() != jsonmeta.KindString {
		return lockSource(path, "/content-hash")
	}

	packages, exists := root.Member("packages")
	if !exists || packages.Kind() != jsonmeta.KindArray {
		return lockSource(path, "/packages")
	}
	if invalid := invalidLockPackages(path, "/packages", packages); invalid != nil {
		return invalid
	}

	if developmentPackages, exists := root.Member("packages-dev"); exists {
		if developmentPackages.Kind() != jsonmeta.KindNull {
			if developmentPackages.Kind() != jsonmeta.KindArray {
				return lockSource(path, "/packages-dev")
			}
			if invalid := invalidLockPackages(
				path,
				"/packages-dev",
				developmentPackages,
			); invalid != nil {
				return invalid
			}
		}
	}

	for _, field := range []string{"platform", "platform-dev"} {
		if value, exists := root.Member(field); exists {
			if invalid := invalidLockStringObject(path, "/"+field, value); invalid != nil {
				return invalid
			}
		}
	}

	if overrides, exists := root.Member("platform-overrides"); exists {
		if overrides.Kind() != jsonmeta.KindObject {
			return lockSource(path, "/platform-overrides")
		}
		for _, member := range overrides.Members() {
			if member.Value.Kind() == jsonmeta.KindString {
				continue
			}
			if boolean, ok := member.Value.BooleanValue(); ok && !boolean {
				continue
			}

			return lockSource(
				path,
				"/platform-overrides/"+escapeJSONPointer(member.Name),
			)
		}
	}

	if pluginAPI, exists := root.Member("plugin-api-version"); exists &&
		pluginAPI.Kind() != jsonmeta.KindString {
		return lockSource(path, "/plugin-api-version")
	}

	return nil
}

func invalidLockPackages(
	path string,
	field string,
	packages *jsonmeta.Value,
) *model.SourceReference {
	for index, value := range packages.Elements() {
		packageField := fmt.Sprintf("%s/%d", field, index)
		if value.Kind() != jsonmeta.KindObject {
			return lockSource(path, packageField)
		}
		for _, required := range []string{"name", "version"} {
			member, exists := value.Member(required)
			if !exists || member.Kind() != jsonmeta.KindString {
				return lockSource(path, packageField+"/"+required)
			}
		}
		if packageType, exists := value.Member("type"); exists &&
			packageType.Kind() != jsonmeta.KindString {
			return lockSource(path, packageField+"/type")
		}
		for _, links := range []string{"require", "conflict"} {
			if member, exists := value.Member(links); exists {
				if invalid := invalidLockStringObject(
					path,
					packageField+"/"+links,
					member,
				); invalid != nil {
					return invalid
				}
			}
		}
	}

	return nil
}

func invalidLockStringObject(
	path string,
	field string,
	value *jsonmeta.Value,
) *model.SourceReference {
	if value.Kind() != jsonmeta.KindObject {
		return lockSource(path, field)
	}
	for _, member := range value.Members() {
		if member.Value.Kind() != jsonmeta.KindString {
			return lockSource(path, field+"/"+escapeJSONPointer(member.Name))
		}
	}

	return nil
}

func lockSource(path string, field string) *model.SourceReference {
	return &model.SourceReference{
		Path:  path,
		Kind:  "composer_lock",
		Field: field,
	}
}

func invalidManifestField(path string, root *jsonmeta.Value) *model.SourceReference {
	for _, field := range []string{"name", "type"} {
		if value, exists := root.Member(field); exists && value.Kind() != jsonmeta.KindString {
			return manifestSource(path, "/"+field)
		}
	}
	for _, field := range []string{"require", "require-dev", "conflict"} {
		if value, exists := root.Member(field); exists {
			if invalid := invalidStringObjectField(path, "/"+field, value); invalid != nil {
				return invalid
			}
		}
	}

	if config, exists := root.Member("config"); exists {
		if config.Kind() != jsonmeta.KindObject {
			return manifestSource(path, "/config")
		}
		if platform, exists := config.Member("platform"); exists {
			if platform.Kind() != jsonmeta.KindObject {
				return manifestSource(path, "/config/platform")
			}
			for _, member := range platform.Members() {
				if member.Value.Kind() == jsonmeta.KindString {
					continue
				}
				if boolean, ok := member.Value.BooleanValue(); ok && !boolean {
					continue
				}

				return manifestSource(
					path,
					"/config/platform/"+escapeJSONPointer(member.Name),
				)
			}
		}
	}

	if scripts, exists := root.Member("scripts"); exists {
		if scripts.Kind() != jsonmeta.KindObject {
			return manifestSource(path, "/scripts")
		}
		for _, member := range scripts.Members() {
			field := "/scripts/" + escapeJSONPointer(member.Name)
			switch member.Value.Kind() {
			case jsonmeta.KindString:
				continue
			case jsonmeta.KindArray:
				for index, entry := range member.Value.Elements() {
					if entry.Kind() != jsonmeta.KindString {
						return manifestSource(path, fmt.Sprintf("%s/%d", field, index))
					}
				}
			default:
				return manifestSource(path, field)
			}
		}
	}

	return nil
}

func invalidStringObjectField(
	path string,
	field string,
	value *jsonmeta.Value,
) *model.SourceReference {
	if value.Kind() != jsonmeta.KindObject {
		return manifestSource(path, field)
	}
	for _, member := range value.Members() {
		if member.Value.Kind() != jsonmeta.KindString {
			return manifestSource(path, field+"/"+escapeJSONPointer(member.Name))
		}
	}

	return nil
}

func manifestSource(path string, field string) *model.SourceReference {
	return &model.SourceReference{
		Path:  path,
		Kind:  "composer_manifest",
		Field: field,
	}
}

func parseLockedPackages(
	path string,
	packages []lockPackageDocument,
	developmentPackages []lockPackageDocument,
) ([]model.ComposerPackage, []model.ComposerPlugin, []model.Requirement) {
	var facts []model.ComposerPackage
	var plugins []model.ComposerPlugin
	var requirements []model.Requirement

	parse := func(packages []lockPackageDocument, development bool, field string) {
		for index, locked := range packages {
			packageName := strings.ToLower(locked.Name)
			packageField := fmt.Sprintf("/%s/%d", field, index)
			packageSource := model.SourceReference{
				Path:  path,
				Kind:  "composer_lock",
				Field: packageField,
			}
			packageRequirements := composerLinks(
				path,
				"composer_lock",
				packageField+"/require",
				locked.Require,
			)
			packageConflicts := composerLinks(
				path,
				"composer_lock",
				packageField+"/conflict",
				locked.Conflict,
			)
			facts = append(facts, model.ComposerPackage{
				Name:         packageName,
				Version:      locked.Version,
				Type:         locked.Type,
				Development:  development,
				Requirements: packageRequirements,
				Conflicts:    packageConflicts,
				Source:       packageSource,
			})

			requireScope := model.RequirementScopeLockedPackage
			conflictScope := model.RequirementScopeLockedPackageConflict
			if development {
				requireScope = model.RequirementScopeLockedDevelopmentPackage
				conflictScope = model.RequirementScopeLockedDevelopmentPackageConflict
			}
			packagePlatformRequirements := collectPlatformRequirements(
				packageRequirements,
				requireScope,
			)
			packagePlatformRequirements = append(
				packagePlatformRequirements,
				collectPlatformRequirements(packageConflicts, conflictScope)...,
			)
			for index := range packagePlatformRequirements {
				packagePlatformRequirements[index].Package = packageName
			}
			requirements = append(requirements, packagePlatformRequirements...)

			if locked.Type == "composer-plugin" {
				plugins = append(plugins, model.ComposerPlugin{
					Name:        packageName,
					Version:     locked.Version,
					Development: development,
					Source:      packageSource,
				})
			}
		}
	}

	parse(packages, false, "packages")
	parse(developmentPackages, true, "packages-dev")
	sort.Slice(plugins, func(left int, right int) bool {
		return plugins[left].Name < plugins[right].Name
	})

	return facts, plugins, requirements
}

func lockConsistencySources(path string, lock lockDocument) []model.SourceReference {
	contentHash, err := hex.DecodeString(lock.ContentHash)
	if err != nil || len(contentHash) != md5.Size {
		return []model.SourceReference{
			{
				Path:  path,
				Kind:  "composer_lock",
				Field: "/content-hash",
			},
		}
	}

	seen := make(map[string]model.SourceReference)
	packageSets := []struct {
		field    string
		packages []lockPackageDocument
	}{
		{field: "packages", packages: lock.Packages},
		{field: "packages-dev", packages: lock.PackagesDev},
	}
	for _, set := range packageSets {
		for index, locked := range set.packages {
			source := model.SourceReference{
				Path:  path,
				Kind:  "composer_lock",
				Field: fmt.Sprintf("/%s/%d", set.field, index),
			}
			if locked.Name == "" || locked.Version == "" {
				return []model.SourceReference{source}
			}

			name := strings.ToLower(locked.Name)
			if previous, exists := seen[name]; exists {
				return []model.SourceReference{previous, source}
			}
			seen[name] = source
		}
	}

	return nil
}

func parseScripts(
	path string,
	definitions map[string]json.RawMessage,
) ([]model.ComposerScript, error) {
	result := make([]model.ComposerScript, 0, len(definitions))
	for name, rawDefinition := range definitions {
		source := model.SourceReference{
			Path:  path,
			Kind:  "composer_manifest",
			Field: "/scripts/" + escapeJSONPointer(name),
		}
		rawDefinition = bytes.TrimSpace(rawDefinition)

		var commands []string
		if len(rawDefinition) > 0 && rawDefinition[0] == '"' {
			var command string
			if err := json.Unmarshal(rawDefinition, &command); err != nil {
				return nil, composerSchemaError(
					source,
					"Composer scripts must use a command string or an array of command strings.",
					err,
				)
			}
			commands = []string{command}
		} else if len(rawDefinition) > 0 && rawDefinition[0] == '[' {
			if err := json.Unmarshal(rawDefinition, &commands); err != nil {
				return nil, composerSchemaError(
					source,
					"Composer scripts must use a command string or an array of command strings.",
					err,
				)
			}
		} else {
			return nil, composerSchemaError(
				source,
				"Composer scripts must use a command string or an array of command strings.",
				nil,
			)
		}

		encoded, err := json.Marshal(commands)
		if err != nil {
			return nil, composerSchemaError(
				source,
				"Could not fingerprint a Composer script definition.",
				err,
			)
		}
		sum := sha256.Sum256(encoded)
		result = append(result, model.ComposerScript{
			Name:          name,
			CommandCount:  len(commands),
			ContentSHA256: "sha256:" + hex.EncodeToString(sum[:]),
			Source:        source,
		})
	}
	sort.Slice(result, func(left int, right int) bool {
		return result[left].Name < result[right].Name
	})

	return result, nil
}

func parsePlatformOverrides(
	path string,
	sourceKind string,
	field string,
	overrides map[string]json.RawMessage,
) ([]model.PlatformOverride, error) {
	result := make([]model.PlatformOverride, 0, len(overrides))
	for originalName, rawValue := range overrides {
		name := strings.ToLower(originalName)
		kind, platform := platformRequirementKind(name)
		source := model.SourceReference{
			Path:  path,
			Kind:  sourceKind,
			Field: field + "/" + escapeJSONPointer(originalName),
		}
		if !platform {
			return nil, composerSchemaError(
				source,
				"Composer platform configuration contains an unsupported package name.",
				nil,
			)
		}

		override := model.PlatformOverride{
			Name:   name,
			Kind:   kind,
			Source: source,
		}
		if err := json.Unmarshal(rawValue, &override.Version); err == nil {
			result = append(result, override)
			continue
		}

		var booleanValue bool
		if err := json.Unmarshal(rawValue, &booleanValue); err == nil && !booleanValue {
			override.Disabled = true
			result = append(result, override)
			continue
		}

		return nil, composerSchemaError(
			source,
			"Composer platform configuration must use a version string or false.",
			nil,
		)
	}
	sort.Slice(result, func(left int, right int) bool {
		return result[left].Name < result[right].Name
	})

	return result, nil
}

func composerSchemaError(
	source model.SourceReference,
	message string,
	cause error,
) *model.Error {
	var commandError *model.Error
	if cause == nil {
		commandError = model.NewError(model.ErrorRequirements, message)
	} else {
		commandError = model.WrapError(model.ErrorRequirements, message, cause)
	}
	commandError.Sources = []model.SourceReference{source}

	return commandError
}

func collectPlatformRequirements(
	links []model.ComposerLink,
	scope model.RequirementScope,
) []model.Requirement {
	result := make([]model.Requirement, 0, len(links))
	for _, link := range links {
		kind, platform := platformRequirementKind(link.Name)
		if !platform {
			continue
		}

		result = append(result, model.Requirement{
			Name:       link.Name,
			Kind:       kind,
			Constraint: link.Constraint,
			Scope:      scope,
			Sources:    []model.SourceReference{link.Source},
		})
	}

	return result
}

func composerLinks(
	path string,
	sourceKind string,
	field string,
	links map[string]string,
) []model.ComposerLink {
	result := make([]model.ComposerLink, 0, len(links))
	for originalName, constraint := range links {
		result = append(result, model.ComposerLink{
			Name:       strings.ToLower(originalName),
			Constraint: constraint,
			Source: model.SourceReference{
				Path:  path,
				Kind:  sourceKind,
				Field: field + "/" + escapeJSONPointer(originalName),
			},
		})
	}
	sort.Slice(result, func(left int, right int) bool {
		return result[left].Name < result[right].Name
	})

	return result
}

func platformRequirementKind(name string) (model.RequirementKind, bool) {
	switch {
	case name == "php":
		return model.RequirementPHP, true
	case strings.HasPrefix(name, "php-"):
		return model.RequirementPHPSubtype, true
	case strings.HasPrefix(name, "ext-"):
		return model.RequirementExtension, true
	case strings.HasPrefix(name, "lib-"):
		return model.RequirementSystemLibrary, true
	case name == "composer":
		return model.RequirementComposer, true
	case name == "composer-plugin-api":
		return model.RequirementComposerPluginAPI, true
	case name == "composer-runtime-api":
		return model.RequirementComposerRuntimeAPI, true
	default:
		return "", false
	}
}

func escapeJSONPointer(value string) string {
	value = strings.ReplaceAll(value, "~", "~0")

	return strings.ReplaceAll(value, "/", "~1")
}
