// Copyright 2021 Palantir Technologies, Inc.
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
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func ParseConfig(c []byte) (*Config, error) {
	config, v1err := parseConfigV1(c)
	if v1err == nil {
		return config, nil
	}

	config, v0err := parseConfigV0(c)
	if v0err == nil {
		return config, nil
	}

	// Encourage v1 usage by reporting the v1 parsing error in all cases
	return nil, v1err
}

func parseConfigV1(bytes []byte) (*Config, error) {
	var config Config
	if err := yaml.UnmarshalStrict(bytes, &config); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal configuration")
	}

	// Merge old signals configurations if they exist when the new values aren't present
	if config.Merge.Blacklist.Enabled() && !config.Merge.Ignore.Enabled() {
		config.Merge.Ignore = config.Merge.Blacklist
	}
	if config.Merge.Whitelist.Enabled() && !config.Merge.Trigger.Enabled() {
		config.Merge.Trigger = config.Merge.Whitelist
	}
	if config.Update.Blacklist.Enabled() && !config.Update.Ignore.Enabled() {
		config.Update.Ignore = config.Update.Blacklist
	}
	if config.Update.Whitelist.Enabled() && !config.Update.Trigger.Enabled() {
		config.Update.Trigger = config.Update.Whitelist
	}

	if config.Version != 1 {
		return nil, errors.Errorf("unexpected version %d, expected 1", config.Version)
	}

	return &config, nil
}

func parseConfigV0(bytes []byte) (*Config, error) {
	var configv0 ConfigV0
	if err := yaml.UnmarshalStrict(bytes, &configv0); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal v0 configuration")
	}

	var config Config
	switch configv0.Mode {
	case ModeWhitelistV0:
		config = Config{
			Version: 1,
			Update: UpdateConfig{
				Trigger: Signals{
					Labels: []string{"update me", "update-me", "update_me"},
				},
			},
			Merge: MergeConfig{
				Trigger: Signals{
					Labels: []string{"merge when ready", "merge-when-ready", "merge_when_ready"},
				},
				DeleteAfterMerge:       configv0.DeleteAfterMerge,
				AllowMergeWithNoChecks: false,
				Method:                 configv0.Strategy,
			},
		}
		if config.Merge.Method == SquashAndMerge {
			config.Merge.Options.Squash = &SquashOptions{
				Body: SummarizeCommits,
			}
		}
	case ModeBlacklistV0:
		config = Config{
			Version: 1,
			Update: UpdateConfig{
				Trigger: Signals{
					Labels: []string{"update me", "update-me", "update_me"},
				},
			},
			Merge: MergeConfig{
				Ignore: Signals{
					Labels: []string{"wip", "do not merge", "do-not-merge", "do_not_merge"},
				},
				DeleteAfterMerge:       configv0.DeleteAfterMerge,
				AllowMergeWithNoChecks: false,
				Method:                 configv0.Strategy,
			},
		}
		if config.Merge.Method == SquashAndMerge {
			config.Merge.Options.Squash = &SquashOptions{
				Body: SummarizeCommits,
			}
		}
	case ModeBodyV0:
		config = Config{
			Version: 1,
			Update: UpdateConfig{
				Trigger: Signals{
					Labels: []string{"update me", "update-me", "update_me"},
				},
			},
			Merge: MergeConfig{
				Trigger: Signals{
					CommentSubstrings: []string{"==MERGE_WHEN_READY=="},
				},
				DeleteAfterMerge:       configv0.DeleteAfterMerge,
				AllowMergeWithNoChecks: false,
				Method:                 configv0.Strategy,
			},
		}
		if config.Merge.Method == SquashAndMerge {
			config.Merge.Options.Squash = &SquashOptions{
				Body:             PullRequestBody,
				MessageDelimiter: "==COMMIT_MSG==",
			}
		}
	default:
		return nil, errors.Errorf("unknown v0 mode: %q", configv0.Mode)
	}

	return &config, nil
}
