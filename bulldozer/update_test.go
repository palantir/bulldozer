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

	"github.com/palantir/bulldozer/pull"
	"github.com/palantir/bulldozer/pull/pulltest"
	"github.com/stretchr/testify/require"
)

func TestShouldUpdatePR(t *testing.T) {
	ctx := context.Background()
	testMatrix := []struct {
		ignoreEnabled   bool
		ignored         bool
		triggerEnabled  bool
		triggered       bool
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
		pullCtx, updateConfig := generateUpdateTestCase(testCase.ignoreEnabled, testCase.ignored, testCase.triggerEnabled, testCase.triggered)
		updating, err := ShouldUpdatePR(ctx, pullCtx, updateConfig)
		require.NoError(t, err)
		msg := fmt.Sprintf("case %d - ignoreEnabled=%t ignored=%t triggerEnabled=%t triggered=%t -> doUpdate=%t",
			ndx, testCase.ignoreEnabled, testCase.ignored, testCase.triggerEnabled, testCase.triggered, testCase.expectingUpdate)
		require.Equal(t, testCase.expectingUpdate, updating, msg)
	}
}
func generateUpdateTestCase(ignorable bool, ignored bool, triggerable bool, triggered bool) (pull.Context, UpdateConfig) {
	updateConfig := UpdateConfig{}
	pullCtx := pulltest.MockPullContext{}

	if ignorable {
		updateConfig.Ignore.Labels = append(updateConfig.Ignore.Labels, "ignore")
	}

	if ignored {
		pullCtx.LabelValue = append(pullCtx.LabelValue, "ignore")
	}

	if triggerable {
		updateConfig.Trigger.Labels = append(updateConfig.Trigger.Labels, "trigger")
	}

	if triggered {
		pullCtx.LabelValue = append(pullCtx.LabelValue, "trigger")
	}

	return &pullCtx, updateConfig
}
