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
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/palantir/bulldozer/pull"
)

const MaxPullRequestPollCount = 5

func MergePR(ctx context.Context, pullCtx pull.Context, client *github.Client, mergeConfig MergeConfig) error {
	logger := zerolog.Ctx(ctx)

	mergeOpts := &github.PullRequestOptions{}

	switch mergeConfig.Method {
	case SquashAndMerge, MergeCommit, RebaseAndMerge:
		mergeOpts.MergeMethod = string(mergeConfig.Method)
	default:
		mergeOpts.MergeMethod = string(MergeCommit)
	}

	commitMessage := ""
	if mergeConfig.Method == SquashAndMerge {
		opt, ok := mergeConfig.Options[SquashAndMerge]
		if !ok {
			logger.Error().Msgf("Unable to find matching %s in merge option configuration; using default %s", SquashAndMerge, EmptyBody)
			opt = MergeOption{Body: EmptyBody}
		}

		switch opt.Body {
		case PullRequestBody:
			body, err := pullCtx.Body(ctx)
			if err != nil {
				return errors.Wrap(err, "failed to determine pull request body")
			}

			commitMessage = body
			if opt.MessageDelimiter != "" {
				var quotedDelimiter = regexp.QuoteMeta(opt.MessageDelimiter)
				var rString = fmt.Sprintf(`(?sm:(%s\s*)^(.*)$(\s*%s))`, quotedDelimiter, quotedDelimiter)
				if m := regexp.MustCompile(rString).FindStringSubmatch(body); len(m) == 4 {
					commitMessage = m[2]
				}
			}
		case SummarizeCommits:
			summarizedMessages, err := summarizeCommitMessages(ctx, pullCtx, client)
			if err != nil {
				return errors.Wrap(err, "failed to collect pull request commit messages")
			}

			commitMessage = summarizedMessages
		case EmptyBody:
		default:
		}
	}

	go func(ctx context.Context) {
		ticker := time.NewTicker(4 * time.Second)
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

			if pr.Mergeable == nil {
				logger.Debug().Msg("Pull request mergeability not yet known")
				continue
			}

			if !pr.GetMergeable() {
				logger.Debug().Msg("Pull request is not mergeable")
				return
			}

			// Try a merge, a 405 is expected if required reviews are not satisfied
			logger.Info().Msgf("Attempting to merge pull request with method %s", mergeOpts.MergeMethod)
			result, _, err := client.PullRequests.Merge(ctx, pullCtx.Owner(), pullCtx.Repo(), pullCtx.Number(), commitMessage, mergeOpts)
			if err != nil {
				gerr, ok := err.(*github.ErrorResponse)
				if !ok {
					logger.Error().Err(errors.WithStack(err)).Msg("Merge failed unexpectedly")
					continue
				}

				switch gerr.Response.StatusCode {
				case http.StatusMethodNotAllowed:
					logger.Info().Msgf("Merge rejected due to unsatisfied condition %q", gerr.Message)
					return
				case http.StatusConflict:
					logger.Info().Msgf("Merge rejected due to being invalid %q", gerr.Message)
					return
				default:
					logger.Error().Err(errors.WithStack(err)).Msgf("Merge failed unexpectedly %q", gerr.Message)
					continue
				}
			}

			logger.Info().Msgf("Successfully merged pull request for sha %s with message %q", result.GetSHA(), result.GetMessage())

			// Delete ref if owner of BASE and HEAD match
			// otherwise, its from a fork that we cannot delete
			if pr.GetBase().GetUser().GetLogin() == pr.GetHead().GetUser().GetLogin() {
				if mergeConfig.DeleteAfterMerge {
					ref := fmt.Sprintf("refs/heads/%s", pr.Head.GetRef())

					// check other open PRs to make sure that nothing is trying to merge into the ref we're about to delete
					prs, err := pull.ListOpenPullRequestsForRef(ctx, client, pullCtx.Owner(), pullCtx.Repo(), ref)
					if err != nil {
						logger.Error().Err(errors.WithStack(err)).Msgf("Unable to list open prs against ref %s to compare delete request", ref)
						return
					}

					if len(prs) > 0 {
						logger.Info().Msgf("Unable to delete ref %s after merging %q because there are open PRs against this ref", ref, pullCtx.Locator())
						return
					}

					logger.Debug().Msgf("Attempting to delete ref %s", ref)
					_, err = client.Git.DeleteRef(ctx, pullCtx.Owner(), pullCtx.Repo(), ref)
					if err != nil {
						logger.Error().Err(errors.WithStack(err)).Msgf("Failed to delete ref %s on %q", pr.Head.GetRef(), pullCtx.Locator())
						return
					}

					logger.Info().Msgf("Successfully deleted ref %s on %q", pr.Head.GetRef(), pullCtx.Locator())
				}
			} else {
				logger.Debug().Msg("Pull Request is from a fork, not deleting")
			}

			return
		}
	}(zerolog.Ctx(ctx).WithContext(context.Background()))

	return nil
}

func summarizeCommitMessages(ctx context.Context, pullCtx pull.Context, client *github.Client) (string, error) {
	var builder strings.Builder
	repositoryCommits, err := allCommits(ctx, pullCtx, client)
	if err != nil {
		return "", errors.Wrapf(err, "cannot list commits for %q", pullCtx.Locator())
	}

	for _, repositoryCommit := range repositoryCommits {
		fmt.Fprintf(&builder, "* %s\n", repositoryCommit.Commit.GetMessage())
	}

	return builder.String(), nil
}

func allCommits(ctx context.Context, pullCtx pull.Context, client *github.Client) ([]*github.RepositoryCommit, error) {
	var repositoryCommits []*github.RepositoryCommit
	opts := &github.ListOptions{
		PerPage: 100,
	}

	for {
		commits, resp, err := client.PullRequests.ListCommits(ctx, pullCtx.Owner(), pullCtx.Repo(), pullCtx.Number(), opts)
		if err != nil {
			return nil, err
		}
		repositoryCommits = append(repositoryCommits, commits...)
		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return repositoryCommits, nil
}
