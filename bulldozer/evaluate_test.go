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

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/palantir/bulldozer/pull/pulltest"
)

func TestSimpleXListed(t *testing.T) {
	mergeConfig := MergeConfig{
		Whitelist: Signals{
			Labels:            []string{"LABEL_MERGE"},
			Comments:          []string{"FULL_COMMENT_PLZ_MERGE"},
			CommentSubstrings: []string{":+1:"},
		},
		Blacklist: Signals{
			Labels:            []string{"LABEL_NOMERGE"},
			Comments:          []string{"NO_WAY"},
			CommentSubstrings: []string{":-1:"},
		},
	}

	ctx := context.Background()

	t.Run("singleCommentCausesWhitelist", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			CommentValue: []string{"FULL_COMMENT_PLZ_MERGE"},
		}

		actualBlacklist, actualBlacklistReason, err := IsPRBlacklisted(ctx, pc, mergeConfig.Blacklist)
		require.Nil(t, err)
		assert.False(t, actualBlacklist)
		assert.Equal(t, "no matching blacklist found", actualBlacklistReason)

		actualWhitelist, actualWhitelistReason, err := IsPRWhitelisted(ctx, pc, mergeConfig.Whitelist)
		require.Nil(t, err)
		assert.True(t, actualWhitelist)
		assert.Equal(t, "PR comment matches one of specified whitelist comments: \"FULL_COMMENT_PLZ_MERGE\"", actualWhitelistReason)
	})

	t.Run("commentSubstringCausesWhitelist", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue:   []string{"LABEL_nothing"},
			CommentValue: []string{"a comment", "another comment", "this is good :+1: yep"},
		}

		actualBlacklist, actualBlacklistReason, err := IsPRBlacklisted(ctx, pc, mergeConfig.Blacklist)
		require.Nil(t, err)
		assert.False(t, actualBlacklist)
		assert.Equal(t, "no matching blacklist found", actualBlacklistReason)

		actualWhitelist, actualWhitelistReason, err := IsPRWhitelisted(ctx, pc, mergeConfig.Whitelist)
		require.Nil(t, err)
		assert.True(t, actualWhitelist)
		assert.Equal(t, "PR comment matches one of specified whitelist comment substrings: \":+1:\"", actualWhitelistReason)
	})

	t.Run("commentSubstringCausesBlacklist", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue:   []string{"LABEL_NOTHING"},
			CommentValue: []string{"a comment", "another comment", "this is no good nope\n\r:-1:\n"},
		}

		actualBlacklist, actualBlacklistReason, err := IsPRBlacklisted(ctx, pc, mergeConfig.Blacklist)
		require.Nil(t, err)
		assert.True(t, actualBlacklist)
		assert.Equal(t, "PR comment matches one of specified blacklist comment substrings: \":-1:\"", actualBlacklistReason)

		actualWhitelist, actualWhitelistReason, err := IsPRWhitelisted(ctx, pc, mergeConfig.Whitelist)
		require.Nil(t, err)
		assert.False(t, actualWhitelist)
		assert.Equal(t, "no matching whitelist found", actualWhitelistReason)
	})

	t.Run("noMatchingWhitelist", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			CommentValue: []string{""},
		}

		actualBlacklist, actualBlacklistReason, err := IsPRBlacklisted(ctx, pc, mergeConfig.Blacklist)
		require.Nil(t, err)
		assert.False(t, actualBlacklist)
		assert.Equal(t, "no matching blacklist found", actualBlacklistReason)

		actualWhitelist, actualWhitelistReason, err := IsPRWhitelisted(ctx, pc, mergeConfig.Whitelist)
		require.Nil(t, err)
		assert.False(t, actualWhitelist)
		assert.Equal(t, "no matching whitelist found", actualWhitelistReason)
	})

	t.Run("commentCausesBlacklist", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			CommentValue: []string{"NO_WAY"},
			LabelValue:   []string{"LABEL_MERGE"},
		}

		actualBlacklist, actualBlacklistReason, err := IsPRBlacklisted(ctx, pc, mergeConfig.Blacklist)
		require.Nil(t, err)
		assert.True(t, actualBlacklist)
		assert.Equal(t, "PR comment matches one of specified blacklist comments: \"NO_WAY\"", actualBlacklistReason)

		actualWhitelist, actualWhitelistReason, err := IsPRWhitelisted(ctx, pc, mergeConfig.Whitelist)
		require.Nil(t, err)
		assert.True(t, actualWhitelist)
		assert.Equal(t, "PR label matches one of specified whitelist labels: \"LABEL_MERGE\"", actualWhitelistReason)
	})

	t.Run("labelCausesBlacklist", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue: []string{"LABEL_NOMERGE"},
		}

		actualBlacklist, actualBlacklistReason, err := IsPRBlacklisted(ctx, pc, mergeConfig.Blacklist)
		require.Nil(t, err)
		assert.True(t, actualBlacklist)
		assert.Equal(t, "PR label matches one of specified blacklist labels: \"LABEL_NOMERGE\"", actualBlacklistReason)
	})

	t.Run("labelCausesBlacklist case-insensitive", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue: []string{"LABEL_nomERGE"},
		}

		actualBlacklist, actualBlacklistReason, err := IsPRBlacklisted(ctx, pc, mergeConfig.Blacklist)
		require.Nil(t, err)
		assert.True(t, actualBlacklist)
		assert.Equal(t, "PR label matches one of specified blacklist labels: \"LABEL_nomERGE\"", actualBlacklistReason)
	})

	t.Run("labelCausesWhitelist", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue: []string{"LABEL_MERGE"},
		}

		actualWhitelist, actualWhitelistReason, err := IsPRWhitelisted(ctx, pc, mergeConfig.Whitelist)
		require.Nil(t, err)
		assert.True(t, actualWhitelist)
		assert.Equal(t, "PR label matches one of specified whitelist labels: \"LABEL_MERGE\"", actualWhitelistReason)
	})

	t.Run("labelCausesWhitelist case-insensitive", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue: []string{"LABEL_merRGE"},
		}

		actualWhitelist, actualWhitelistReason, err := IsPRWhitelisted(ctx, pc, mergeConfig.Whitelist)
		require.Nil(t, err)
		assert.True(t, actualWhitelist)
		assert.Equal(t, "PR label matches one of specified whitelist labels: \"LABEL_merRGE\"", actualWhitelistReason)
	})
}

