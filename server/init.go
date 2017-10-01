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

package server

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" // Import for side-effects
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/palantir/bulldozer/persist"
	"github.com/palantir/bulldozer/server/config"
)

func InitDB(dbc *config.DatabaseConfig) (*gorm.DB, error) {
	connectStr := fmt.Sprintf("host=%s dbname=%s user=%s sslmode=%s", dbc.Host, dbc.DBName, dbc.Username, dbc.SSLMode)
	log.WithFields(log.Fields{
		"connectionString": connectStr,
	}).Info("Attempting to connect to DB")

	if dbc.Password != "" {
		connectStr += fmt.Sprintf(" password=%s", dbc.Password)
	}

	db, err := gorm.Open("postgres", connectStr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed connecting to postgres")
	}

	persist.InitializeSchema(db)
	return db, nil
}
