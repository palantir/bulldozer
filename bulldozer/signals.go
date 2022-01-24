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

type Signal []string

type Signals struct {
	Labels            Signal `yaml:"labels"`
	CommentSubstrings Signal `yaml:"comment_substrings"`
	Comments          Signal `yaml:"comments"`
	PRBodySubstrings  Signal `yaml:"pr_body_substrings"`
	Branches          Signal `yaml:"branches"`
	BranchPatterns    Signal `yaml:"branch_patterns"`
	MaxCommits        int    `yaml:"max_commits"`
}

// count returns the number of signals that are non-zero value
func (s *Signals) count() int {
	count := 0
	if len(s.Labels) > 0 {
		count += 1
	}
	if len(s.CommentSubstrings) > 0 {
		count += 1
	}
	if len(s.Comments) > 0 {
		count += 1
	}
	if len(s.PRBodySubstrings) > 0 {
		count += 1
	}
	if len(s.Branches) > 0 {
		count += 1
	}
	if len(s.BranchPatterns) > 0 {
		count += 1
	}
	if s.MaxCommits > 0 {
		count += 1
	}
	return count
}

func (s *Signals) Enabled() bool {
	return s.count() > 0
}

// MatchesAll returns true if the pull request matches ALL of the signals. It also
// returns a description of the match status. The tag argument appears
// in this description and indicates the behavior (trigger, ignore) this
// set of signals is associated with.
func (s *Signals) MatchesAll(ctx context.Context, pullCtx pull.Context, tag string) (bool, string, error) {
	matches := Signals{}
	var err error

	_, matches.Labels, err = s.labelMatches(ctx, pullCtx, tag)
	// TODO: Collect errors as a list
	if err != nil {
		return false, "", err
	}

	_, matches.Comments, err = s.commentMatches(ctx, pullCtx, tag)
	if err != nil {
		return false, "", err
	}

	_, matches.CommentSubstrings, err = s.commentSubstringMatches(ctx, pullCtx, tag)
	if err != nil {
		return false, "", err
	}

	_, matches.PRBodySubstrings, err = s.prBodyMatches(ctx, pullCtx, tag)
	if err != nil {
		return false, "", err
	}

	_, matches.Branches, err = s.branchMatches(ctx, pullCtx, tag)
	if err != nil {
		return false, "", err
	}

	_, matches.BranchPatterns, err = s.branchPatternMatches(ctx, pullCtx, tag)
	if err != nil {
		return false, "", err
	}

	_, matches.MaxCommits, err = s.maxCommitsMatches(ctx, pullCtx, tag)
	if err != nil {
		return false, "", err
	}

	if matches.count() == s.count() {
		return true, fmt.Sprintf("pull request matches all %s signals", tag), nil
	}

	return false, fmt.Sprintf("pull request does not match all %s signals", tag), nil
}

// MatchesAny returns true if the pull request meets one or more signals. It also
// returns a description of the signal that was met. The tag argument appears
// in this description and indicates the behavior (trigger, ignore) this
// set of signals is associated with.
func (s *Signals) MatchesAny(ctx context.Context, pullCtx pull.Context, tag string) (bool, string, error) {
	matches, description, err := s.labelMatches(ctx, pullCtx, tag)
	if err != nil {
		return false, "", err
	}

	if matches {
		return true, description, nil
	}

	matches, description, err = s.commentMatches(ctx, pullCtx, tag)
	if err != nil {
		return false, "", err
	}

	if matches {
		return true, description, nil
	}

	matches, description, err = s.commentSubstringMatches(ctx, pullCtx, tag)
	if err != nil {
		return false, "", err
	}

	if matches {
		return true, description, nil
	}

	matches, description, err = s.prBodyMatches(ctx, pullCtx, tag)
	if err != nil {
		return false, "", err
	}

	if matches {
		return true, description, nil
	}

	matches, description, err = s.branchMatches(ctx, pullCtx, tag)
	if err != nil {
		return false, "", err
	}

	if matches {
		return true, description, nil
	}

	matches, description, err = s.branchPatternMatches(ctx, pullCtx, tag)
	if err != nil {
		return false, "", err
	}

	if matches {
		return true, description, nil
	}

	return false, fmt.Sprintf("pull request does not match the %s", tag), nil
}

