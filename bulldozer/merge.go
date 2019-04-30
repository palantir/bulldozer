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

type Merger interface {
	// Merge merges the pull request in the context using the commit message
	// and options. It returns the SHA of the merge commit on success.
	Merge(ctx context.Context, pullCtx pull.Context, method MergeMethod, msg CommitMessage) (string, error)

	// DeleteHead deletes the head branch of the pull request in the context.
	DeleteHead(ctx context.Context, pullCtx pull.Context) error
}

type CommitMessage struct {
	Title   string
	Message string
}

// GitHubMerger merges pull requests using a GitHub client.
type GitHubMerger struct {
	client *github.Client
}

func NewGitHubMerger(client *github.Client) Merger {
	return &GitHubMerger{
		client: client,
	}
}

func (m *GitHubMerger) Merge(ctx context.Context, pullCtx pull.Context, method MergeMethod, msg CommitMessage) (string, error) {
	opts := github.PullRequestOptions{
		CommitTitle: msg.Title,
		MergeMethod: string(method),
	}

	result, _, err := m.client.PullRequests.Merge(ctx, pullCtx.Owner(), pullCtx.Repo(), pullCtx.Number(), msg.Message, &opts)
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

// PushRestrictionMerger delegates merge operations to different Mergers based
// on whether or not the pull requests targets a branch with push restrictions.
type PushRestrictionMerger struct {
	Normal     Merger
	Restricted Merger
}

func NewPushRestrictionMerger(normal, restricted Merger) Merger {
	return &PushRestrictionMerger{
		Normal:     normal,
		Restricted: restricted,
	}
}

func (m *PushRestrictionMerger) Merge(ctx context.Context, pullCtx pull.Context, method MergeMethod, msg CommitMessage) (string, error) {
	restricted, err := pullCtx.PushRestrictions(ctx)
	if err != nil {
		return "", err
	}

	if restricted {
		zerolog.Ctx(ctx).Info().Msg("Target branch has push restrictions, using restricted client for merge")
		return m.Restricted.Merge(ctx, pullCtx, method, msg)
	}
	return m.Normal.Merge(ctx, pullCtx, method, msg)
}

func (m *PushRestrictionMerger) DeleteHead(ctx context.Context, pullCtx pull.Context) error {
	restricted, err := pullCtx.PushRestrictions(ctx)
	if err != nil {
		return err
	}

	// this is not necessary: the normal client should have delete permissions,
	// but having the merge user also delete the branch is a better UX
	if restricted {
		zerolog.Ctx(ctx).Info().Msg("Target branch has push restrictions, using restricted client for delete")
		return m.Restricted.DeleteHead(ctx, pullCtx)
	}
	return m.Normal.DeleteHead(ctx, pullCtx)
}

// MergePR spawns a goroutine that attempts to merge a pull request. It returns
// an error if an error occurs while preparing for the merge before starting
// the goroutine.
func MergePR(ctx context.Context, pullCtx pull.Context, merger Merger, mergeConfig MergeConfig) error {
	logger := zerolog.Ctx(ctx)

	base, head := pullCtx.Branches()
	mergeMethod := mergeConfig.Method

	if branchMergeMethod, ok := mergeConfig.BranchMethod[base]; ok {
		mergeMethod = branchMergeMethod
	}
	if !isValidMergeMethod(mergeMethod) {
		mergeMethod = MergeCommit
	}

	commitMsg := CommitMessage{}
	if mergeMethod == SquashAndMerge {
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
		commitMsg.Message = message

		title, err := calculateCommitTitle(ctx, pullCtx, *opt)
		if err != nil {
			return errors.Wrap(err, "failed to calculate commit title")
		}
		commitMsg.Title = title
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
			logger.Info().Msgf("Attempting to merge pull request with method %s", mergeMethod)
			sha, err := merger.Merge(ctx, pullCtx, mergeMethod, commitMsg)
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
				ref := fmt.Sprintf("refs/heads/%s", head)
				if mergeConfig.DeleteAfterMerge {
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

					logger.Info().Msgf("Attempting to delete ref %s", ref)
					if err := merger.DeleteHead(ctx, pullCtx); err != nil {
						logger.Error().Err(err).Msgf("Failed to delete ref %s on %q", ref, pullCtx.Locator())
						return
					}

					logger.Info().Msgf("Successfully deleted ref %s on %q", ref, pullCtx.Locator())
				} else {
					logger.Debug().Msgf("Not deleting ref %s, delete_after_merge is not enabled", ref)
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
