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

type MergeConfigV1 struct {
	Whitelist Signals `yaml:"whitelist"`
	Blacklist Signals `yaml:"blacklist"`

	DeleteAfterMerge bool `yaml:"delete_after_merge"`

	Method  MergeMethod  `yaml:"method"`
	Options MergeOptions `yaml:"options"`

	BranchMethod map[string]MergeMethod `yaml:"branch_method"`

	// Additional status checks that bulldozer should require
	// (even if the branch protection settings doesn't require it)
	RequiredStatuses []string `yaml:"required_statuses"`
}

type UpdateConfigV1 struct {
	Whitelist Signals `yaml:"whitelist"`
	Blacklist Signals `yaml:"blacklist"`
}

type ConfigV1 struct {
	Version int `yaml:"version"`

	Merge  MergeConfigV1  `yaml:"merge"`
	Update UpdateConfigV1 `yaml:"update"`
}
