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

type Merger interface {
	// Merge merges the pull request in the context using the commit message
	// and options. It returns the SHA of the merge commit on success.
	Merge(ctx context.Context, pullCtx pull.Context, message string, options *github.PullRequestOptions) (string, error)

	// DeleteHead deletes the head branch of the pull request in the context.
	DeleteHead(ctx context.Context, pullCtx pull.Context) error
}

type GitHubMerger struct {
	client *github.Client
}

func NewGitHubMerger(client *github.Client) Merger {
	return &GitHubMerger{
		client: client,
	}
}

func (m *GitHubMerger) Merge(ctx context.Context, pullCtx pull.Context, message string, options *github.PullRequestOptions) (string, error) {
	result, _, err := m.client.PullRequests.Merge(ctx, pullCtx.Owner(), pullCtx.Repo(), pullCtx.Number(), message, options)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return result.GetSHA(), nil
}

func (m *GitHubMerger) DeleteHead(ctx context.Context, pullCtx pull.Context) error {
	_, head := pullCtx.Branches()
	_, err := m.client.Git.DeleteRef(ctx, pullCtx.Owner(), pullCtx.Repo(), fmt.Sprintf("refs/heads/%s", head))
	return errors.WithStack(err)
}

const MaxPullRequestPollCount = 5

func MergePR(ctx context.Context, pullCtx pull.Context, merger Merger, mergeConfig MergeConfig) error {
	logger := zerolog.Ctx(ctx)

	mergeOpts := &github.PullRequestOptions{}

	base, head := pullCtx.Branches()
	mergeMethod := mergeConfig.Method

	if branchMergeMethod, ok := mergeConfig.BranchMethod[base]; ok {
		mergeMethod = branchMergeMethod
	}

	if !isValidMergeMethod(mergeMethod) {
		mergeMethod = MergeCommit
	}

	mergeOpts.MergeMethod = string(mergeMethod)

	commitMessage := ""
	if mergeConfig.Method == SquashAndMerge {
		opt := mergeConfig.Options.Squash
		if opt == nil {
			logger.Info().Msgf("No squash options defined; using defaults")
			opt = &SquashOptions{}
		}

		if opt.Title == "" {
			opt.Title = PullRequestTitle
		}
		if opt.Body == "" {
			opt.Body = EmptyBody
		}

		message, err := calculateCommitMessage(ctx, pullCtx, *opt)
		if err != nil {
			return errors.Wrap(err, "failed to calculate commit message")
		}
		commitMessage = message

		title, err := calculateCommitTitle(ctx, pullCtx, *opt)
		if err != nil {
			return errors.Wrap(err, "failed to calculate commit title")
		}
		mergeOpts.CommitTitle = title
	}

	go func(ctx context.Context) {
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()

		for i := 0; i < MaxPullRequestPollCount; i++ {
			<-ticker.C

			mergeState, err := pullCtx.MergeState(ctx)
			if err != nil {
				logger.Error().Err(err).Msgf("Failed to get merge state for %q", pullCtx.Locator())
				return
			}

			if mergeState.Closed {
				logger.Debug().Msg("Pull request already closed")
				return
			}

			if mergeState.Mergeable == nil {
				logger.Debug().Msg("Pull request mergeability not yet known")
				continue
			}

			if !*mergeState.Mergeable {
				logger.Debug().Msg("Pull request is not mergeable")
				return
			}

			// Try a merge, a 405 is expected if required reviews are not satisfied
			logger.Info().Msgf("Attempting to merge pull request with method %s", mergeOpts.MergeMethod)
			sha, err := merger.Merge(ctx, pullCtx, commitMessage, mergeOpts)
			if err != nil {
				gerr, ok := errors.Cause(err).(*github.ErrorResponse)
				if !ok {
					logger.Error().Err(err).Msg("Merge failed unexpectedly")
					continue
				}

				switch gerr.Response.StatusCode {
				case http.StatusMethodNotAllowed:
					logger.Info().Msgf("Merge rejected due to unsatisfied condition: %q", gerr.Message)
					return
				case http.StatusConflict:
					logger.Info().Msgf("Merge rejected due to being invalid: %q", gerr.Message)
					return
				default:
					logger.Error().Msgf("Merge failed with unexpected status: %d: %q", gerr.Response.StatusCode, gerr.Message)
					continue
				}
			}

			logger.Info().Msgf("Successfully merged pull request as SHA %s", sha)

			// if head is qualified (contains ":"), PR is from a fork and we don't have delete permission
			if !strings.ContainsRune(head, ':') {
				if mergeConfig.DeleteAfterMerge {
					ref := fmt.Sprintf("refs/heads/%s", head)

					// check other open PRs to make sure that nothing is trying to merge into the ref we're about to delete
					isTargeted, err := pullCtx.IsTargeted(ctx)
					if err != nil {
						logger.Error().Err(err).Msgf("Unable to determine if ref %s is targeted by other open pull requests before deletion", ref)
						return
					}
					if isTargeted {
						logger.Info().Msgf("Unable to delete ref %s after merging %q because there are open PRs against this ref", ref, pullCtx.Locator())
						return
					}

					logger.Debug().Msgf("Attempting to delete ref %s", ref)
					if err := merger.DeleteHead(ctx, pullCtx); err != nil {
						logger.Error().Err(err).Msgf("Failed to delete ref %s on %q", ref, pullCtx.Locator())
						return
					}

					logger.Info().Msgf("Successfully deleted ref %s on %q", ref, pullCtx.Locator())
				}
			} else {
				logger.Debug().Msg("Pull Request is from a fork, not deleting")
			}

			return
		}
	}(zerolog.Ctx(ctx).WithContext(context.Background()))

	return nil
}

