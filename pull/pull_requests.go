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
	"fmt"

	"github.com/google/go-github/v65/github"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// ListOpenPullRequestsForSHA returns all pull requests where the HEAD of the source branch
// in the pull request matches the given SHA.
func ListOpenPullRequestsForSHA(ctx context.Context, client *github.Client, owner, repoName, SHA string) ([]*github.PullRequest, error) {
	var results []*github.PullRequest

	openPRs, err := ListOpenPullRequests(ctx, client, owner, repoName)

	if err != nil {
		return nil, err
	}

	for _, openPR := range openPRs {
		if openPR.Head.GetSHA() == SHA {
			results = append(results, openPR)
		}
	}

	return results, nil
}

func ListOpenPullRequestsForRef(ctx context.Context, client *github.Client, owner, repoName, ref string) ([]*github.PullRequest, error) {
	var results []*github.PullRequest
	logger := zerolog.Ctx(ctx)

	openPRs, err := ListOpenPullRequests(ctx, client, owner, repoName)

	if err != nil {
		return nil, err
	}

	for _, openPR := range openPRs {
		formattedRef := fmt.Sprintf("refs/heads/%s", openPR.GetBase().GetRef())
		logger.Debug().Msgf("found open pull request with base ref %s", formattedRef)
		if formattedRef == ref {
			results = append(results, openPR)
		}
	}

	return results, nil
}

func ListOpenPullRequests(ctx context.Context, client *github.Client, owner, repoName string) ([]*github.PullRequest, error) {
	var results []*github.PullRequest

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
			results = append(results, pr)
		}
		if resp.NextPage == 0 {
			break
		}
		opts.ListOptions.Page = resp.NextPage
	}

	return results, nil
}
