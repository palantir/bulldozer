// Copyright 2026 Palantir Technologies, Inc.
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

package handler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsPassingCheckConclusion(t *testing.T) {
	for conclusion, expected := range map[string]bool{
		"success":         true,
		"neutral":         true,
		"skipped":         true,
		"failure":         false,
		"cancelled":       false,
		"timed_out":       false,
		"action_required": false,
		"stale":           false,
		"startup_failure": false,
		"":                false,
	} {
		assert.Equal(t, expected, isPassingCheckConclusion(conclusion), conclusion)
	}
}

func TestCheckRunIgnoresNonPassingConclusion(t *testing.T) {
	// A nil ClientCreator would panic if the handler tried to call GitHub.
	h := &CheckRun{}
	payload := []byte(`{
		"action": "completed",
		"check_run": {"conclusion": "failure", "head_sha": "abc", "pull_requests": [{"number": 1}]},
		"repository": {"name": "repo", "owner": {"login": "owner"}},
		"installation": {"id": 1}
	}`)

	require.NoError(t, h.Handle(context.Background(), "check_run", "delivery", payload))
}
