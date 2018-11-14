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

package bulldozer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/palantir/bulldozer/pull"
	"github.com/palantir/bulldozer/pull/pulltest"
)

func TestShouldUpdatePR(t *testing.T) {
	ctx := context.Background()
	testMatrix := []struct {
		blacklistable   bool
		blacklisted     bool
		whitelistable   bool
		whitelisted     bool
		expectingUpdate bool
	}{
		{false, false, false, false, false},
		{false, false, false, true, false},
		{false, false, true, false, false},
		{false, false, true, true, true},
		{false, true, false, false, false},
		{false, true, false, true, false},
		{false, true, true, false, false},
		{false, true, true, true, true},
		{true, false, false, false, true},
		{true, false, false, true, true},
		{true, false, true, false, false},
		{true, false, true, true, true},
		{true, true, false, false, false},
		{true, true, false, true, false},
		{true, true, true, false, false},
		{true, true, true, true, false},
	}

	for ndx, testCase := range testMatrix {
		pullCtx, updateConfig := generateUpdatableTestCases(testCase.blacklistable, testCase.blacklisted, testCase.whitelistable, testCase.whitelisted)
		updating, err := ShouldUpdatePR(ctx, pullCtx, updateConfig)
		require.Equal(t, testCase.expectingUpdate, updating, "case %d", ndx)
		require.NoError(t, err)
	}
}
func generateUpdatableTestCases(blacklistable bool, blacklisted bool, whitelistable bool, whitelisted bool) (pull.Context, UpdateConfig) {
	updateConfig := UpdateConfig{}
	pullCtx := pulltest.MockPullContext{}

	if blacklistable {
		updateConfig.Blacklist.Labels = append(updateConfig.Blacklist.Labels, "blacklist")
	}

	if blacklisted {
		pullCtx.LabelValue = append(pullCtx.LabelValue, "blacklist")
	}

	if whitelistable {
		updateConfig.Whitelist.Labels = append(updateConfig.Whitelist.Labels, "whitelist")
	}

	if whitelisted {
		pullCtx.LabelValue = append(pullCtx.LabelValue, "whitelist")
	}

	return &pullCtx, updateConfig
}
