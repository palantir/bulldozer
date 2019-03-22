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
	"strings"

	"github.com/palantir/bulldozer/pull"
)

type Signals struct {
	Labels            []string `yaml:"labels"`
	CommentSubstrings []string `yaml:"comment_substrings"`
	Comments          []string `yaml:"comments"`
	PRBodySubstrings  []string `yaml:"pr_body_substrings"`
	Branches          []string `yaml:"branches"`
}

func (s *Signals) Enabled() bool {
	size := 0
	size += len(s.Labels)
	size += len(s.CommentSubstrings)
	size += len(s.Comments)
	size += len(s.PRBodySubstrings)
	size += len(s.Branches)
	return size > 0
}

// Matches returns true if the pull request meets one or more signals. It also
// returns a description of the signal that was met. The tag argument appears
// in this description and indicates the behavior (whitelist, blacklist) this
// set of signals is associated with.
func (s *Signals) Matches(ctx context.Context, pullCtx pull.Context, tag string) (bool, string, error) {
	labels, err := pullCtx.Labels(ctx)
	if err != nil {
		return false, "unable to list pull request labels", err
	}

	for _, signalLabel := range s.Labels {
		for _, label := range labels {
			if strings.EqualFold(signalLabel, label) {
				return true, fmt.Sprintf("pull request has a %s label: %q", tag, signalLabel), nil
			}
		}
	}

	body, err := pullCtx.Body(ctx)
	if err != nil {
		return false, "unable to get pull request body", err
	}
	comments, err := pullCtx.Comments(ctx)
	if err != nil {
		return false, "unable to list pull request comments", err
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

	for _, signalSubstring := range s.PRBodySubstrings {
		if strings.Contains(body, signalSubstring) {
			return true, fmt.Sprintf("pull request body matches a %s substring: %q", tag, signalSubstring), nil
		}
	}

	targetBranch, _, err := pullCtx.Branches(ctx)
	if err != nil {
		return false, "unable to get pull request branches", err
	}

	for _, signalBranch := range s.Branches {
		if targetBranch == signalBranch {
			return true, fmt.Sprintf("pull request target is a %s branch: %q", tag, signalBranch), nil
		}
	}

	return false, fmt.Sprintf("pull request does not match the %s", tag), nil
}
