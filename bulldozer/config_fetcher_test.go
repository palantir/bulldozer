// Copyright 2020 Palantir Technologies, Inc.
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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalling(t *testing.T) {
	t.Run("parseNewConfig", func(t *testing.T) {
		cf := NewConfigFetcher("", []string{""}, nil)

		config := `
version: 1

merge:
  allowlist:
    labels: ["merge when ready"]
    comment_substrings: ["==MERGE_WHEN_READY=="]
  denylist:
    labels: ["do not merge"]
    comment_substrings: ["==DO_NOT_MERGE=="]
  method: squash
  options:
    squash:
      body: summarize_commits
  delete_after_merge: true

update:
  allowlist:
    labels: ["wip", "update me"]
  denylist:
    labels: ["do not update"]
`

		actual, err := cf.unmarshalConfig([]byte(config))
		require.Nil(t, err)
		assert.Equal(t, Signals{
			Labels:            []string{"merge when ready"},
			CommentSubstrings: []string{"==MERGE_WHEN_READY=="},
		}, actual.Merge.Allowlist)
		assert.Equal(t, Signals{
			Labels:            []string{"do not merge"},
			CommentSubstrings: []string{"==DO_NOT_MERGE=="},
		}, actual.Merge.Denylist)
	})

	t.Run("parseExisting", func(t *testing.T) {
		cf := NewConfigFetcher("", []string{""}, nil)

		config := `
version: 1

merge:
  whitelist:
    labels: ["merge when ready"]
    comment_substrings: ["==OLD_MERGE_WHEN_READY=="]
  blacklist:
    labels: ["do not merge"]
    comment_substrings: ["==OLD_DO_NOT_MERGE=="]
  method: squash
  options:
    squash:
      body: summarize_commits
  delete_after_merge: true

update:
  whitelist:
    labels: ["wip", "update me"]
  blacklist:
    labels: ["do not update"]
`

		actual, err := cf.unmarshalConfig([]byte(config))
		require.Nil(t, err)

		assert.Equal(t, Signals{
			Labels:            []string{"merge when ready"},
			CommentSubstrings: []string{"==OLD_MERGE_WHEN_READY=="},
		}, actual.Merge.Allowlist)
		assert.Equal(t, Signals{
			Labels:            []string{"do not merge"},
			CommentSubstrings: []string{"==OLD_DO_NOT_MERGE=="},
		}, actual.Merge.Denylist)

		assert.Equal(t, Signals{
			Labels: []string{"wip", "update me"},
		}, actual.Update.Allowlist)
		assert.Equal(t, Signals{
			Labels: []string{"do not update"},
		}, actual.Update.Denylist)
	})

	t.Run("ignoresOldConfig", func(t *testing.T) {
		cf := NewConfigFetcher("", []string{""}, nil)

		config := `
version: 1

merge:
  allowlist:
    labels: ["mwr"]
  denylist:
    labels: ["new dnm"]
  whitelist:
    labels: ["merge when ready"]
    comment_substrings: ["==OLD_MERGE_WHEN_READY=="]
  blacklist:
    labels: ["do not merge"]
    comment_substrings: ["==OLD_DO_NOT_MERGE=="]
  method: squash
  options:
    squash:
      body: summarize_commits
  delete_after_merge: true

update:
  allowlist:
    labels: ["new wip"]
  denylist:
    labels: ["new dnu"]
  whitelist:
    labels: ["wip", "update me"]
  blacklist:
    labels: ["do not update"]
`

		actual, err := cf.unmarshalConfig([]byte(config))
		require.Nil(t, err)

		assert.Equal(t, Signals{
			Labels: []string{"mwr"},
		}, actual.Merge.Allowlist)
		assert.Equal(t, Signals{
			Labels: []string{"new dnm"},
		}, actual.Merge.Denylist)

		assert.Equal(t, Signals{
			Labels: []string{"new wip"},
		}, actual.Update.Allowlist)
		assert.Equal(t, Signals{
			Labels: []string{"new dnu"},
		}, actual.Update.Denylist)
	})
}
