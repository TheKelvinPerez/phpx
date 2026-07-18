package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/paths"
	"github.com/elefantephp/elefante/internal/security"
)

const maximumStateSize = 8 << 20

type Store struct {
	paths  paths.UserPaths
	writer AtomicWriter
}

func NewStore(userPaths paths.UserPaths, redactor security.Redactor) Store {
	return Store{
		paths: userPaths,
		writer: AtomicWriter{
			Redactor: redactor,
		},
	}
}

func (store Store) SaveJournal(
	identity string,
	journal ActionJournal,
) error {
	if err := validateRecord(
		identity,
		journal.ProjectIdentity,
		journal.SchemaVersion,
		ActionJournalSchemaVersion,
	); err != nil {
		return err
	}
	projectPaths, err := store.projectPaths(identity)
	if err != nil {
		return err
	}
	if err := store.writer.WriteJSON(projectPaths.Journal, journal); err != nil {
		return stateError("Could not persist the action journal.", projectPaths.Journal, err)
	}

	return nil
}

func (store Store) LoadJournal(
	identity string,
) (ActionJournal, bool, error) {
	projectPaths, err := store.projectPaths(identity)
	if err != nil {
		return ActionJournal{}, false, err
	}
	var journal ActionJournal
	exists, err := loadRecord(
		projectPaths.Journal,
		ActionJournalSchemaVersion,
		&journal,
	)
	if err != nil {
		return ActionJournal{}, false, err
	}
	if exists && journal.ProjectIdentity != identity {
		return ActionJournal{}, false, stateError(
			"Action journal identity does not match its state path.",
			projectPaths.Journal,
			nil,
		)
	}

	return journal, exists, nil
}

func (store Store) SaveTrust(identity string, trust TrustRecord) error {
	if err := validateRecord(
		identity,
		trust.ProjectIdentity,
		trust.SchemaVersion,
		TrustSchemaVersion,
	); err != nil {
		return err
	}
	projectPaths, err := store.projectPaths(identity)
	if err != nil {
		return err
	}
	if err := store.writer.WriteJSON(projectPaths.Trust, trust); err != nil {
		return stateError("Could not persist trust approvals.", projectPaths.Trust, err)
	}

	return nil
}

func (store Store) LoadTrust(
	identity string,
) (TrustRecord, bool, error) {
	projectPaths, err := store.projectPaths(identity)
	if err != nil {
		return TrustRecord{}, false, err
	}
	var trust TrustRecord
	exists, err := loadRecord(projectPaths.Trust, TrustSchemaVersion, &trust)
	if err != nil {
		return TrustRecord{}, false, err
	}
	if exists && trust.ProjectIdentity != identity {
		return TrustRecord{}, false, stateError(
			"Trust record identity does not match its state path.",
			projectPaths.Trust,
			nil,
		)
	}

	return trust, exists, nil
}

func (store Store) SaveEnvironment(
	identity string,
	environment EnvironmentRecord,
) error {
	if err := validateRecord(
		identity,
		environment.ProjectIdentity,
		environment.SchemaVersion,
		EnvironmentSchemaVersion,
	); err != nil {
		return err
	}
	projectPaths, err := store.projectPaths(identity)
	if err != nil {
		return err
	}
	if err := store.writer.WriteJSON(projectPaths.Environment, environment); err != nil {
		return stateError(
			"Could not persist the environment record.",
			projectPaths.Environment,
			err,
		)
	}

	return nil
}

func (store Store) LoadEnvironment(
	identity string,
) (EnvironmentRecord, bool, error) {
	projectPaths, err := store.projectPaths(identity)
	if err != nil {
		return EnvironmentRecord{}, false, err
	}
	var environment EnvironmentRecord
	exists, err := loadRecord(
		projectPaths.Environment,
		EnvironmentSchemaVersion,
		&environment,
	)
	if err != nil {
		return EnvironmentRecord{}, false, err
	}
	if exists && environment.ProjectIdentity != identity {
		return EnvironmentRecord{}, false, stateError(
			"Environment record identity does not match its state path.",
			projectPaths.Environment,
			nil,
		)
	}

	return environment, exists, nil
}

func (store Store) projectPaths(identity string) (paths.ProjectPaths, error) {
	projectPaths, err := store.paths.Project(identity)
	if err != nil {
		return paths.ProjectPaths{}, stateError(
			"Could not resolve the project state path.",
			"",
			err,
		)
	}

	return projectPaths, nil
}

func loadRecord(path string, expectedSchema string, target any) (bool, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, stateError("Could not read local state.", path, err)
	}
	if len(content) > maximumStateSize {
		return false, stateError(
			"Local state exceeds the supported size.",
			path,
			nil,
		)
	}

	var header struct {
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(content, &header); err != nil {
		return false, stateError("Local state is not valid JSON.", path, err)
	}
	if header.SchemaVersion != expectedSchema {
		return false, stateError(
			fmt.Sprintf(
				"Local state schema %q is not supported, expected %q.",
				header.SchemaVersion,
				expectedSchema,
			),
			path,
			nil,
		)
	}
	if err := json.Unmarshal(content, target); err != nil {
		return false, stateError("Could not decode local state.", path, err)
	}

	return true, nil
}

func validateRecord(
	expectedIdentity string,
	actualIdentity string,
	actualSchema string,
	expectedSchema string,
) error {
	if expectedIdentity != actualIdentity {
		return stateError(
			"State record identity does not match the requested project.",
			"",
			nil,
		)
	}
	if actualSchema != expectedSchema {
		return stateError(
			fmt.Sprintf(
				"State record schema %q is not supported, expected %q.",
				actualSchema,
				expectedSchema,
			),
			"",
			nil,
		)
	}

	return nil
}

func stateError(message string, path string, cause error) *model.Error {
	var commandError *model.Error
	if cause == nil {
		commandError = model.NewError(model.ErrorState, message)
	} else {
		commandError = model.WrapError(model.ErrorState, message, cause)
	}
	if path != "" {
		commandError.Sources = []model.SourceReference{
			{Path: path, Kind: "local_state"},
		}
	}

	return commandError
}
