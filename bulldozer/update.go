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

	"github.com/google/go-github/v67/github"
	"github.com/palantir/bulldozer/pull"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

func UpdatePR(ctx context.Context, pullCtx pull.Context, client *github.Client, updateConfig UpdateConfig, baseRef string) bool {
	logger := zerolog.Ctx(ctx)

	pr, _, err := client.PullRequests.Get(ctx, pullCtx.Owner(), pullCtx.Repo(), pullCtx.Number())
	if err != nil {
		logger.Error().Err(errors.WithStack(err)).Msgf("Failed to retrieve pull request %q", pullCtx.Locator())
		return false
	}

	if pr.GetState() == "closed" {
		logger.Debug().Msg("Pull request already closed")
		return false
	}

	if pr.Head.Repo.GetFork() {
		logger.Debug().Msg("Pull request is from a fork, cannot keep it up to date with base ref")
		return false
	}

	comparison, _, err := client.Repositories.CompareCommits(ctx, pullCtx.Owner(), pullCtx.Repo(), baseRef, pr.GetHead().GetSHA(), nil)
	if err != nil {
		logger.Error().Err(errors.WithStack(err)).Msgf("Cannot compare %s and %s for %q", baseRef, pr.GetHead().GetSHA(), pullCtx.Locator())
		return false
	}
	if comparison.GetBehindBy() == 0 {
		logger.Debug().Msg("Pull request is not out of date, not updating")
		return false
	}

	logger.Debug().Msg("Pull request is not up to date, attempting an update")
	mergeCommit, _, err := client.Repositories.Merge(ctx, pullCtx.Owner(), pullCtx.Repo(), &github.RepositoryMergeRequest{
		Base: github.String(pr.Head.GetRef()),
		Head: github.String(baseRef),
	})
	if err != nil {
		logger.Error().Err(errors.WithStack(err)).Msg("Update merge failed unexpectedly")
		return false
	}
	logger.Info().Msgf("Successfully updated pull request from base ref %s as merge %s", baseRef, mergeCommit.GetSHA())
	return true
}
