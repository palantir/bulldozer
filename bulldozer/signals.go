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
	"github.com/rs/zerolog"
)

type Signals struct {
	Labels            []string `yaml:"labels"`
	CommentSubstrings []string `yaml:"comment_substrings"`
	Comments          []string `yaml:"comments"`
	PRBodySubstrings  []string `yaml:"pr_body_substrings"`
	Branches          []string `yaml:"branches"`
	BranchPatterns    []string `yaml:"branch_patterns"`
}

func (s *Signals) Enabled() bool {
	size := 0
	size += len(s.Labels)
	size += len(s.CommentSubstrings)
	size += len(s.Comments)
	size += len(s.PRBodySubstrings)
	size += len(s.Branches)
	size += len(s.BranchPatterns)
	return size > 0
}

// Matches returns true if the pull request meets one or more signals. It also
// returns a description of the signal that was met. The tag argument appears
// in this description and indicates the behavior (trigger, ignore) this
// set of signals is associated with.
func (s *Signals) Matches(ctx context.Context, pullCtx pull.Context, tag string) (bool, string, error) {
	logger := zerolog.Ctx(ctx)

	labels, err := pullCtx.Labels(ctx)
	if err != nil {
		return false, "unable to list pull request labels", err
	}

	if len(labels) == 0 {
		logger.Debug().Msgf("No labels found to match against")
	}
	for _, signalLabel := range s.Labels {
		for _, label := range labels {
			if strings.EqualFold(signalLabel, label) {
				return true, fmt.Sprintf("pull request has a %s label: %q", tag, signalLabel), nil
			}
		}
	}

	body := pullCtx.Body()
	comments, err := pullCtx.Comments(ctx)
	if err != nil {
		return false, "unable to list pull request comments", err
	}

	if len(comments) == 0 {
		logger.Debug().Msgf("No comments found to match against")
	}
	for _, signalComment := range s.Comments {
		if body == signalComment {
			return true, fmt.Sprintf("pull request body is a %s comment: %q", tag, signalComment), nil
		}
		for _, comment := range comments {
			if comment == signalComment {
				return true, fmt.Sprintf("pull request has a %s comment: %q", tag, signalComment), nil
			}
		}
	}

	if len(s.CommentSubstrings) == 0 {
		logger.Debug().Msgf("No comment substrings found to match against")
	}
	for _, signalSubstring := range s.CommentSubstrings {
		if strings.Contains(body, signalSubstring) {
			return true, fmt.Sprintf("pull request body matches a %s substring: %q", tag, signalSubstring), nil
		}
		for _, comment := range comments {
			if strings.Contains(comment, signalSubstring) {
				return true, fmt.Sprintf("pull request comment matches a %s substring: %q", tag, signalSubstring), nil
			}
		}
	}

	if len(s.PRBodySubstrings) == 0 {
		logger.Debug().Msgf("No PR body substrings found to match against")
	}
	for _, signalSubstring := range s.PRBodySubstrings {
		if strings.Contains(body, signalSubstring) {
			return true, fmt.Sprintf("pull request body matches a %s substring: %q", tag, signalSubstring), nil
		}
	}

	targetBranch, _ := pullCtx.Branches()
	if len(s.Branches) == 0 || len(s.BranchPatterns) == 0 {
		logger.Debug().Msgf("No branches or branch patterns found to match against")
	}
	for _, signalBranch := range s.Branches {
		if targetBranch == signalBranch {
			return true, fmt.Sprintf("pull request target is a %s branch: %q", tag, signalBranch), nil
		}
	}
	for _, signalBranch := range s.BranchPatterns {
		if matched, _ := regexp.MatchString(fmt.Sprintf("^%s$", signalBranch), targetBranch); matched {
			return true, fmt.Sprintf("pull request target branch (%q) matches pattern: %q", targetBranch, signalBranch), nil
		}
	}

	return false, fmt.Sprintf("pull request does not match the %s", tag), nil
}
