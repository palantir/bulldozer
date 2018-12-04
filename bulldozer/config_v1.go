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

type MessageStrategy string
type MergeMethod string

const (
	PullRequestBody  MessageStrategy = "pull_request_body"
	SummarizeCommits MessageStrategy = "summarize_commits"
	EmptyBody        MessageStrategy = "empty_body"

	MergeCommit    MergeMethod = "merge"
	SquashAndMerge MergeMethod = "squash"
	RebaseAndMerge MergeMethod = "rebase"
)

type Signals struct {
	Labels            []string `yaml:"labels"`
	CommentSubstrings []string `yaml:"comment_substrings"`
	Comments          []string `yaml:"comments"`
	PRBodySubstrings  []string `yaml:"pr_body_substrings"`
}

func (s *Signals) Enabled() bool {
	return len(s.Labels)+len(s.CommentSubstrings)+len(s.Comments)+len(s.PRBodySubstrings) > 0
}

type MergeConfig struct {
	Whitelist Signals `yaml:"whitelist"`
	Blacklist Signals `yaml:"blacklist"`

	DeleteAfterMerge bool `yaml:"delete_after_merge"`

	Method  MergeMethod                 `yaml:"method"`
	Options map[MergeMethod]MergeOption `yaml:"options"`

	// Additional status checks that bulldozer should require
	// (even if the branch protection settings doesn't require it)
	RequiredStatuses []string `yaml:"required_statuses"`
}

type MergeOption struct {
	Body             MessageStrategy `yaml:"body"`
	MessageDelimiter string          `yaml:"message_delimiter"`
}

type UpdateConfig struct {
	Whitelist Signals `yaml:"whitelist"`
	Blacklist Signals `yaml:"blacklist"`
}

type Config struct {
	Version int `yaml:"version"`

	Merge  MergeConfig  `yaml:"merge"`
	Update UpdateConfig `yaml:"update"`
}
