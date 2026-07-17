package discovery

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/elefantephp/elefante/internal/model"
	projectpaths "github.com/elefantephp/elefante/internal/paths"
)

const (
	defaultMaxMetadataSize int64 = 1024 * 1024
	maxJSONNesting               = 128
	maxInt64                     = int64(^uint64(0) >> 1)
)

type duplicateObjectKeyError struct {
	key string
}

func (err *duplicateObjectKeyError) Error() string {
	return fmt.Sprintf("duplicate object key %q", err.key)
}

type unexpectedRootTypeError struct{}

func (*unexpectedRootTypeError) Error() string {
	return "root value is not an object"
}

type jsonNestingLimitError struct{}

func (*jsonNestingLimitError) Error() string {
	return fmt.Sprintf("JSON exceeds the %d level nesting limit", maxJSONNesting)
}

func readComposerMetadata(
	composerRoot string,
	boundary string,
	maxSize int64,
) (model.InputFingerprint, error) {
	if maxSize <= 0 {
		maxSize = defaultMaxMetadataSize
	}

	path := filepath.Join(composerRoot, "composer.json")
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return model.InputFingerprint{}, composerMetadataError(
			path,
			"Could not resolve Composer metadata.",
			err,
		)
	}
	if !projectpaths.Contains(boundary, resolvedPath) {
		return model.InputFingerprint{}, composerMetadataError(
			path,
			"Composer metadata resolves outside the project boundary.",
			nil,
		)
	}

	file, err := os.Open(resolvedPath)
	if err != nil {
		return model.InputFingerprint{}, composerMetadataError(
			path,
			"Could not open Composer metadata.",
			err,
		)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return model.InputFingerprint{}, composerMetadataError(
			path,
			"Could not inspect Composer metadata.",
			err,
		)
	}
	if !info.Mode().IsRegular() {
		return model.InputFingerprint{}, composerMetadataError(
			path,
			"Composer metadata must be a regular file.",
			nil,
		)
	}
	if info.Size() > maxSize {
		return model.InputFingerprint{}, composerMetadataSizeError(path, maxSize)
	}

	readLimit := maxSize
	if readLimit < maxInt64 {
		readLimit++
	}
	content, err := io.ReadAll(io.LimitReader(file, readLimit))
	if err != nil {
		return model.InputFingerprint{}, composerMetadataError(
			path,
			"Could not read Composer metadata.",
			err,
		)
	}
	if int64(len(content)) > maxSize {
		return model.InputFingerprint{}, composerMetadataSizeError(path, maxSize)
	}

	if err := validateJSONObject(content); err != nil {
		var duplicateKey *duplicateObjectKeyError
		var unexpectedType *unexpectedRootTypeError
		var nestingLimit *jsonNestingLimitError

		switch {
		case errors.As(err, &duplicateKey):
			return model.InputFingerprint{}, composerMetadataError(
				path,
				fmt.Sprintf(
					"Composer metadata contains duplicate object key %q.",
					duplicateKey.key,
				),
				err,
			)
		case errors.As(err, &unexpectedType):
			return model.InputFingerprint{}, composerMetadataError(
				path,
				"Composer metadata must be a JSON object.",
				err,
			)
		case errors.As(err, &nestingLimit):
			return model.InputFingerprint{}, composerMetadataError(
				path,
				fmt.Sprintf(
					"Composer metadata exceeds the %d level JSON nesting limit.",
					maxJSONNesting,
				),
				err,
			)
		default:
			return model.InputFingerprint{}, composerMetadataError(
				path,
				"Composer metadata must be a valid JSON object.",
				err,
			)
		}
	}

	sum := sha256.Sum256(content)

	return model.InputFingerprint{
		Path:   path,
		Kind:   "composer_manifest",
		SHA256: hex.EncodeToString(sum[:]),
		Size:   int64(len(content)),
	}, nil
}

func validateJSONObject(content []byte) error {
	if !utf8.Valid(content) {
		return errors.New("JSON is not valid UTF 8")
	}

	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()

	kind, err := consumeJSONValue(decoder, 0)
	if err != nil {
		return err
	}
	if kind != jsonObject {
		return &unexpectedRootTypeError{}
	}

	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("JSON contains more than one root value")
		}

		return err
	}

	return nil
}

type jsonValueKind uint8

const (
	jsonScalar jsonValueKind = iota
	jsonObject
	jsonArray
)

func consumeJSONValue(decoder *json.Decoder, depth int) (jsonValueKind, error) {
	token, err := decoder.Token()
	if err != nil {
		return jsonScalar, err
	}

	delimiter, ok := token.(json.Delim)
	if !ok {
		return jsonScalar, nil
	}

	switch delimiter {
	case '{':
		if depth >= maxJSONNesting {
			return jsonObject, &jsonNestingLimitError{}
		}
		if err := consumeJSONObject(decoder, depth+1); err != nil {
			return jsonObject, err
		}

		return jsonObject, nil
	case '[':
		if depth >= maxJSONNesting {
			return jsonArray, &jsonNestingLimitError{}
		}
		if err := consumeJSONArray(decoder, depth+1); err != nil {
			return jsonArray, err
		}

		return jsonArray, nil
	default:
		return jsonScalar, fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
}

func consumeJSONObject(decoder *json.Decoder, depth int) error {
	keys := make(map[string]struct{})
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		key, ok := token.(string)
		if !ok {
			return errors.New("JSON object key is not a string")
		}
		if _, exists := keys[key]; exists {
			return &duplicateObjectKeyError{key: key}
		}
		keys[key] = struct{}{}

		if _, err := consumeJSONValue(decoder, depth); err != nil {
			return err
		}
	}

	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token != json.Delim('}') {
		return errors.New("JSON object is not closed")
	}

	return nil
}

func consumeJSONArray(decoder *json.Decoder, depth int) error {
	for decoder.More() {
		if _, err := consumeJSONValue(decoder, depth); err != nil {
			return err
		}
	}

	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token != json.Delim(']') {
		return errors.New("JSON array is not closed")
	}

	return nil
}

func composerMetadataSizeError(path string, maxSize int64) *model.Error {
	return composerMetadataError(
		path,
		fmt.Sprintf(
			"Composer metadata exceeds the configured %d byte limit.",
			maxSize,
		),
		nil,
	)
}

func composerMetadataError(path string, message string, cause error) *model.Error {
	var commandError *model.Error
	if cause == nil {
		commandError = model.NewError(model.ErrorDiscovery, message)
	} else {
		commandError = model.WrapError(model.ErrorDiscovery, message, cause)
	}
	commandError.Sources = []model.SourceReference{
		{
			Path: path,
			Kind: "composer_manifest",
		},
	}

	return commandError
}
