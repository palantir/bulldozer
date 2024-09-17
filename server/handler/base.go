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

	"github.com/google/go-github/v65/github"
	"github.com/palantir/bulldozer/bulldozer"
	"github.com/palantir/bulldozer/pull"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type Base struct {
	githubapp.ClientCreator

	ConfigFetcher            *ConfigFetcher
	PushRestrictionUserToken string
	DisableUpdateFeature     bool
}

func (b *Base) FetchConfigForPR(ctx context.Context, client *github.Client, pr *github.PullRequest) (*bulldozer.Config, error) {
	owner := pr.GetBase().GetRepo().GetOwner().GetLogin()
	repo := pr.GetBase().GetRepo().GetName()
	ref := pr.GetBase().GetRef()
	return b.FetchConfig(ctx, client, owner, repo, ref)
}

func (b *Base) FetchConfig(ctx context.Context, client *github.Client, owner, repo, ref string) (*bulldozer.Config, error) {
	logger := zerolog.Ctx(ctx)

	fc := b.ConfigFetcher.Config(ctx, client, owner, repo, ref)
	switch {
	case fc.LoadError != nil:
		return nil, errors.Wrapf(fc.LoadError, "failed to load configuration: %s: %s", fc.Source, fc.Path)

	case fc.ParseError != nil:
		logger.Warn().Msgf("Invalid configuration in %s: %s", fc.Source, fc.Path)
		return nil, nil

	case fc.Config == nil:
		logger.Debug().Msg("No configuration defined for repository")
		return nil, nil
	}

	return fc.Config, nil
}

func (b *Base) ProcessPullRequest(ctx context.Context, pullCtx pull.Context, client *github.Client, config *bulldozer.Config, pr *github.PullRequest) error {
	logger := zerolog.Ctx(ctx)

	if config == nil {
		logger.Debug().Msg("ProcessPullRequest: returning immediately due to nil config")
		return nil
	}

	merger := bulldozer.NewGitHubMerger(client)
	if b.PushRestrictionUserToken != "" {
		tokenClient, err := b.NewTokenClient(b.PushRestrictionUserToken)
		if err != nil {
			return errors.Wrap(err, "failed to create token client")
		}
		merger = bulldozer.NewPushRestrictionMerger(merger, bulldozer.NewGitHubMerger(tokenClient))
	}

	shouldMerge, err := bulldozer.ShouldMergePR(ctx, pullCtx, config.Merge)
	if err != nil {
		return errors.Wrap(err, "unable to determine merge status")
	}
	if shouldMerge {
		bulldozer.MergePR(ctx, pullCtx, merger, config.Merge)
	}

	return nil
}

func (b *Base) UpdatePullRequest(ctx context.Context, pullCtx pull.Context, client *github.Client, config *bulldozer.Config, pr *github.PullRequest, baseRef string) (bool, error) {
	logger := zerolog.Ctx(ctx)

	if config == nil {
		logger.Debug().Msg("UpdatePullRequest: returning immediately due to nil config")
		return false, nil
	}

	shouldUpdate, err := bulldozer.ShouldUpdatePR(ctx, pullCtx, config.Update)
	if err != nil {
		return false, errors.Wrap(err, "unable to determine update status")
	}

	didUpdatePR := false

	if shouldUpdate {
		didUpdatePR = bulldozer.UpdatePR(ctx, pullCtx, client, config.Update, baseRef)
	}

	return didUpdatePR, nil
}
