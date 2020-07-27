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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/palantir/bulldozer/pull"
	"github.com/palantir/bulldozer/pull/pulltest"
)

func TestShouldUpdatePR(t *testing.T) {
	ctx := context.Background()
	testMatrix := []struct {
		denylistEnabled  bool
		denied           bool
		allowlistEnabled bool
		allowed          bool
		expectingUpdate  bool
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
		pullCtx, updateConfig := generateUpdateTestCase(testCase.denylistEnabled, testCase.denied, testCase.allowlistEnabled, testCase.allowed)
		updating, err := ShouldUpdatePR(ctx, pullCtx, updateConfig)
		require.NoError(t, err)
		msg := fmt.Sprintf("case %d - denylistEnabled=%t denied=%t allowlistEnabled=%t allowed=%t -> doUpdate=%t",
			ndx, testCase.denylistEnabled, testCase.denied, testCase.allowlistEnabled, testCase.allowed, testCase.expectingUpdate)
		require.Equal(t, testCase.expectingUpdate, updating, msg)
	}
}
func generateUpdateTestCase(deniable bool, denied bool, allowable bool, allowed bool) (pull.Context, UpdateConfig) {
	updateConfig := UpdateConfig{}
	pullCtx := pulltest.MockPullContext{}

	if deniable {
		updateConfig.Denylist.Labels = append(updateConfig.Denylist.Labels, "denylist")
	}

	if denied {
		pullCtx.LabelValue = append(pullCtx.LabelValue, "denylist")
	}

	if allowable {
		updateConfig.Allowlist.Labels = append(updateConfig.Allowlist.Labels, "allowlist")
	}

	if allowed {
		pullCtx.LabelValue = append(pullCtx.LabelValue, "allowlist")
	}

	return &pullCtx, updateConfig
}
