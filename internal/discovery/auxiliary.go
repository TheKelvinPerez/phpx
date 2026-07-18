package discovery

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/elefantephp/elefante/internal/model"
)

type auxiliaryFacts struct {
	versionFiles    []model.VersionFileFact
	providerMarkers []model.ProviderMarkerFact
	fingerprints    []model.InputFingerprint
}

type providerMarkerDefinition struct {
	path     string
	provider string
}

var providerMarkerDefinitions = []providerMarkerDefinition{
	{path: filepath.Join(".ddev", "config.yaml"), provider: "ddev"},
	{path: "compose.yaml", provider: "docker_compose"},
	{path: "compose.yml", provider: "docker_compose"},
	{path: "docker-compose.yaml", provider: "docker_compose"},
	{path: "docker-compose.yml", provider: "docker_compose"},
	{path: "herd.yml", provider: "herd"},
}

func discoverAuxiliaryFacts(
	composerRoot string,
	boundary string,
	maxSize int64,
) (auxiliaryFacts, error) {
	var facts auxiliaryFacts

	versionPath := filepath.Join(composerRoot, ".php-version")
	versionMetadata, err := readOptionalMetadataFile(
		versionPath,
		boundary,
		maxSize,
		"php_version",
		"PHP version file",
	)
	if err != nil {
		return auxiliaryFacts{}, err
	}
	if versionMetadata != nil {
		if !utf8.Valid(versionMetadata.content) {
			return auxiliaryFacts{}, metadataError(
				versionPath,
				"php_version",
				"PHP version file must contain valid UTF-8 text.",
				nil,
			)
		}
		version := strings.TrimSpace(string(versionMetadata.content))
		if version == "" || len(strings.Fields(version)) != 1 {
			return auxiliaryFacts{}, metadataError(
				versionPath,
				"php_version",
				"PHP version file must contain exactly one version value.",
				nil,
			)
		}
		facts.versionFiles = append(facts.versionFiles, model.VersionFileFact{
			Runtime: "php",
			Version: version,
			Source: model.SourceReference{
				Path: versionPath,
				Kind: "php_version",
				Line: 1,
			},
		})
		facts.fingerprints = append(
			facts.fingerprints,
			versionMetadata.fingerprint,
		)
	}

	for _, definition := range providerMarkerDefinitions {
		path := filepath.Join(composerRoot, definition.path)
		metadata, err := readOptionalMetadataFile(
			path,
			boundary,
			maxSize,
			"provider_config",
			"provider configuration",
		)
		if err != nil {
			return auxiliaryFacts{}, err
		}
		if metadata == nil {
			continue
		}
		facts.providerMarkers = append(
			facts.providerMarkers,
			model.ProviderMarkerFact{
				Provider: definition.provider,
				Source: model.SourceReference{
					Path: path,
					Kind: "provider_config",
				},
			},
		)
		facts.fingerprints = append(
			facts.fingerprints,
			metadata.fingerprint,
		)
	}

	return facts, nil
}

func readOptionalMetadataFile(
	path string,
	boundary string,
	maxSize int64,
	sourceKind string,
	label string,
) (*metadataFile, error) {
	_, err := os.Lstat(path)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		return nil, nil
	default:
		return nil, metadataError(
			path,
			sourceKind,
			"Could not inspect "+label+".",
			err,
		)
	}

	metadata, err := readMetadataFile(
		path,
		boundary,
		maxSize,
		sourceKind,
		label,
	)
	if err != nil {
		return nil, err
	}

	return &metadata, nil
}
