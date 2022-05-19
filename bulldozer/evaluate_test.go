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

	"github.com/palantir/bulldozer/pull/pulltest"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleXListed(t *testing.T) {
	mergeConfig := MergeConfig{
		Trigger: Signals{
			Labels:            []string{"LABEL_MERGE"},
			Comments:          []string{"FULL_COMMENT_PLZ_MERGE"},
			CommentSubstrings: []string{":+1:"},
			PRBodySubstrings:  []string{"BODY_MERGE_PLZ"},
			Branches:          []string{"develop"},
		},
		Ignore: Signals{
			Labels:            []string{"LABEL_NOMERGE"},
			Comments:          []string{"NO_WAY"},
			CommentSubstrings: []string{":-1:"},
			PRBodySubstrings:  []string{"BODY_NOMERGE"},
			Branches:          []string{"master"},
		},
	}

	ctx := context.Background()

	t.Run("errCommentFailsClosedDenylist", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			CommentErrValue: errors.New("failure"),
		}

		actual, _, err := IsPRIgnored(ctx, pc, mergeConfig.Ignore)
		require.NotNil(t, err)
		assert.True(t, actual)
	})

	t.Run("errCommentFailsClosedAllowlist", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			CommentErrValue: errors.New("failure"),
		}

		actual, _, err := IsPRTriggered(ctx, pc, mergeConfig.Trigger)
		require.NotNil(t, err)
		assert.False(t, actual)
	})

	t.Run("errLabelFailsClosedAllowlist", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelErrValue: errors.New("failure"),
		}

		actual, _, err := IsPRTriggered(ctx, pc, mergeConfig.Trigger)
		require.NotNil(t, err)
		assert.False(t, actual)
	})

	t.Run("errCommentsFailsClosedAllowlist", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			CommentErrValue: errors.New("failure"),
		}

		actual, _, err := IsPRTriggered(ctx, pc, mergeConfig.Trigger)
		require.NotNil(t, err)
		assert.False(t, actual)
	})
}

func TestShouldMerge(t *testing.T) {
	mergeConfig := MergeConfig{
		Trigger: Signals{
			Labels:            []string{"LABEL_MERGE", "LABEL2_MERGE"},
			Comments:          []string{"FULL_COMMENT_PLZ_MERGE"},
			CommentSubstrings: []string{":+1:", ":y:"},
		},
		Ignore: Signals{
			Labels:            []string{"LABEL_NOMERGE"},
			Comments:          []string{"NO_WAY"},
			CommentSubstrings: []string{":-1:"},
		},
		AllowMergeWithNoChecks: true,
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

	t.Run("labelShouldMergeCaseInsensitive", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue: []string{"LABEL_merGE"},
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

	t.Run("ignoreOverridesAllowlist", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue:   []string{"LABEL2_MERGE"},
			CommentValue: []string{"NO_WAY"},
		}

		actualShouldMerge, err := ShouldMergePR(ctx, pc, mergeConfig)

		require.Nil(t, err)
		assert.False(t, actualShouldMerge)
	})

	t.Run("labelCausesDenylist", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue: []string{"LABEL_NOMERGE"},
		}

		actualShouldMerge, err := ShouldMergePR(ctx, pc, mergeConfig)

		require.Nil(t, err)
		assert.False(t, actualShouldMerge)
	})

	t.Run("labelCausesDenylistCaseInsensitive", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue: []string{"LABEL_nomERGE"},
		}

		actualShouldMerge, err := ShouldMergePR(ctx, pc, mergeConfig)

		require.Nil(t, err)
		assert.False(t, actualShouldMerge)
	})

	t.Run("substringCausesAllowlist", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue:   []string{"NOT_A_LABEL"},
			CommentValue: []string{"a comment", "another comment", "this is good :+1: yep"},
		}

		actualShouldMerge, err := ShouldMergePR(ctx, pc, mergeConfig)

		require.Nil(t, err)
		assert.True(t, actualShouldMerge)
	})

	t.Run("substringCausesDenylist", func(t *testing.T) {
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

	t.Run("travisCiPushCheckMet", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue:            []string{"LABEL_MERGE"},
			SuccessStatusesValue:  []string{"continuous-integration/travis-ci/push", "StatusCheckA"},
			RequiredStatusesValue: []string{"continuous-integration/travis-ci"},
		}

		actualShouldMerge, err := ShouldMergePR(ctx, pc, mergeConfig)

		require.Nil(t, err)
		assert.True(t, actualShouldMerge)
	})

	t.Run("travisCiPrCheckMet", func(t *testing.T) {
		pc := &pulltest.MockPullContext{
			LabelValue:            []string{"LABEL_MERGE"},
			SuccessStatusesValue:  []string{"continuous-integration/travis-ci/pr", "StatusCheckA"},
			RequiredStatusesValue: []string{"continuous-integration/travis-ci"},
		}

		actualShouldMerge, err := ShouldMergePR(ctx, pc, mergeConfig)

		require.Nil(t, err)
		assert.True(t, actualShouldMerge)
	})
}

