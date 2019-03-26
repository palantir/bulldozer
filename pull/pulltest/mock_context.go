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

package pulltest

import (
	"context"

	"github.com/palantir/bulldozer/pull"
)

// MockPullContext is a dummy Context implementation.
type MockPullContext struct {
	OwnerValue  string
	RepoValue   string
	NumberValue int

	TitleValue   string
	BodyValue    string
	LocatorValue string

	BranchBase string
	BranchName string

	MergeStateValue    *pull.MergeState
	MergeStateErrValue error

	LabelValue    []string
	LabelErrValue error

	CommentValue    []string
	CommentErrValue error

	CommitsValue    []*pull.Commit
	CommitsErrValue error

	RequiredStatusesValue    []string
	RequiredStatusesErrValue error

	PushRestrictionsValue    bool
	PushRestrictionsErrValue error

	SuccessStatusesValue    []string
	SuccessStatusesErrValue error

	IsTargetedValue    bool
	IsTargetedErrValue error
}

func (c *MockPullContext) Owner() string {
	return c.OwnerValue
}

func (c *MockPullContext) Repo() string {
	return c.RepoValue
}

func (c *MockPullContext) Number() int {
	return c.NumberValue
}

func (c *MockPullContext) Locator() string {
	if c.LocatorValue != "" {
		return c.LocatorValue
	}
	return "pulltest/context#1"
}

func (c *MockPullContext) Title() string {
	return c.TitleValue
}

func (c *MockPullContext) Body() string {
	return c.BodyValue
}

func (c *MockPullContext) Branches() (base string, head string) {
	return c.BranchBase, c.BranchName
}

func (c *MockPullContext) MergeState(ctx context.Context) (*pull.MergeState, error) {
	return c.MergeStateValue, c.MergeStateErrValue
}

func (c *MockPullContext) Comments(ctx context.Context) ([]string, error) {
	return c.CommentValue, c.CommentErrValue
}

func (c *MockPullContext) Commits(ctx context.Context) ([]*pull.Commit, error) {
	return c.CommitsValue, c.CommitsErrValue
}

func (c *MockPullContext) RequiredStatuses(ctx context.Context) ([]string, error) {
	return c.RequiredStatusesValue, c.RequiredStatusesErrValue
}

func (c *MockPullContext) PushRestrictions(ctx context.Context) (bool, error) {
	return c.PushRestrictionsValue, c.PushRestrictionsErrValue
}

func (c *MockPullContext) CurrentSuccessStatuses(ctx context.Context) ([]string, error) {
	return c.SuccessStatusesValue, c.SuccessStatusesErrValue
}

func (c *MockPullContext) Labels(ctx context.Context) ([]string, error) {
	return c.LabelValue, c.LabelErrValue
}

func (c *MockPullContext) IsTargeted(ctx context.Context) (bool, error) {
	return c.IsTargetedValue, c.IsTargetedErrValue
}

// type assertion
var _ pull.Context = &MockPullContext{}
