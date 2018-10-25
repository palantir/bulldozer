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

	TitleValue    string
	TitleErrValue error

	BodyValue    string
	BodyErrValue error

	LocatorValue string

	LabelValue    []string
	LabelErrValue error

	CommentValue    []string
	CommentErrValue error

	RequiredStatusesValue    []string
	RequiredStatusesErrValue error

	SuccessStatusesValue    []string
	SuccessStatusesErrValue error

	BranchBase     string
	BranchName     string
	BranchErrValue error
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

func (c *MockPullContext) Title(ctx context.Context) (string, error) {
	return c.TitleValue, c.TitleErrValue
}

func (c *MockPullContext) Body(ctx context.Context) (string, error) {
	return c.BodyValue, c.BodyErrValue
}

func (c *MockPullContext) Comments(ctx context.Context) ([]string, error) {
	return c.CommentValue, c.CommentErrValue
}

func (c *MockPullContext) RequiredStatuses(ctx context.Context) ([]string, error) {
	return c.RequiredStatusesValue, c.RequiredStatusesErrValue
}

func (c *MockPullContext) CurrentSuccessStatuses(ctx context.Context) ([]string, error) {
	return c.SuccessStatusesValue, c.SuccessStatusesErrValue
}

func (c *MockPullContext) Branches(ctx context.Context) (base string, head string, err error) {
	return c.BranchBase, c.BranchName, c.BranchErrValue
}

func (c *MockPullContext) Labels(ctx context.Context) ([]string, error) {
	return c.LabelValue, c.LabelErrValue
}

// type assertion
var _ pull.Context = &MockPullContext{}