func TestShouldMerge(t *testing.T) {
	mergeConfig := MergeConfig{
		Whitelist: Signals{
			Labels:            []string{"LABEL_MERGE", "LABEL2_MERGE"},
			Comments:          []string{"FULL_COMMENT_PLZ_MERGE"},
			CommentSubstrings: []string{":+1:", ":y:"},
		},
		Blacklist: Signals{
			Labels:            []string{"LABEL_NOMERGE"},
			Comments:          []string{"NO_WAY"},
			CommentSubstrings: []string{":-1:"},
		},
	}

	ctx := context.Background()

	t.Run("fullCommentShouldMerge", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			CommentValue: []string{"FULL_COMMENT_PLZ_MERGE"},
		}

		actualShouldMerge, err := ShouldMergePR(ctx, pc, mergeConfig)

		require.Nil(t, err)
		assert.True(t, actualShouldMerge)
	})

	t.Run("partialCommentShouldntMerge", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			CommentValue: []string{"This is not a FULL_COMMENT_PLZ_MERGE"},
		}

		actualShouldMerge, err := ShouldMergePR(ctx, pc, mergeConfig)

		require.Nil(t, err)
		assert.False(t, actualShouldMerge)
	})

	t.Run("labelShouldMerge", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue: []string{"LABEL_MERGE"},
		}

		actualShouldMerge, err := ShouldMergePR(ctx, pc, mergeConfig)

		require.Nil(t, err)
		assert.True(t, actualShouldMerge)
	})

	t.Run("noContextShouldntMerge", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue:   []string{"NOT_A_LABEL"},
			CommentValue: []string{"commenta", "foo", "bar", "baz\n\rbaz"},
		}

		actualShouldMerge, err := ShouldMergePR(ctx, pc, mergeConfig)

		require.Nil(t, err)
		assert.False(t, actualShouldMerge)
	})

	t.Run("noMatchingShouldntMerge", func(t *testing.T) {
		pc := &pulltest.MockPullContext{}

		actualShouldMerge, err := ShouldMergePR(ctx, pc, mergeConfig)

		require.Nil(t, err)
		assert.False(t, actualShouldMerge)
	})

	t.Run("blacklistOverridesWhitelist", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue:   []string{"LABEL2_MERGE"},
			CommentValue: []string{"NO_WAY"},
		}

		actualShouldMerge, err := ShouldMergePR(ctx, pc, mergeConfig)

		require.Nil(t, err)
		assert.False(t, actualShouldMerge)
	})

	t.Run("labelCausesBlacklist", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue: []string{"LABEL_NOMERGE"},
		}

		actualShouldMerge, err := ShouldMergePR(ctx, pc, mergeConfig)

		require.Nil(t, err)
		assert.False(t, actualShouldMerge)
	})

	t.Run("substringCausesWhitelist", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue:   []string{"NOT_A_LABEL"},
			CommentValue: []string{"a comment", "another comment", "this is good :+1: yep"},
		}

		actualShouldMerge, err := ShouldMergePR(ctx, pc, mergeConfig)

		require.Nil(t, err)
		assert.True(t, actualShouldMerge)
	})

	t.Run("substringCausesBlacklist", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue:   []string{"LABEL_NOMERGE"},
			CommentValue: []string{"a comment", "another comment", "this is no good nope\n\r:-1:"},
		}

		actualShouldMerge, err := ShouldMergePR(ctx, pc, mergeConfig)

		require.Nil(t, err)
		assert.False(t, actualShouldMerge)
	})

	t.Run("failClosedOnLabelErr", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue:    []string{"LABEL_NOMERGE"},
			CommentValue:  []string{"a comment", "another comment", "this is no good nope\n\r:-1:"},
			LabelErrValue: errors.New("failure"),
		}

		actualShouldMerge, err := ShouldMergePR(ctx, pc, mergeConfig)

		require.NotNil(t, err)
		assert.False(t, actualShouldMerge)
	})

	t.Run("failClosedOnCommentErr", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			CommentValue:    []string{"a comment", "another comment", "this is no good nope\n\r:-1:"},
			CommentErrValue: errors.New("failure"),
		}

		actualShouldMerge, err := ShouldMergePR(ctx, pc, mergeConfig)

		require.NotNil(t, err)
		assert.False(t, actualShouldMerge)
	})

	t.Run("failClosedOnRequiredStatusCheckErr", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue:               []string{"LABEL_MERGE"},
			RequiredStatusesErrValue: errors.New("failure"),
		}

		actualShouldMerge, err := ShouldMergePR(ctx, pc, mergeConfig)

		require.NotNil(t, err)
		assert.False(t, actualShouldMerge)
	})

	t.Run("failClosedOnSuccessStatusCheckErr", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue:              []string{"LABEL_MERGE"},
			SuccessStatusesErrValue: errors.New("failure"),
		}

		actualShouldMerge, err := ShouldMergePR(ctx, pc, mergeConfig)

		require.NotNil(t, err)
		assert.False(t, actualShouldMerge)
	})

	t.Run("allStatusChecksMet", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue:            []string{"LABEL_MERGE"},
			SuccessStatusesValue:  []string{"StatusCheckA", "StatusCheckB"},
			RequiredStatusesValue: []string{"StatusCheckB", "StatusCheckA"},
		}

		actualShouldMerge, err := ShouldMergePR(ctx, pc, mergeConfig)

		require.Nil(t, err)
		assert.True(t, actualShouldMerge)
	})

	t.Run("notAllStatusChecksMet", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue:            []string{"LABEL_MERGE"},
			SuccessStatusesValue:  []string{"StatusCheckA"},
			RequiredStatusesValue: []string{"StatusCheckA", "StatusCheckB"},
		}

		actualShouldMerge, err := ShouldMergePR(ctx, pc, mergeConfig)

		require.Nil(t, err)
		assert.False(t, actualShouldMerge)
	})
}
