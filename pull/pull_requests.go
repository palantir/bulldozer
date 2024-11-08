// Copyright 2024 Palantir Technologies, Inc.
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

package pull

import (
	"context"
	"strings"

	"github.com/google/go-github/v66/github"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// GitHubClient is an interface that wraps the methods used from the github.Client.
type GitHubClient interface {
	ListPullRequestsWithCommit(ctx context.Context, owner, repo, sha string, opts *github.ListOptions) ([]*github.PullRequest, *github.Response, error)
	List(ctx context.Context, owner, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
}

// GetOpenPullRequestsForSHA returns all open pull requests where the HEAD of the source branch
// matches the given SHA.
func GetOpenPullRequestsForSHA(ctx context.Context, client GitHubClient, owner, repo, sha string) ([]*github.PullRequest, error) {
	logger := zerolog.Ctx(ctx)
	var results []*github.PullRequest
	opts := &github.ListOptions{PerPage: 100}

	for {
		prs, resp, err := client.ListPullRequestsWithCommit(ctx, owner, repo, sha, opts)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list pull requests for repository %s/%s", owner, repo)
		}

		for _, pr := range prs {
			if pr.GetState() == "open" && pr.GetHead().GetSHA() == sha {
				logger.Debug().Msgf("found open pull request with sha %s", pr.GetHead().GetSHA())
				results = append(results, pr)
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return results, nil
}

// ListOpenPullRequestsForSHA returns all open pull requests where the HEAD of the source branch
// matches the given SHA by fetching all open PRs and filtering.
func ListOpenPullRequestsForSHA(ctx context.Context, client GitHubClient, owner, repo, sha string) ([]*github.PullRequest, error) {
	logger := zerolog.Ctx(ctx)
	var results []*github.PullRequest
	opts := &github.PullRequestListOptions{
		State:       "open",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		prs, resp, err := client.List(ctx, owner, repo, opts)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list pull requests for repository %s/%s", owner, repo)
		}

		for _, pr := range prs {
			if pr.Head.GetSHA() == sha {
				logger.Debug().Msgf("found open pull request with sha %s", pr.Head.GetSHA())
				results = append(results, pr)
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return results, nil
}

// GetAllPossibleOpenPullRequestsForSHA attempts to find all open pull requests
// associated with the given SHA using multiple methods in case we are dealing with a fork
func GetAllPossibleOpenPullRequestsForSHA(ctx context.Context, client GitHubClient, owner, repo, sha string) ([]*github.PullRequest, error) {
	logger := zerolog.Ctx(ctx)

	prs, err := GetOpenPullRequestsForSHA(ctx, client, owner, repo, sha)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get open pull requests matching the SHA")
	}

	if len(prs) == 0 {
		logger.Debug().Msg("No pull requests associated with the check run, searching by SHA")
		prs, err = ListOpenPullRequestsForSHA(ctx, client, owner, repo, sha)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list open pull requests matching the SHA")
		}
		if len(prs) == 0 {
			logger.Debug().Msg("No open pull requests found for the given SHA")
			return nil, nil
		}
	}

	return prs, nil
}

// ListOpenPullRequestsForRef returns all open pull requests for a given base branch reference.
func ListOpenPullRequestsForRef(ctx context.Context, client GitHubClient, owner, repo, ref string) ([]*github.PullRequest, error) {
	logger := zerolog.Ctx(ctx)
	ref = strings.TrimPrefix(ref, "refs/heads/")
	opts := &github.PullRequestListOptions{
		State:       "open",
		Base:        ref,
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var results []*github.PullRequest
	for {
		prs, resp, err := client.List(ctx, owner, repo, opts)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list pull requests for repository %s/%s", owner, repo)
		}

		for _, pr := range prs {
			logger.Debug().Msgf("found open pull request with base ref %s", pr.GetBase().GetRef())
			results = append(results, pr)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return results, nil
}
