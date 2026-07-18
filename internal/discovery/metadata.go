package discovery

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/elefantephp/elefante/internal/jsonmeta"
	"github.com/elefantephp/elefante/internal/model"
	projectpaths "github.com/elefantephp/elefante/internal/paths"
)

const (
	defaultMaxMetadataSize int64 = 1024 * 1024
	maxJSONNesting               = 128
	maxInt64                     = int64(^uint64(0) >> 1)
)

type metadataFile struct {
	content     []byte
	fingerprint model.InputFingerprint
}

func readComposerMetadata(
	composerRoot string,
	boundary string,
	maxSize int64,
) (metadataFile, error) {
	path := filepath.Join(composerRoot, "composer.json")
	metadata, err := readMetadataFile(
		path,
		boundary,
		maxSize,
		"composer_manifest",
		"Composer metadata",
	)
	if err != nil {
		return metadataFile{}, err
	}

	if err := validateJSONObject(metadata.content); err != nil {
		var duplicateKey *jsonmeta.DuplicateKeyError
		var unexpectedType *jsonmeta.RootTypeError
		var nestingLimit *jsonmeta.DepthError

		switch {
		case errors.As(err, &duplicateKey):
			return metadataFile{}, metadataError(
				path,
				"composer_manifest",
				fmt.Sprintf(
					"Composer metadata contains duplicate object key %q.",
					duplicateKey.Key,
				),
				err,
			)
		case errors.As(err, &unexpectedType):
			return metadataFile{}, metadataError(
				path,
				"composer_manifest",
				"Composer metadata must be a JSON object.",
				err,
			)
		case errors.As(err, &nestingLimit):
			return metadataFile{}, metadataError(
				path,
				"composer_manifest",
				fmt.Sprintf(
					"Composer metadata exceeds the %d level JSON nesting limit.",
					maxJSONNesting,
				),
				err,
			)
		default:
			return metadataFile{}, metadataError(
				path,
				"composer_manifest",
				"Composer metadata must be a valid JSON object.",
				err,
			)
		}
	}

	return metadata, nil
}

func readComposerLock(
	composerRoot string,
	boundary string,
	maxSize int64,
) (*metadataFile, error) {
	path := filepath.Join(composerRoot, "composer.lock")
	if _, err := os.Lstat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, metadataError(
			path,
			"composer_lock",
			"Could not inspect Composer lock metadata.",
			err,
		)
	}

	metadata, err := readMetadataFile(
		path,
		boundary,
		maxSize,
		"composer_lock",
		"Composer lock metadata",
	)
	if err != nil {
		return nil, err
	}

	return &metadata, nil
}

func readMetadataFile(
	path string,
	boundary string,
	maxSize int64,
	sourceKind string,
	label string,
) (metadataFile, error) {
	if maxSize <= 0 {
		maxSize = defaultMaxMetadataSize
	}

	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return metadataFile{}, metadataError(
			path,
			sourceKind,
			fmt.Sprintf("Could not resolve %s.", label),
			err,
		)
	}
	if !projectpaths.Contains(boundary, resolvedPath) {
		return metadataFile{}, metadataError(
			path,
			sourceKind,
			fmt.Sprintf("%s resolves outside the project boundary.", label),
			nil,
		)
	}

	file, err := os.Open(resolvedPath)
	if err != nil {
		return metadataFile{}, metadataError(
			path,
			sourceKind,
			fmt.Sprintf("Could not open %s.", label),
			err,
		)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return metadataFile{}, metadataError(
			path,
			sourceKind,
			fmt.Sprintf("Could not inspect %s.", label),
			err,
		)
	}
	if !info.Mode().IsRegular() {
		return metadataFile{}, metadataError(
			path,
			sourceKind,
			fmt.Sprintf("%s must be a regular file.", label),
			nil,
		)
	}
	if info.Size() > maxSize {
		return metadataFile{}, metadataSizeError(
			path,
			sourceKind,
			label,
			maxSize,
		)
	}

	readLimit := maxSize
	if readLimit < maxInt64 {
		readLimit++
	}
	content, err := io.ReadAll(io.LimitReader(file, readLimit))
	if err != nil {
		return metadataFile{}, metadataError(
			path,
			sourceKind,
			fmt.Sprintf("Could not read %s.", label),
			err,
		)
	}
	if int64(len(content)) > maxSize {
		return metadataFile{}, metadataSizeError(
			path,
			sourceKind,
			label,
			maxSize,
		)
	}

	sum := sha256.Sum256(content)

	return metadataFile{
		content: content,
		fingerprint: model.InputFingerprint{
			Path:   path,
			Kind:   sourceKind,
			SHA256: hex.EncodeToString(sum[:]),
			Size:   int64(len(content)),
		},
	}, nil
}

func validateJSONObject(content []byte) error {
	_, err := jsonmeta.DecodeObject(content, maxJSONNesting)

	return err
}

func metadataSizeError(
	path string,
	sourceKind string,
	label string,
	maxSize int64,
) *model.Error {
	return metadataError(
		path,
		sourceKind,
		fmt.Sprintf(
			"%s exceeds the configured %d byte limit.",
			label,
			maxSize,
		),
		nil,
	)
}

func metadataError(
	path string,
	sourceKind string,
	message string,
	cause error,
) *model.Error {
	var commandError *model.Error
	if cause == nil {
		commandError = model.NewError(model.ErrorDiscovery, message)
	} else {
		commandError = model.WrapError(model.ErrorDiscovery, message, cause)
	}
	commandError.Sources = []model.SourceReference{
		{
			Path: path,
			Kind: sourceKind,
		},
	}

	return commandError
}
