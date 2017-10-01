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
	"time"

	"github.com/jinzhu/gorm"
)

type Repository struct {
	gorm.Model
	GitHubID  int `gorm:"column:github_id"`
	Name      string
	EnabledBy User
	EnabledAt time.Time
	HookID    int
}

type User struct {
	gorm.Model
	GitHubID int `gorm:"column:github_id"`
	Name     string
	Token    string
}

// InitializeSchema initializes the schema for storing artifact data
func InitializeSchema(db *gorm.DB) {
	db.AutoMigrate(&Repository{}, &User{})
}
