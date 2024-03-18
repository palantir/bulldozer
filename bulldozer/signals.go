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
	"fmt"
	"regexp"
	"strings"

	"github.com/palantir/bulldozer/pull"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type Signal interface {
	// Determine if the signal has values assigned to it and should be considered when matching
	Enabled() bool

	// Determine if the signal matches a value in the target pull request
	Matches(context.Context, pull.Context, string) (bool, string, error)
}

type LabelsSignal []string
type CommentSubstringsSignal []string
type CommentsSignal []string
type PRBodySubstringsSignal []string
type BranchesSignal []string
type BranchPatternsSignal []string
type MaxCommitsSignal int
type AutoMergeSignal bool
type DraftSignal bool

type Signals struct {
	Labels            LabelsSignal            `yaml:"labels"`
	CommentSubstrings CommentSubstringsSignal `yaml:"comment_substrings"`
	Comments          CommentsSignal          `yaml:"comments"`
	PRBodySubstrings  PRBodySubstringsSignal  `yaml:"pr_body_substrings"`
	Branches          BranchesSignal          `yaml:"branches"`
	BranchPatterns    BranchPatternsSignal    `yaml:"branch_patterns"`
	MaxCommits        MaxCommitsSignal        `yaml:"max_commits"`
	AutoMerge         AutoMergeSignal         `yaml:"auto_merge"`
	Draft             DraftSignal             `yaml:"draft"`
}

func (signal LabelsSignal) Enabled() bool {
	return len(signal) > 0
}

func (signal CommentSubstringsSignal) Enabled() bool {
	return len(signal) > 0
}

func (signal CommentsSignal) Enabled() bool {
	return len(signal) > 0
}

func (signal PRBodySubstringsSignal) Enabled() bool {
	return len(signal) > 0
}

func (signal BranchesSignal) Enabled() bool {
	return len(signal) > 0
}

func (signal BranchPatternsSignal) Enabled() bool {
	return len(signal) > 0
}

func (signal MaxCommitsSignal) Enabled() bool {
	return signal > 0
}

func (signal AutoMergeSignal) Enabled() bool {
	return bool(signal)
}

func (signal DraftSignal) Enabled() bool {
	return bool(signal)
}

func (s Signals) Enabled() bool {
	return s.Labels.Enabled() ||
		s.CommentSubstrings.Enabled() ||
		s.Comments.Enabled() ||
		s.PRBodySubstrings.Enabled() ||
		s.Branches.Enabled() ||
		s.BranchPatterns.Enabled() ||
		s.MaxCommits.Enabled() ||
		s.AutoMerge.Enabled()
}

// MatchesAll returns true if the pull request matches ALL of the signals. It also
// returns a description of the match status. The tag argument appears
// in this description and indicates the behavior (trigger, ignore) this
// set of signals is associated with.
func (s Signals) MatchesAll(ctx context.Context, pullCtx pull.Context, tag string) (bool, string, error) {
	if !s.Enabled() {
		return false, fmt.Sprintf("no %s signals provided to match against", tag), nil
	}

	signals := []Signal{
		&s.Labels,
		&s.CommentSubstrings,
		&s.Comments,
		&s.PRBodySubstrings,
		&s.Branches,
		&s.BranchPatterns,
		&s.MaxCommits,
		&s.AutoMerge,
	}

	for _, signal := range signals {
		if signal.Enabled() {
			matches, _, err := signal.Matches(ctx, pullCtx, tag)
			if err != nil {
				return false, "", err
			}

			if !matches {
				return false, fmt.Sprintf("pull request does not match all %s signals", tag), nil
			}
		}
	}

	return true, fmt.Sprintf("pull request matches all %s signals", tag), nil
}

// MatchesAny returns true if the pull request meets one or more signals. It also
// returns a description of the signal that was met. The tag argument appears
// in this description and indicates the behavior (trigger, ignore) this
// set of signals is associated with.
func (s Signals) MatchesAny(ctx context.Context, pullCtx pull.Context, tag string) (bool, string, error) {
	if !s.Enabled() {
		return false, fmt.Sprintf("no %s signals provided to match against", tag), nil
	}

	signals := []Signal{
		&s.Labels,
		&s.CommentSubstrings,
		&s.Comments,
		&s.PRBodySubstrings,
		&s.Branches,
		&s.BranchPatterns,
		&s.AutoMerge,
		&s.Draft,
	}

	for _, signal := range signals {
		matches, description, err := signal.Matches(ctx, pullCtx, tag)
		if err != nil {
			return false, "", err
		}

		if matches {
			return true, description, nil
		}
	}

	return false, fmt.Sprintf("pull request does not match the %s", tag), nil
}

// Matches Determines which label signals match the given PR. It returns:
// - A boolean to indicate if a signal matched
// - A description of the first matched signal
func (signal LabelsSignal) Matches(ctx context.Context, pullCtx pull.Context, tag string) (bool, string, error) {
	logger := zerolog.Ctx(ctx)

	if !signal.Enabled() {
		return false, "", nil
	}

	labels, err := pullCtx.Labels(ctx)
	if err != nil {
		return false, "", errors.Wrap(err, "unable to list pull request labels")
	}

	if len(labels) == 0 {
		logger.Debug().Msgf("No labels found to match against")
		return false, "", nil
	}

	for _, signalLabel := range signal {
		for _, label := range labels {
			if strings.EqualFold(signalLabel, label) {
				return true, fmt.Sprintf("pull request has a %s label: %q", tag, signalLabel), nil
			}
		}
	}

	return false, "", nil
}

