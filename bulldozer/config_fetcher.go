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
	"net/http"

	"github.com/google/go-github/v32/github"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v2"
)

type FetchedConfig struct {
	Owner  string
	Repo   string
	Ref    string
	Config *Config
	Error  error
}

func (fc FetchedConfig) Missing() bool {
	return fc.Config == nil && fc.Error == nil
}

func (fc FetchedConfig) Valid() bool {
	return fc.Config != nil && fc.Error == nil
}

func (fc FetchedConfig) Invalid() bool {
	return fc.Error != nil
}

func (fc FetchedConfig) String() string {
	return fmt.Sprintf("%s/%s ref=%s", fc.Owner, fc.Repo, fc.Ref)
}

type ConfigFetcher struct {
	configurationV1Path     string
	configurationV0Paths    []string
	defaultRepositoryConfig *Config
}

func NewConfigFetcher(configurationV1Path string, configurationV0Paths []string, defaultRepositoryConfig *Config) ConfigFetcher {
	return ConfigFetcher{
		configurationV1Path:     configurationV1Path,
		configurationV0Paths:    configurationV0Paths,
		defaultRepositoryConfig: defaultRepositoryConfig,
	}
}

// ConfigForPR fetches the configuration for a PR. It returns an error
// only if the existence of the configuration file could not be determined. If the file
// does not exist or is invalid, the returned error is nil and the appropriate
// fields are set on the FetchedConfig.
func (cf *ConfigFetcher) ConfigForPR(ctx context.Context, client *github.Client, pr *github.PullRequest) (FetchedConfig, error) {
	fc := FetchedConfig{
		Owner: pr.GetBase().GetRepo().GetOwner().GetLogin(),
		Repo:  pr.GetBase().GetRepo().GetName(),
		Ref:   pr.GetBase().GetRef(),
	}

	logger := zerolog.Ctx(ctx)

	bytes, err := cf.fetchConfigContents(ctx, client, fc.Owner, fc.Repo, fc.Ref, cf.configurationV1Path)
	if err == nil && bytes != nil {
		if config, err := cf.unmarshalConfig(bytes); err == nil {
			logger.Debug().Msgf("Found v1 configuration at %s", cf.configurationV1Path)
			fc.Config = config
			return fc, nil
		}
	}
	logger.Debug().Msgf("v1 configuration was missing or invalid, falling back to v0 or server configuration")

	for _, configV0Path := range cf.configurationV0Paths {
		bytes, err := cf.fetchConfigContents(ctx, client, fc.Owner, fc.Repo, fc.Ref, configV0Path)
		if err != nil || bytes == nil {
			continue
		}

		config, err := cf.unmarshalConfigV0(bytes)
		if err != nil {
			continue
		}

		logger.Debug().Msgf("Found v0 configuration at %s", configV0Path)
		fc.Config = config
		return fc, nil
	}

	if cf.defaultRepositoryConfig != nil {
		logger.Debug().Msgf("No repository configuration found, using server-provided default")
		fc.Config = cf.defaultRepositoryConfig
		return fc, nil
	}

	fc.Error = errors.New("No configuration found")
	return fc, nil
}

// fetchConfigContents returns a nil slice if there is no configuration file
func (cf *ConfigFetcher) fetchConfigContents(ctx context.Context, client *github.Client, owner, repo, ref, configPath string) ([]byte, error) {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Str("path", configPath).Str("ref", ref).Msg("Attempting to fetch configuration definition")

	opts := &github.RepositoryContentGetOptions{
		Ref: ref,
	}

	file, _, _, err := client.Repositories.GetContents(ctx, owner, repo, configPath, opts)
	if err != nil {
		if rerr, ok := err.(*github.ErrorResponse); ok && rerr.Response.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to fetch content of %q", configPath)
	}

	// file will be nil if the ref contains a directory at the expected file path
	if file == nil {
		return nil, nil
	}

	content, err := file.GetContent()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode content of %q", configPath)
	}

	return []byte(content), nil
}

func (cf *ConfigFetcher) unmarshalConfig(bytes []byte) (*Config, error) {
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
		return nil, errors.Errorf("unexpected version '%d', expected 1", config.Version)
	}

	return &config, nil
}

func (cf *ConfigFetcher) unmarshalConfigV0(bytes []byte) (*Config, error) {
	var configv0 ConfigV0
	var config Config
	if err := yaml.UnmarshalStrict(bytes, &configv0); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal v0 configuration")
	}

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
				DeleteAfterMerge: configv0.DeleteAfterMerge,
				Method:           configv0.Strategy,
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
				DeleteAfterMerge: configv0.DeleteAfterMerge,
				Method:           configv0.Strategy,
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
				DeleteAfterMerge: configv0.DeleteAfterMerge,
				Method:           configv0.Strategy,
			},
		}
		if config.Merge.Method == SquashAndMerge {
			config.Merge.Options.Squash = &SquashOptions{
				Body:             PullRequestBody,
				MessageDelimiter: "==COMMIT_MSG==",
			}
		}
	default:
	}

	return &config, nil
}
