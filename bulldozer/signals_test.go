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

	"github.com/palantir/bulldozer/pull"
	"github.com/palantir/bulldozer/pull/pulltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignalsMatchesAny(t *testing.T) {
	signals := Signals{
		Labels:            []string{"LABEL_MERGE"},
		Comments:          []string{"FULL_COMMENT_PLZ_MERGE"},
		CommentSubstrings: []string{":+1:"},
		PRBodySubstrings:  []string{"BODY_MERGE_PLZ"},
		Branches:          []string{"develop"},
		BranchPatterns:    []string{"test/.*", "^feature/.*$"},
		AutoMerge:         true,
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
		"autoMergeMatch": {
			PullContext: &pulltest.MockPullContext{
				AutoMergeValue: true,
			},
			Matches: true,
			Reason:  "pull request is configured to auto merge",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			matches, reason, err := signals.MatchesAny(ctx, test.PullContext, "testlist")
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

func TestSignalsMatchesAnyNoSignals(t *testing.T) {
	signals := Signals{}

	ctx := context.Background()

	tests := map[string]struct {
		PullContext pull.Context
		Matches     bool
		Reason      string
	}{
		"noMatchNoSignalsProvidedWithEmptyPR": {
			PullContext: &pulltest.MockPullContext{},
			Matches:     false,
			Reason:      `no testlist signals provided to match against`,
		},
		"noMatchNoSignalsProvidedWithNonEmptyPR": {
			PullContext: &pulltest.MockPullContext{
				CommentValue: []string{"FULL_COMMENT_PLZ_MERGE"},
			},
			Matches: false,
			Reason:  `no testlist signals provided to match against`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			matches, reason, err := signals.MatchesAny(ctx, test.PullContext, "testlist")
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

func TestSignalsMaxCommits(t *testing.T) {
	signals := Signals{
		MaxCommits: 2,
	}
	ctx := context.Background()

	tests := map[string]struct {
		PullContext pull.Context
		Matches     bool
		Reason      string
	}{
		"noMatchWithGreaterThanMaxCommits": {
			PullContext: &pulltest.MockPullContext{
				CommitsValue: []*pull.Commit{
					{SHA: "1", Message: "commit 1"},
					{SHA: "2", Message: "commit 2"},
					{SHA: "3", Message: "commit 3"},
				},
			},
			Matches: false,
			Reason:  `pull request does not match all testlist signals`,
		},
		"matchWithLessThanMaxCommits": {
			PullContext: &pulltest.MockPullContext{
				CommitsValue: []*pull.Commit{
					{SHA: "1", Message: "commit 1"},
				},
			},
			Matches: true,
			Reason:  `pull request matches all testlist signals`,
		},
		"matchWithMaxCommits": {
			PullContext: &pulltest.MockPullContext{
				CommitsValue: []*pull.Commit{
					{SHA: "1", Message: "commit 1"},
					{SHA: "2", Message: "commit 2"},
				},
			},
			Matches: true,
			Reason:  `pull request matches all testlist signals`,
		},
		"matchWithZeroCommits": {
			PullContext: &pulltest.MockPullContext{
				CommitsValue: []*pull.Commit{},
			},
			Matches: true,
			Reason:  `pull request matches all testlist signals`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			matches, reason, err := signals.MatchesAll(ctx, test.PullContext, "testlist")
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

func TestSignalsMatchesAll(t *testing.T) {
	signals := Signals{
		Labels:            []string{"LABEL_MERGE", "OTHER_LABEL"},
		Comments:          []string{"FULL_COMMENT_PLZ_MERGE", "OTHER_COMMENT"},
		CommentSubstrings: []string{"PLZ_MERGE", "OTHER_SUBSTRING"},
		PRBodySubstrings:  []string{":+1:", "OTHER_SUBSTRING"},
		Branches:          []string{"test/v9.9.9", "other"},
		BranchPatterns:    []string{"test/.*", "^feature/.*$"},
		MaxCommits:        2,
		AutoMerge:         true,
	}

	ctx := context.Background()

	tests := map[string]struct {
		PullContext pull.Context
		Matches     bool
		Reason      string
	}{
		"matchWithAll": {
			PullContext: &pulltest.MockPullContext{
				LabelValue:   []string{"LABEL_MERGE"},
				CommentValue: []string{"FULL_COMMENT_PLZ_MERGE"},
				BodyValue:    "My PR Body\n\n\n:+1:",
				BranchBase:   "test/v9.9.9",
				CommitsValue: []*pull.Commit{
					{SHA: "1", Message: "commit 1"},
					{SHA: "2", Message: "commit 2"},
				},
				AutoMergeValue: true,
			},
			Matches: true,
			Reason:  `pull request matches all testlist signals`,
		},
		"noMatchWithMissingElements": {
			PullContext: &pulltest.MockPullContext{
				LabelValue:   []string{"LABEL_MERGE"},
				CommentValue: []string{"FULL_COMMENT_PLZ_MERGE"},
				BodyValue:    "My PR Body\n\n\n:-1:",
				BranchBase:   "test/v1.1.1",
				CommitsValue: []*pull.Commit{
					{SHA: "1", Message: "commit 1"},
					{SHA: "2", Message: "commit 2"},
				},
				AutoMergeValue: false,
			},
			Matches: false,
			Reason:  `pull request does not match all testlist signals`,
		},
		"noMatchWithEmptyPR": {
			PullContext: &pulltest.MockPullContext{},
			Matches:     false,
			Reason:      `pull request does not match all testlist signals`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			matches, reason, err := signals.MatchesAll(ctx, test.PullContext, "testlist")
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

func TestSignalsMatchesAllNoSignals(t *testing.T) {
	signals := Signals{}

	ctx := context.Background()

	tests := map[string]struct {
		PullContext pull.Context
		Matches     bool
		Reason      string
	}{
		"noMatchNoSignalsProvidedWithEmptyPR": {
			PullContext: &pulltest.MockPullContext{},
			Matches:     false,
			Reason:      `no testlist signals provided to match against`,
		},
		"noMatchNoSignalsProvidedWithNonEmptyPR": {
			PullContext: &pulltest.MockPullContext{
				CommentValue: []string{"FULL_COMMENT_PLZ_MERGE"},
			},
			Matches: false,
			Reason:  `no testlist signals provided to match against`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			matches, reason, err := signals.MatchesAll(ctx, test.PullContext, "testlist")
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