func TestShouldUpdatePR(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name            string
		pullCtx         pulltest.MockPullContext
		updateConfig    UpdateConfig
		expectingUpdate bool
	}{
		// Test default cases and trigger / ignore handling (excluding drafts and required statuses)
		{
			name: "default",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: false,
			},
			updateConfig:    UpdateConfig{},
			expectingUpdate: false,
		},
		{
			name: "labelsOnly",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: false,
				LabelValue:   []string{"ignore", "trigger"},
			},
			updateConfig:    UpdateConfig{},
			expectingUpdate: false,
		},
		{
			name: "ignoreLabelOnly",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: false,
				LabelValue:   []string{"ignore"},
			},
			updateConfig:    UpdateConfig{},
			expectingUpdate: false,
		},
		{
			name: "triggerLabelOnly",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: false,
				LabelValue:   []string{"trigger"},
			},
			updateConfig:    UpdateConfig{},
			expectingUpdate: false,
		},
		{
			name: "configOnly",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: false,
			},
			updateConfig: UpdateConfig{
				Ignore: Signals{
					Labels: []string{"ignore"},
				},
				Trigger: Signals{
					Labels: []string{"trigger"},
				},
			},
			expectingUpdate: false,
		},
		{
			name: "ignoreConfigOnly",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: false,
			},
			updateConfig: UpdateConfig{
				Ignore: Signals{
					Labels: []string{"ignore"},
				},
			},
			expectingUpdate: true,
		},
		{
			name: "triggerConfigOnly",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: false,
			},
			updateConfig: UpdateConfig{
				Trigger: Signals{
					Labels: []string{"trigger"},
				},
			},
			expectingUpdate: false,
		},
		{
			name: "ignoreConfigTriggerLabel",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: false,
				LabelValue:   []string{"trigger"},
			},
			updateConfig: UpdateConfig{
				Ignore: Signals{
					Labels: []string{"ignore"},
				},
			},
			expectingUpdate: true,
		},
		{
			name: "triggerConfigIgnoreLabel",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: false,
				LabelValue:   []string{"ignore"},
			},
			updateConfig: UpdateConfig{
				Trigger: Signals{
					Labels: []string{"trigger"},
				},
			},
			expectingUpdate: false,
		},
		{
			name: "ignored",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: false,
				LabelValue:   []string{"ignore"},
			},
			updateConfig: UpdateConfig{
				Ignore: Signals{
					Labels: []string{"ignore"},
				},
			},
			expectingUpdate: false,
		},
		{
			name: "triggered",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: false,
				LabelValue:   []string{"trigger"},
			},
			updateConfig: UpdateConfig{
				Trigger: Signals{
					Labels: []string{"trigger"},
				},
			},
			expectingUpdate: true,
		},
		{
			name: "ignoredAndTriggered",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: false,
				LabelValue:   []string{"ignore", "trigger"},
			},
			updateConfig: UpdateConfig{
				Ignore: Signals{
					Labels: []string{"ignore"},
				},
				Trigger: Signals{
					Labels: []string{"trigger"},
				},
			},
			expectingUpdate: false,
		},
		{
			name: "triggeredWithIgnoreConfig",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: false,
				LabelValue:   []string{"trigger"},
			},
			updateConfig: UpdateConfig{
				Ignore: Signals{
					Labels: []string{"ignore"},
				},
				Trigger: Signals{
					Labels: []string{"trigger"},
				},
			},
			expectingUpdate: true,
		},
		{
			name: "ignoredWithTriggerConfig",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: false,
				LabelValue:   []string{"ignore"},
			},
			updateConfig: UpdateConfig{
				Ignore: Signals{
					Labels: []string{"ignore"},
				},
				Trigger: Signals{
					Labels: []string{"trigger"},
				},
			},
			expectingUpdate: false,
		},
		{
			name: "triggeredWithIgnoreLabel",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: false,
				LabelValue:   []string{"trigger", "ignore"},
			},
			updateConfig: UpdateConfig{
				Trigger: Signals{
					Labels: []string{"trigger"},
				},
			},
			expectingUpdate: true,
		},
		{
			name: "ignoredWithTriggerLabel",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: false,
				LabelValue:   []string{"ignore", "trigger"},
			},
			updateConfig: UpdateConfig{
				Ignore: Signals{
					Labels: []string{"ignore"},
				},
			},
			expectingUpdate: false,
		},
		// Test ignore draft handling
		{
			name: "defaultDraft",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: true,
			},
			updateConfig:    UpdateConfig{},
			expectingUpdate: false,
		},
		{
			name: "ignoredDraft",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: true,
				LabelValue:   []string{"ignore"},
			},
			updateConfig: UpdateConfig{
				Ignore: Signals{
					Labels: []string{"ignore"},
				},
			},
			expectingUpdate: false,
		},
		{
			name: "triggeredDraft",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: true,
				LabelValue:   []string{"trigger"},
			},
			updateConfig: UpdateConfig{
				Trigger: Signals{
					Labels: []string{"trigger"},
				},
			},
			expectingUpdate: true,
		},
		{
			name: "ignoreDraftsDraft",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: true,
			},
			updateConfig: UpdateConfig{
				IgnoreDrafts: boolVal(true),
			},
			expectingUpdate: false,
		},
		{
			name: "ignoreDraftsNonDraft",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: false,
			},
			updateConfig: UpdateConfig{
				IgnoreDrafts: boolVal(true),
			},
			expectingUpdate: true,
		},
		{
			name: "ignoreDraftsAndTriggeredDraft",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: true,
				LabelValue:   []string{"trigger"},
			},
			updateConfig: UpdateConfig{
				IgnoreDrafts: boolVal(true),
				Trigger: Signals{
					Labels: []string{"trigger"},
				},
			},
			expectingUpdate: true,
		},
		{
			name: "ignoreDraftsAndTriggeredNonDraft",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: false,
				LabelValue:   []string{"trigger"},
			},
			updateConfig: UpdateConfig{
				IgnoreDrafts: boolVal(true),
				Trigger: Signals{
					Labels: []string{"trigger"},
				},
			},
			expectingUpdate: true,
		},
		{
			name: "ignoreDraftsAndIgnoredDraft",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: true,
				LabelValue:   []string{"ignore"},
			},
			updateConfig: UpdateConfig{
				IgnoreDrafts: boolVal(true),
				Ignore: Signals{
					Labels: []string{"ignore"},
				},
			},
			expectingUpdate: false,
		},
		{
			name: "ignoreDraftsAndIgnoredNonDraft",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: false,
				LabelValue:   []string{"ignore"},
			},
			updateConfig: UpdateConfig{
				IgnoreDrafts: boolVal(true),
				Ignore: Signals{
					Labels: []string{"ignore"},
				},
			},
			expectingUpdate: false,
		},
		{
			name: "ignoreDraftsTriggeredAndIgnoredDraft",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: true,
				LabelValue:   []string{"ignore", "trigger"},
			},
			updateConfig: UpdateConfig{
				IgnoreDrafts: boolVal(true),
				Ignore: Signals{
					Labels: []string{"ignore"},
				},
				Trigger: Signals{
					Labels: []string{"trigger"},
				},
			},
			expectingUpdate: false,
		},
		{
			name: "ignoreDraftsTriggeredAndIgnoredNonDraft",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: false,
				LabelValue:   []string{"ignore", "trigger"},
			},
			updateConfig: UpdateConfig{
				IgnoreDrafts: boolVal(true),
				Ignore: Signals{
					Labels: []string{"ignore"},
				},
				Trigger: Signals{
					Labels: []string{"trigger"},
				},
			},
			expectingUpdate: false,
		},
		{
			name: "ignoreDraftsAndIgnoreConfigOnly",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: true,
			},
			updateConfig: UpdateConfig{
				IgnoreDrafts: boolVal(true),
				Ignore: Signals{
					Labels: []string{"ignore"},
				},
			},
			expectingUpdate: false,
		},
		{
			name: "ignoreDraftsAndTriggerConfigOnly",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue: true,
			},
			updateConfig: UpdateConfig{
				IgnoreDrafts: boolVal(true),
				Trigger: Signals{
					Labels: []string{"trigger"},
				},
			},
			expectingUpdate: false,
		},
		// Test required statuses handling
		{
			name: "statusesOnly",
			pullCtx: pulltest.MockPullContext{
				SuccessStatusesValue: []string{"status1", "status2"},
			},
			updateConfig:    UpdateConfig{},
			expectingUpdate: false,
		},
		{
			name: "fulfilledStatuses",
			pullCtx: pulltest.MockPullContext{
				SuccessStatusesValue: []string{"status1", "status2"},
			},
			updateConfig: UpdateConfig{
				RequiredStatuses: []string{"status1", "status2"},
			},
			expectingUpdate: true,
		},
		{
			name: "missingStatuses",
			pullCtx: pulltest.MockPullContext{
				SuccessStatusesValue: []string{"status1"},
			},
			updateConfig: UpdateConfig{
				RequiredStatuses: []string{"status1", "status2"},
			},
			expectingUpdate: false,
		},
		{
			name: "fulfilledTravisCIStatuses",
			pullCtx: pulltest.MockPullContext{
				SuccessStatusesValue: []string{"status1", "continuous-integration/travis-ci"},
			},
			updateConfig: UpdateConfig{
				RequiredStatuses: []string{"status1", "continuous-integration/travis-ci"},
			},
			expectingUpdate: true,
		},
		{
			name: "missingTravisCIStatuses",
			pullCtx: pulltest.MockPullContext{
				SuccessStatusesValue: []string{"status1"},
			},
			updateConfig: UpdateConfig{
				RequiredStatuses: []string{"status1", "continuous-integration/travis-ci"},
			},
			expectingUpdate: false,
		},
		{
			name: "triggeredMissingStatuses",
			pullCtx: pulltest.MockPullContext{
				LabelValue:           []string{"trigger"},
				SuccessStatusesValue: []string{"status1", "status2"},
			},
			updateConfig: UpdateConfig{
				RequiredStatuses: []string{"status1"},
				Trigger: Signals{
					Labels: []string{"trigger"},
				},
			},
			expectingUpdate: true,
		},
		{
			name: "ignoredMissingStatuses",
			pullCtx: pulltest.MockPullContext{
				LabelValue:           []string{"ignore"},
				SuccessStatusesValue: []string{"status1", "status2"},
			},
			updateConfig: UpdateConfig{
				RequiredStatuses: []string{"status1"},
				Ignore: Signals{
					Labels: []string{"ignore"},
				},
			},
			expectingUpdate: false,
		},
		{
			name: "triggeredAndIgnoredMissingStatuses",
			pullCtx: pulltest.MockPullContext{
				LabelValue:           []string{"trigger", "ignore"},
				SuccessStatusesValue: []string{"status1", "status2"},
			},
			updateConfig: UpdateConfig{
				RequiredStatuses: []string{"status1"},
				Trigger: Signals{
					Labels: []string{"trigger"},
				},
				Ignore: Signals{
					Labels: []string{"ignore"},
				},
			},
			expectingUpdate: false,
		},
		{
			name: "triggeredFulfilledStatuses",
			pullCtx: pulltest.MockPullContext{
				LabelValue:           []string{"trigger"},
				SuccessStatusesValue: []string{"status1", "status2"},
			},
			updateConfig: UpdateConfig{
				RequiredStatuses: []string{"status1", "status2"},
				Trigger: Signals{
					Labels: []string{"trigger"},
				},
			},
			expectingUpdate: true,
		},
		{
			name: "ignoredFulfilledStatuses",
			pullCtx: pulltest.MockPullContext{
				LabelValue:           []string{"ignore"},
				SuccessStatusesValue: []string{"status1", "status2"},
			},
			updateConfig: UpdateConfig{
				RequiredStatuses: []string{"status1", "status2"},
				Ignore: Signals{
					Labels: []string{"ignore"},
				},
			},
			expectingUpdate: false,
		},
		{
			name: "triggeredAndIgnoredFulfilledStatuses",
			pullCtx: pulltest.MockPullContext{
				LabelValue:           []string{"trigger", "ignore"},
				SuccessStatusesValue: []string{"status1", "status2"},
			},
			updateConfig: UpdateConfig{
				RequiredStatuses: []string{"status1", "status2"},
				Trigger: Signals{
					Labels: []string{"trigger"},
				},
				Ignore: Signals{
					Labels: []string{"ignore"},
				},
			},
			expectingUpdate: false,
		},
		{
			name: "missingStatusesDraft",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue:         true,
				SuccessStatusesValue: []string{"status1"},
			},
			updateConfig: UpdateConfig{
				RequiredStatuses: []string{"status1", "status2"},
			},
			expectingUpdate: false,
		},
		{
			name: "missingStatusesIgnoreDraftDraft",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue:         true,
				SuccessStatusesValue: []string{"status1"},
			},
			updateConfig: UpdateConfig{
				IgnoreDrafts:     boolVal(true),
				RequiredStatuses: []string{"status1", "status2"},
			},
			expectingUpdate: false,
		},
		{
			name: "fulfilledStatusesIgnoreDraftDraft",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue:         true,
				SuccessStatusesValue: []string{"status1", "status2"},
			},
			updateConfig: UpdateConfig{
				IgnoreDrafts:     boolVal(true),
				RequiredStatuses: []string{"status1", "status2"},
			},
			expectingUpdate: false,
		},
		{
			name: "missingStatusesIgnoreDraftNonDraft",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue:         false,
				SuccessStatusesValue: []string{"status1"},
			},
			updateConfig: UpdateConfig{
				IgnoreDrafts:     boolVal(true),
				RequiredStatuses: []string{"status1", "status2"},
			},
			expectingUpdate: false,
		},
		{
			name: "fulfilledStatusesIgnoreDraftNonDraft",
			pullCtx: pulltest.MockPullContext{
				IsDraftValue:         false,
				SuccessStatusesValue: []string{"status1", "status2"},
			},
			updateConfig: UpdateConfig{
				IgnoreDrafts:     boolVal(true),
				RequiredStatuses: []string{"status1", "status2"},
			},
			expectingUpdate: true,
		},
		// Test error handling
		// TestShouldUpdatePR TODO: add error scenario coverage
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			updating, err := ShouldUpdatePR(ctx, &tc.pullCtx, tc.updateConfig)
			require.NoError(t, err)
			msg := fmt.Sprintf("case %s - pullCtx %+v updateConfig %+v -> expectingUpdate=%t",
				tc.name, tc.pullCtx, tc.updateConfig, tc.expectingUpdate)
			require.Equal(t, tc.expectingUpdate, updating, msg)
		})
	}
}

func boolVal(b bool) *bool {
	return &b
}