// Matches Determines which comment signals match the given PR. It returns:
// - A boolean to indicate if a signal matched
// - A description of the first matched signal
func (signal CommentsSignal) Matches(ctx context.Context, pullCtx pull.Context, tag string) (bool, string, error) {
	logger := zerolog.Ctx(ctx)

	if !signal.Enabled() {
		return false, "", nil
	}

	body := pullCtx.Body()
	comments, err := pullCtx.Comments(ctx)
	if err != nil {
		return false, "", errors.Wrap(err, "unable to list pull request comments")
	}

	if len(comments) == 0 && body == "" {
		logger.Debug().Msgf("No comments or body content found to match against")
		return false, "", nil
	}

	for _, signalComment := range signal {
		if body == signalComment {
			return true, fmt.Sprintf("pull request body is a %s comment: %q", tag, signalComment), nil
		}
		for _, comment := range comments {
			if comment == signalComment {
				return true, fmt.Sprintf("pull request has a %s comment: %q", tag, signalComment), nil
			}
		}
	}

	return false, "", nil
}

// Matches Determines which comment substring signals match the given PR. It returns:
// - A boolean to indicate if a signal matched
// - A description of the first matched signal
func (signal CommentSubstringsSignal) Matches(ctx context.Context, pullCtx pull.Context, tag string) (bool, string, error) {
	logger := zerolog.Ctx(ctx)

	if !signal.Enabled() {
		return false, "", nil
	}

	body := pullCtx.Body()
	comments, err := pullCtx.Comments(ctx)
	if err != nil {
		return false, "", errors.Wrap(err, "unable to list pull request comments")
	}

	if len(comments) == 0 && body == "" {
		logger.Debug().Msgf("No comments or body content found to match against")
		return false, "", nil
	}

	for _, signalSubstring := range signal {
		if strings.Contains(body, signalSubstring) {
			return true, fmt.Sprintf("pull request body matches a %s substring: %q", tag, signalSubstring), nil
		}
		for _, comment := range comments {
			if strings.Contains(comment, signalSubstring) {
				return true, fmt.Sprintf("pull request comment matches a %s substring: %q", tag, signalSubstring), nil
			}
		}
	}

	return false, "", nil
}

// Matches Determines which PR body signals match the given PR. It returns:
// - A boolean to indicate if a signal matched
// - A description of the first matched signal
func (signal PRBodySubstringsSignal) Matches(ctx context.Context, pullCtx pull.Context, tag string) (bool, string, error) {
	logger := zerolog.Ctx(ctx)

	if !signal.Enabled() {
		return false, "", nil
	}

	body := pullCtx.Body()

	if body == "" {
		logger.Debug().Msgf("No body content found to match against")
		return false, "", nil
	}

	for _, signalSubstring := range signal {
		if strings.Contains(body, signalSubstring) {
			return true, fmt.Sprintf("pull request body matches a %s substring: %q", tag, signalSubstring), nil
		}
	}

	return false, "", nil
}

// Matches Determines which branch signals match the given PR. It returns:
// - A boolean to indicate if a signal matched
// - A description of the first matched signal
func (signal BranchesSignal) Matches(ctx context.Context, pullCtx pull.Context, tag string) (bool, string, error) {
	if !signal.Enabled() {
		return false, "", nil
	}

	targetBranch, _ := pullCtx.Branches()

	for _, signalBranch := range signal {
		if targetBranch == signalBranch {
			return true, fmt.Sprintf("pull request target is a %s branch: %q", tag, signalBranch), nil
		}
	}

	return false, "", nil
}

// Matches Determines which branch pattern signals match the given PR. It returns:
// - A boolean to indicate if a signal matched
// - A description of the first matched signal
func (signal BranchPatternsSignal) Matches(ctx context.Context, pullCtx pull.Context, tag string) (bool, string, error) {
	if !signal.Enabled() {
		return false, "", nil
	}

	targetBranch, _ := pullCtx.Branches()

	for _, signalBranch := range signal {
		if matched, _ := regexp.MatchString(fmt.Sprintf("^%s$", signalBranch), targetBranch); matched {
			return true, fmt.Sprintf("pull request target branch (%q) matches pattern: %q", targetBranch, signalBranch), nil
		}
	}

	return false, "", nil
}

// Matches Determines if the number of commits in a PR is at or below a given max. It returns:
// - An empty list if there is no match, otherwise a single string description of the match
// - A match value of 0 if there is no match, otherwise the value of the max commits signal
func (signal MaxCommitsSignal) Matches(ctx context.Context, pullCtx pull.Context, tag string) (bool, string, error) {
	logger := zerolog.Ctx(ctx)

	if !signal.Enabled() {
		logger.Debug().Msgf("No valid max commits value has been provided to match against")
		return false, "", nil
	}

	commits, _ := pullCtx.Commits(ctx)

	if len(commits) <= int(signal) {
		return true, fmt.Sprintf("pull request has %q commits, which is less than or equal to the maximum of %q", len(commits), signal), nil
	}

	return false, "", nil
}

func (signal AutoMergeSignal) Matches(ctx context.Context, pullCtx pull.Context, tag string) (bool, string, error) {
	logger := zerolog.Ctx(ctx)

	if !signal.Enabled() {
		logger.Debug().Msgf("No valid auto merge value has been provided to match against")
		return false, "", nil
	}

	autoMerge := pullCtx.AutoMerge(ctx)

	if autoMerge {
		return true, "pull request is configured to auto merge", nil
	}

	return false, "", nil
}

func (signal DraftSignal) Matches(ctx context.Context, pullCtx pull.Context, tag string) (bool, string, error) {
	logger := zerolog.Ctx(ctx)

	if !signal.Enabled() {
		logger.Debug().Msgf("No valid draft pr value has been provided to match against")
		return false, "", nil
	}

	isDraft := pullCtx.IsDraft(ctx)

	if isDraft {
		return true, "pull request is a draft", nil
	}

	return false, "", nil
}