func isValidMergeMethod(input MergeMethod) bool {
	return input == SquashAndMerge || input == RebaseAndMerge || input == MergeCommit
}

func calculateCommitMessage(ctx context.Context, pullCtx pull.Context, option SquashOptions) (string, error) {
	commitMessage := ""
	switch option.Body {
	case PullRequestBody:
		commitMessage = pullCtx.Body()
		if option.MessageDelimiter != "" {
			var quotedDelimiter = regexp.QuoteMeta(option.MessageDelimiter)
			var rString = fmt.Sprintf(`(?sm:(%s\s*)^(.*)$(\s*%s))`, quotedDelimiter, quotedDelimiter)
			matcher, err := regexp.Compile(rString)
			if err != nil {
				return "", errors.Wrap(err, "failed to compile message delimiter regex")
			}

			if m := matcher.FindStringSubmatch(commitMessage); len(m) == 4 {
				commitMessage = m[2]
			}
		}
	case SummarizeCommits:
		summarizedMessages, err := summarizeCommitMessages(ctx, pullCtx)
		if err != nil {
			return "", errors.Wrap(err, "failed to summarize pull request commit messages")
		}
		commitMessage = summarizedMessages
	case EmptyBody:
	}

	return commitMessage, nil
}

func calculateCommitTitle(ctx context.Context, pullCtx pull.Context, option SquashOptions) (string, error) {
	var title string
	switch option.Title {
	case PullRequestTitle:
		title = pullCtx.Title()
	case FirstCommitTitle:
		commits, err := pullCtx.Commits(ctx)
		if err != nil {
			return "", err
		}
		// commits are ordered from oldest to newest, must have at least one to make a PR
		title = strings.SplitN(commits[0].Message, "\n", 2)[0]
	case GithubDefaultTitle:
	}

	if title != "" {
		title = fmt.Sprintf("%s (#%d)", title, pullCtx.Number())
	}
	return title, nil
}

func summarizeCommitMessages(ctx context.Context, pullCtx pull.Context) (string, error) {
	commits, err := pullCtx.Commits(ctx)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	for _, c := range commits {
		fmt.Fprintf(&builder, "* %s\n", c.Message)
	}
	return builder.String(), nil
}
