// Copyright 2017 Palantir Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package persist

import (
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var metaSchema = `
CREATE TABLE IF NOT EXISTS schema (
    version INTEGER PRIMARY KEY CHECK (version > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS schema_one_row ON schema((TRUE));`

var currSchemaVersion = 1

var schema = `
CREATE TABLE IF NOT EXISTS USERS (
  github_id INTEGER PRIMARY KEY UNIQUE,
  name TEXT,
  token TEXT
);

CREATE TABLE IF NOT EXISTS REPOS (
  id INTEGER PRIMARY KEY UNIQUE,
  name TEXT,
  enabled_by TEXT,
  enabled_at BIGINT,
  hook_id INTEGER
);
`

// Persistable are structs that can be persisted to a DB and
// are compatible with associated utility methods in this package
type Persistable interface {
	InsertStmt() string
	DeleteStmt() string
	UpdateStmt() string
}

// Put persists a Persistable to the given DB
func Put(db *sqlx.DB, p Persistable) error {
	_, err := db.NamedExec(p.InsertStmt(), p)
	if err != nil {
		return errors.Wrapf(err, "failed persisting %v", p)
	}
	return nil
}

// Delete deletes a Persistable from the given DB
func Delete(db *sqlx.DB, p Persistable) error {
	_, err := db.NamedExec(p.DeleteStmt(), p)
	if err != nil {
		return errors.Wrapf(err, "failed deleting %v", p)
	}
	return nil
}

func Update(db *sqlx.DB, p Persistable) error {
	_, err := db.NamedExec(p.UpdateStmt(), p)
	if err != nil {
		return errors.Wrapf(err, "failed updating %v", p)
	}
	return nil
}

// InitializeSchema initializes the schema for storing artifact data
func InitializeSchema(db *sqlx.DB) error {
	version, err := getSchemaVersion(db)
	if err != nil {
		return errors.Wrapf(err, "failed to determine database schema version")
	}
	if version != currSchemaVersion {
		err := migrateSchema(db, currSchemaVersion)
		if err != nil {
			return errors.Wrapf(err, "failed migrating database schema from version %d to %d", version, currSchemaVersion)
		}
	}
	_, err = db.Exec(schema)
	if err != nil {
		return errors.Wrapf(err, "failed initializing database schema")
	}
	return nil
}

func getSchemaVersion(db *sqlx.DB) (int, error) {
	_, err := db.Exec(metaSchema)
	if err != nil {
		return 0, errors.Wrapf(err, "failed initializing schema table")
	}
	var version []int
	err = db.Select(&version, "SELECT version FROM schema")
	if err != nil {
		return 0, errors.Wrapf(err, "failed querying schema table")
	}
	if len(version) == 0 {
		_, err := db.Exec("INSERT INTO schema (version) VALUES ($1)", currSchemaVersion)
		if err != nil {
			return 0, errors.Wrapf(err, "failed setting schema version in database")
		}
		version = append(version, currSchemaVersion)
	}
	return version[0], nil
}

func migrateSchema(db *sqlx.DB, schemaVersion int) error {
	return errors.New("SCHEMA MIGRATION NOT IMPLEMENTED AT THIS TIME :(")
}
