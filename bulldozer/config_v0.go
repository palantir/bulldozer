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

type ModeV0 string

const (
	ModeWhitelistV0 ModeV0 = "whitelist"
	ModeBlacklistV0 ModeV0 = "blacklist"
	ModeBodyV0      ModeV0 = "pr_body"
)

type ConfigV0 struct {
	Mode                   ModeV0      `yaml:"mode"`
	Strategy               MergeMethod `yaml:"strategy"`
	DeleteAfterMerge       bool        `yaml:"deleteAfterMerge"`
	AllowMergeWithNoChecks bool        `yaml:"allow_merge_with_no_checks"`

	// this setting is unused, but needs to be present for valid v0 configuration
	IgnoreSquashedMessages bool `yaml:"ignoreSquashedMessages"`
}
