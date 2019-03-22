// Copyright 2018 Palantir Technologies, Inc.
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

package pull

import (
	"context"
)

// Context is the context for a pull request. It defines methods to get
// information about the pull request. It is assumed that the implementation
// is not thread safe.
//
// A new Context should be created each time a Pull Request is being evaluated
// such that implementations are not required to consider cache invalidation.
type Context interface {
	// The Pull Request repository owner
	Owner() string

	// The Pull Request repository name
	Repo() string

	// The Pull Request Number
	Number() int

	// Locator returns a locator string for the pull request. The locator
	// string is formatted as "<owner>/<repository>#<number>"
	Locator() string

	// Title returns the pull request title
	Title(ctx context.Context) (string, error)

	// Body returns the pull request body
	Body(ctx context.Context) (string, error)

	// RequiredStatuses returns the names of the required status
	// checks for the pull request.
	RequiredStatuses(ctx context.Context) ([]string, error)

	// CurrentSuccessStatuses returns the names of all currently
	// successful status checks for the pull request.
	CurrentSuccessStatuses(ctx context.Context) ([]string, error)

	// Comments lists all comments on the pull request.
	Comments(ctx context.Context) ([]string, error)

	// Commits lists all commits on the pull request.
	Commits(ctx context.Context) ([]*Commit, error)

	// Labels lists all labels on the pull request.
	Labels(ctx context.Context) ([]string, error)

	// Branches returns the base (also known as target) and head branch names
	// of this pull request. Branches in this repository have no prefix, while
	// branches in forks are prefixed with the owner of the fork and a colon.
	// The base branch will always be unprefixed.
	Branches(ctx context.Context) (base string, head string, err error)
}

type Commit struct {
	SHA     string
	Message string
}