// labelMatches determines which label signals match the given PR. It returns:
// - A list of descriptions for each matched signal
//   - These will only include the first item the signal matched when there is more than one
// - A list of the matched signals
func (s *Signals) labelMatches(ctx context.Context, pullCtx pull.Context, tag string) ([]string, Signal, error) {
	logger := zerolog.Ctx(ctx)
	matches := Signal{}
	descriptions := []string{}

	if len(s.Labels) == 0 {
		logger.Debug().Msgf("No label singals have been provided to match against")
		return descriptions, matches, nil
	}

	labels, err := pullCtx.Labels(ctx)
	if err != nil {
		return descriptions, matches, errors.Wrap(err, "unable to list pull request labels")
	}

	if len(labels) == 0 {
		logger.Debug().Msgf("No labels found to match against")
		return descriptions, matches, nil
	}

	for _, signalLabel := range s.Labels {
		for _, label := range labels {
			if strings.EqualFold(signalLabel, label) {
				matches = append(matches, signalLabel)
				descriptions = append(descriptions, fmt.Sprintf("pull request has a %s label: %q", tag, signalLabel))
			}
		}
	}

	return descriptions, matches, nil
}

// commentMatches determines which comment signals match the given PR. It returns:
// - A list of descriptions for each matched signal
//   - These will only include the first item the signal matched when there is more than one
// - A list of the matched signals
func (s *Signals) commentMatches(ctx context.Context, pullCtx pull.Context, tag string) ([]string, Signal, error) {
	logger := zerolog.Ctx(ctx)
	matches := Signal{}
	descriptions := []string{}

	if len(s.Comments) == 0 {
		logger.Debug().Msgf("No comment singals have been provided to match against")
		return descriptions, matches, nil
	}

	body := pullCtx.Body()
	comments, err := pullCtx.Comments(ctx)
	if err != nil {
		return descriptions, matches, errors.Wrap(err, "unable to list pull request comments")
	}

	if len(comments) == 0 && body == "" {
		logger.Debug().Msgf("No comments or body content found to match against")
		return descriptions, matches, nil
	}

	for _, signalComment := range s.Comments {
		if body == signalComment {
			matches = append(matches, signalComment)
			descriptions = append(descriptions, fmt.Sprintf("pull request body is a %s comment: %q", tag, signalComment))
		} else {
			for _, comment := range comments {
				if comment == signalComment {
					matches = append(matches, signalComment)
					descriptions = append(descriptions, fmt.Sprintf("pull request has a %s comment: %q", tag, signalComment))
				}
			}
		}

	}

	return descriptions, matches, nil
}

// commentSubstringMatches determines which comment substring signals match the given PR. It returns:
// - A list of descriptions for each matched signal
//   - These will only include the first item the signal matched when there is more than one
// - A list of the matched signals
func (s *Signals) commentSubstringMatches(ctx context.Context, pullCtx pull.Context, tag string) ([]string, Signal, error) {
	logger := zerolog.Ctx(ctx)
	matches := Signal{}
	descriptions := []string{}

	if len(s.CommentSubstrings) == 0 {
		logger.Debug().Msgf("No comment substring singals have been provided to match against")
		return descriptions, matches, nil
	}

	body := pullCtx.Body()
	comments, err := pullCtx.Comments(ctx)
	if err != nil {
		return descriptions, matches, errors.Wrap(err, "unable to list pull request comments")
	}

	if len(comments) == 0 && body == "" {
		logger.Debug().Msgf("No comments or body content found to match against")
		return descriptions, matches, nil
	}

	for _, signalSubstring := range s.CommentSubstrings {
		if strings.Contains(body, signalSubstring) {
			matches = append(matches, signalSubstring)
			descriptions = append(descriptions, fmt.Sprintf("pull request body matches a %s substring: %q", tag, signalSubstring))
		} else {
			for _, comment := range comments {
				if strings.Contains(comment, signalSubstring) {
					matches = append(matches, signalSubstring)
					descriptions = append(descriptions, fmt.Sprintf("pull request comment matches a %s substring: %q", tag, signalSubstring))
				}
			}
		}
	}

	return descriptions, matches, nil
}

