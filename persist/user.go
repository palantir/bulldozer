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

type User struct {
	GithubID int    `db:"github_id"`
	Name     string `db:"name"`
	Token    string `db:"token"`
}

func (*User) InsertStmt() string {
	return "INSERT INTO USERS (github_id, name, token) VALUES (:github_id, :name, :token)"
}

func (*User) DeleteStmt() string {
	return "DELETE FROM USERS WHERE github_id = :github_id"
}

func (*User) UpdateStmt() string {
	return "UPDATE USERS SET token = :token WHERE github_id = :github_id"
}

func GetUserByName(db *sqlx.DB, name string) (*User, error) {
	u := &User{}

	q := fmt.Sprintf("SELECT * FROM USERS WHERE name='%s'", name)
	row := db.QueryRowx(q)
	err := row.StructScan(u)

	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil, nil
		}

		return nil, errors.Wrapf(err, "cannot get user %s", name)
	}

	return u, nil
}

func GetUserByID(db *sqlx.DB, id int) (*User, error) {
	u := &User{}

	q := fmt.Sprintf("SELECT * FROM USERS WHERE github_id='%d'", id)
	row := db.QueryRowx(q)
	err := row.StructScan(u)

	if err != nil {
		return nil, errors.Wrapf(err, "cannot get user %d", id)
	}

	return u, nil
}

func UpdateUserToken(db *sqlx.DB, id int, token string) error {
	q := fmt.Sprintf("UPDATE USERS SET token='%s' WHERE github_id='%d'", token, id)
	return db.QueryRowx(q).Err()
}
