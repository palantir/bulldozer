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
	// Owner returns the pull request repository owner.
	Owner() string

	// Repo returns the pull request repository name.
	Repo() string

	// Number returns the pull request number.
	Number() int

	// Locator returns a locator string for the pull request. The locator
	// string is formatted as "<owner>/<repository>#<number>"
	Locator() string

	// Title returns the pull request title.
	Title() string

	// Body returns the pull request body.
	Body() string

	// HeadSHA returns the SHA hash of the latest commit in the pull request.
	HeadSHA() string

	// Branches returns the base (also known as target) and head branch names
	// of this pull request. Branches in this repository have no prefix, while
	// branches in forks are prefixed with the owner of the fork and a colon.
	// The base branch will always be unprefixed.
	Branches() (base string, head string)

	// MergeState returns the current mergability of the pull request. It
	// always returns the most up-to-date state possible.
	MergeState(ctx context.Context) (*MergeState, error)

	// RequiredStatuses returns the names of the required status
	// checks for the pull request.
	RequiredStatuses(ctx context.Context) ([]string, error)

	// PushRestrictions returns true if the target barnch of the pull request
	// restricts the users or teams that have push access.
	PushRestrictions(ctx context.Context) (bool, error)

	// CurrentSuccessStatuses returns the names of all currently
	// successful status checks for the pull request.
	CurrentSuccessStatuses(ctx context.Context) (map[string]string, error)

	// Comments lists all comments on the pull request.
	Comments(ctx context.Context) ([]string, error)

	// Commits lists all commits on the pull request.
	Commits(ctx context.Context) ([]*Commit, error)

	// Labels lists all labels on the pull request.
	Labels(ctx context.Context) ([]string, error)

	// IsTargeted returns true if the head branch of this pull request is the
	// target branch of other open PRs on the repository.
	IsTargeted(ctx context.Context) (bool, error)

	// IsDraft returns true if the PR is in a draft state.
	IsDraft(ctx context.Context) bool

	// AutoMerge returns true if the PR is configured to be automatically merged.
	AutoMerge(ctx context.Context) bool
}

type MergeState struct {
	Closed    bool
	Mergeable *bool
}

type Commit struct {
	SHA     string
	Message string
}