// prBodyMatches determines which PR body signals match the given PR. It returns:
// - A list of descriptions for each matched signal
//   - These will only include the first item the signal matched when there is more than one
// - A list of the matched signals
func (s *Signals) prBodyMatches(ctx context.Context, pullCtx pull.Context, tag string) ([]string, Signal, error) {
	logger := zerolog.Ctx(ctx)
	matches := Signal{}
	descriptions := []string{}

	if len(s.PRBodySubstrings) == 0 {
		logger.Debug().Msgf("No pr body substring singals have been provided to match against")
		return descriptions, matches, nil
	}

	body := pullCtx.Body()

	if body == "" {
		logger.Debug().Msgf("No body content found to match against")
		return descriptions, matches, nil
	}

	for _, signalSubstring := range s.PRBodySubstrings {
		if strings.Contains(body, signalSubstring) {
			matches = append(matches, signalSubstring)
			descriptions = append(descriptions, fmt.Sprintf("pull request body matches a %s substring: %q", tag, signalSubstring))
		}
	}

	return descriptions, matches, nil
}

// branchMatches determines which branch signals match the given PR. It returns:
// - A list of descriptions for each matched signal
//   - These will only include the first item the signal matched when there is more than one
// - A list of the matched signals
func (s *Signals) branchMatches(ctx context.Context, pullCtx pull.Context, tag string) ([]string, Signal, error) {
	logger := zerolog.Ctx(ctx)
	matches := Signal{}
	descriptions := []string{}

	if len(s.Branches) == 0 {
		logger.Debug().Msgf("No branch singals have been provided to match against")
		return descriptions, matches, nil
	}

	targetBranch, _ := pullCtx.Branches()

	for _, signalBranch := range s.Branches {
		if targetBranch == signalBranch {
			matches = append(matches, signalBranch)
			descriptions = append(descriptions, fmt.Sprintf("pull request target is a %s branch: %q", tag, signalBranch))
		}
	}

	return descriptions, matches, nil
}

// branchPatternMatches determines which branch pattern signals match the given PR. It returns:
// - A list of descriptions for each matched signal
//   - These will only include the first item the signal matched when there is more than one
// - A list of the matched signals
func (s *Signals) branchPatternMatches(ctx context.Context, pullCtx pull.Context, tag string) ([]string, Signal, error) {
	logger := zerolog.Ctx(ctx)
	matches := Signal{}
	descriptions := []string{}

	if len(s.BranchPatterns) == 0 {
		logger.Debug().Msgf("No branch pattern singals have been provided to match against")
		return descriptions, matches, nil
	}

	targetBranch, _ := pullCtx.Branches()

	for _, signalBranch := range s.BranchPatterns {
		if matched, _ := regexp.MatchString(fmt.Sprintf("^%s$", signalBranch), targetBranch); matched {
			matches = append(matches, signalBranch)
			descriptions = append(descriptions, fmt.Sprintf("pull request target branch (%q) matches pattern: %q", targetBranch, signalBranch))
		}
	}

	return descriptions, matches, nil
}

// maxCommitsMatches determines if the number of commits in a PR is at or below a given max. It returns:
// - An empty list if there is no match, otherwise a single string description of the match
// - A match value of 0 if there is no match, otherwise the value of the max commits signal
func (s *Signals) maxCommitsMatches(ctx context.Context, pullCtx pull.Context, tag string) ([]string, int, error) {
	logger := zerolog.Ctx(ctx)
	matches := 0
	descriptions := []string{}

	if s.MaxCommits <= 0 {
		logger.Debug().Msgf("No valid max commits value has been provided to match against")
		return descriptions, matches, nil
	}

	commits, _ := pullCtx.Commits(ctx)

	if len(commits) < s.MaxCommits {
		matches = s.MaxCommits
		descriptions = append(descriptions, fmt.Sprintf("pull request has %q commits, which is less than the maximum of %q", len(commits), s.MaxCommits))
	}

	if len(commits) == s.MaxCommits {
		matches = s.MaxCommits
		descriptions = append(descriptions, fmt.Sprintf("pull request has %q commits, which is the maximum", len(commits)))
	}

	return descriptions, matches, nil
}
