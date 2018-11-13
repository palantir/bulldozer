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
	"time"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/palantir/bulldozer/pull"
)

func ShouldUpdatePR(ctx context.Context, pullCtx pull.Context, updateConfig UpdateConfig) (bool, error) {
	logger := zerolog.Ctx(ctx)

	blacklistable := updateConfig.Blacklist.Enabled()
	blacklisted, blacklistedReason, err := IsPRBlacklisted(ctx, pullCtx, updateConfig.Blacklist)
	if err != nil {
		return false, errors.Wrap(err, "failed to determine if pull request is blacklisted")
	}

	whitelistable := updateConfig.Whitelist.Enabled()
	whitelisted, whitelistedReason, err := IsPRWhitelisted(ctx, pullCtx, updateConfig.Whitelist)
	if err != nil {
		return false, errors.Wrap(err, "failed to determine if pull request is whitelisted")
	}

	logger.Debug().Msgf("determine should update PR: blacklistable=%t blacklisted=%t whitelistable=%t whitelisted=%t blacklistReason='%s' whitelistReason='%s'",
		blacklistable, blacklisted, whitelistable, whitelisted, blacklistedReason, whitelistedReason)

	if blacklistable && !blacklisted && whitelisted {
		return true, nil
	}

	if blacklistable && !blacklisted && !whitelistable {
		return true, nil
	}

	if !blacklistable && whitelisted {
		return true, nil
	}

	return false, nil
}

func UpdatePR(ctx context.Context, pullCtx pull.Context, client *github.Client, updateConfig UpdateConfig, baseRef string) error {
	logger := zerolog.Ctx(ctx)

	//todo: should the updateConfig struct provide any other details here?

	go func(ctx context.Context, baseRef string) {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for i := 0; i < MaxPullRequestPollCount; i++ {
			<-ticker.C

			pr, _, err := client.PullRequests.Get(ctx, pullCtx.Owner(), pullCtx.Repo(), pullCtx.Number())
			if err != nil {
				logger.Error().Err(errors.WithStack(err)).Msgf("Failed to retrieve pull request %q", pullCtx.Locator())
				return
			}

			if pr.GetState() == "closed" {
				logger.Debug().Msg("Pull request already closed")
				return
			}

			if pr.Head.Repo.GetFork() {
				logger.Debug().Msg("Pull request is from a fork, cannot keep it up to date with base ref")
				return
			}

			comparison, _, err := client.Repositories.CompareCommits(ctx, pullCtx.Owner(), pullCtx.Repo(), baseRef, pr.GetHead().GetSHA())
			if err != nil {
				logger.Error().Err(errors.WithStack(err)).Msgf("cannot compare %s and %s for %q", baseRef, pr.GetHead().GetSHA(), pullCtx.Locator())
			}
			if comparison.GetBehindBy() > 0 {
				logger.Debug().Msg("Pull request is not up to date")

				mergeRequest := &github.RepositoryMergeRequest{
					Base: github.String(pr.Head.GetRef()),
					Head: github.String(baseRef),
				}

				mergeCommit, _, err := client.Repositories.Merge(ctx, pullCtx.Owner(), pullCtx.Repo(), mergeRequest)
				if err != nil {
					logger.Error().Err(errors.WithStack(err)).Msg("Merge failed unexpectedly")
				}

				logger.Info().Msgf("Successfully updated pull request from base ref %s as merge %s", baseRef, mergeCommit.GetSHA())
			} else {
				logger.Debug().Msg("Pull request is not out of date, not updating")
			}

			return
		}
	}(zerolog.Ctx(ctx).WithContext(context.Background()), baseRef)

	return nil
}
