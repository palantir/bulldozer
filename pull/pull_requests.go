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

package pull

import (
	"context"
	"strings"

	"github.com/google/go-github/v66/github"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// ListOpenPullRequestsForSHA returns all pull requests where the HEAD of the source branch
// in the pull request matches the given SHA.
func ListOpenPullRequestsForSHA(ctx context.Context, client *github.Client, owner, repoName, SHA string) ([]*github.PullRequest, error) {
	var results []*github.PullRequest
	logger := zerolog.Ctx(ctx)

	opts := &github.PullRequestListOptions{
		State: "open",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		prs, resp, err := client.PullRequests.List(ctx, owner, repoName, opts)
		if err != nil {
			return results, errors.Wrapf(err, "failed to list pull requests for repository %s/%s", owner, repoName)
		}
		for _, pr := range prs {
			if pr.Head.GetSHA() == SHA {
				logger.Debug().Msgf("found open pull request with sha %s", pr.Head.GetSHA())
				results = append(results, pr)
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.ListOptions.Page = resp.NextPage
	}

	return results, nil
}

func ListOpenPullRequestsForRef(ctx context.Context, client *github.Client, owner, repoName, ref string) ([]*github.PullRequest, error) {
	var results []*github.PullRequest
	logger := zerolog.Ctx(ctx)

	ref = strings.TrimPrefix(ref, "refs/heads/")

	opts := &github.PullRequestListOptions{
		State: "open",
		Base:  ref, // Filter by base branch name
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		prs, resp, err := client.PullRequests.List(ctx, owner, repoName, opts)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list pull requests for repository %s/%s", owner, repoName)
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
