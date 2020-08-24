// Copyright 2019 Palantir Technologies, Inc.
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/palantir/bulldozer/pull"
	"github.com/palantir/bulldozer/pull/pulltest"
)

func TestSignalsMatches(t *testing.T) {
	signals := Signals{
		Labels:            []string{"LABEL_MERGE"},
		Comments:          []string{"FULL_COMMENT_PLZ_MERGE"},
		CommentSubstrings: []string{":+1:"},
		PRBodySubstrings:  []string{"BODY_MERGE_PLZ"},
		Branches:          []string{"develop"},
		BranchPatterns:    []string{"test/.*", "^feature/.*$"},
	}

	ctx := context.Background()

	tests := map[string]struct {
		PullContext pull.Context
		Matches     bool
		Reason      string
	}{
		"noMatch": {
			PullContext: &pulltest.MockPullContext{
				CommentValue: []string{""},
			},
			Matches: false,
			Reason:  `pull request does not match the testlist`,
		},
		"commentMatchesComment": {
			PullContext: &pulltest.MockPullContext{
				CommentValue: []string{"FULL_COMMENT_PLZ_MERGE"},
			},
			Matches: true,
			Reason:  `pull request has a testlist comment: "FULL_COMMENT_PLZ_MERGE"`,
		},
		"commentMatchesCommentSubstring": {
			PullContext: &pulltest.MockPullContext{
				LabelValue:   []string{"LABEL_nothing"},
				CommentValue: []string{"a comment", "another comment", "this is good :+1: yep"},
			},
			Matches: true,
			Reason:  `pull request comment matches a testlist substring: ":+1:"`,
		},
		"labelMatchesLabel": {
			PullContext: &pulltest.MockPullContext{
				LabelValue: []string{"LABEL_MERGE"},
			},
			Matches: true,
			Reason:  `pull request has a testlist label: "LABEL_MERGE"`,
		},
		"labelMatchesLabelCaseInsensitive": {
			PullContext: &pulltest.MockPullContext{
				LabelValue: []string{"LABEL_meRGE"},
			},
			Matches: true,
			Reason:  `pull request has a testlist label: "LABEL_MERGE"`,
		},
		"bodyMatchesBodySubstring": {
			PullContext: &pulltest.MockPullContext{
				BodyValue: "My PR Body\n\n\n BODY_MERGE_PLZ",
			},
			Matches: true,
			Reason:  `pull request body matches a testlist substring: "BODY_MERGE_PLZ"`,
		},
		"bodyMatchesComment": {
			PullContext: &pulltest.MockPullContext{
				BodyValue: "FULL_COMMENT_PLZ_MERGE",
			},
			Matches: true,
			Reason:  `pull request body is a testlist comment: "FULL_COMMENT_PLZ_MERGE"`,
		},
		"bodyMatchesCommentSubstring": {
			PullContext: &pulltest.MockPullContext{
				BodyValue: "My PR Body\n\n\n:+1:",
			},
			Matches: true,
			Reason:  `pull request body matches a testlist substring: ":+1:"`,
		},
		"targetBranchMatchesBranch": {
			PullContext: &pulltest.MockPullContext{
				BranchBase: "develop",
			},
			Matches: true,
			Reason:  `pull request target is a testlist branch: "develop"`,
		},
		"targetBranchMatchesBranchWildcard": {
			PullContext: &pulltest.MockPullContext{
				BranchBase: "test/v9.9.9",
			},
			Matches: true,
			Reason:  `pull request target branch ("test/v9.9.9") matches pattern: "test/.*"`,
		},
		"targetBranchLikeSignalWithSpecialChars": {
			PullContext: &pulltest.MockPullContext{
				BranchBase: "testlist/v9.9.9",
			},
			Matches: false,
			Reason:  `pull request does not match the testlist`,
		},
		"wildcardNoMatch": {
			PullContext: &pulltest.MockPullContext{
				BranchBase: "pretest/pretest",
			},
			Matches: false,
			Reason:  `pull request does not match the testlist`,
		},
		"signalBranchContainsBoundaryMarkers": {
			PullContext: &pulltest.MockPullContext{
				BranchBase: "feature/awesomeFeature",
			},
			Matches: true,
			Reason:  `pull request target branch ("feature/awesomeFeature") matches pattern: "^feature/.*$"`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			matches, reason, err := signals.Matches(ctx, test.PullContext, "testlist")
			require.NoError(t, err)

			if test.Matches {
				assert.True(t, matches, "expected pull request to match, but it didn't")
			} else {
				assert.False(t, matches, "expected pull request to not match, but it did")
			}
			assert.Equal(t, test.Reason, reason)
		})
	}
}
