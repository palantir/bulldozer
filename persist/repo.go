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
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type Repository struct {
	ID        int    `db:"id"`
	Name      string `db:"name"`
	EnabledBy string `db:"enabled_by"`
	EnabledAt int64  `db:"enabled_at"`
	HookID    int    `db:"hook_id"`
}

func (*Repository) InsertStmt() string {
	return "INSERT INTO REPOS (id, name, enabled_by, enabled_at, hook_id) VALUES (:id, :name, :enabled_by, :enabled_at, :hook_id)"
}

func (*Repository) DeleteStmt() string {
	return "DELETE FROM REPOS WHERE id = :id"
}

func (*Repository) UpdateStmt() string {
	return "TODO"
}

func GetRepositoryByID(db *sqlx.DB, repoID int) (*Repository, error) {
	r := &Repository{}

	q := fmt.Sprintf("SELECT * FROM REPOS WHERE id=%d", repoID)
	row := db.QueryRowx(q)
	err := row.StructScan(r)

	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil, nil
		}

		return nil, errors.Wrapf(err, "cannot get repo %d", repoID)
	}

	return r, nil
}
