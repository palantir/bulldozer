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

package handler

import (
	"context"

	"github.com/google/go-github/v69/github"
	"github.com/palantir/bulldozer/bulldozer"
	"github.com/palantir/go-githubapp/appconfig"
	"github.com/rs/zerolog"
)

type FetchedConfig struct {
	Config     *bulldozer.Config
	LoadError  error
	ParseError error

	Source string
	Path   string
}

type ConfigFetcher struct {
	loader        *appconfig.Loader
	defaultConfig *bulldozer.Config
}

func NewConfigFetcher(loader *appconfig.Loader, defaultConfig *bulldozer.Config) *ConfigFetcher {
	return &ConfigFetcher{
		loader:        loader,
		defaultConfig: defaultConfig,
	}
}

func (cf *ConfigFetcher) Config(ctx context.Context, client *github.Client, owner, repo, ref string) FetchedConfig {
	logger := zerolog.Ctx(ctx)

	c, err := cf.loader.LoadConfig(ctx, client, owner, repo, ref)
	fc := FetchedConfig{
		Source: c.Source,
		Path:   c.Path,
	}

	switch {
	case err != nil:
		fc.LoadError = err
		return fc
	case c.IsUndefined():
		if cf.defaultConfig != nil {
			logger.Debug().Msgf("No repository configuration found, using server default")
			fc.Config = cf.defaultConfig
		}
		return fc
	}

	if config, err := bulldozer.ParseConfig(c.Content); err != nil {
		fc.ParseError = err
	} else {
		fc.Config = config
	}
	return fc
}
