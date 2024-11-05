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

	"github.com/google/go-github/v66/github"
	"github.com/palantir/bulldozer/pull"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
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
	if method == FastForwardOnly {
		return m.ffOnlyMerge(ctx, pullCtx)
	}
	return m.defaultMerge(ctx, pullCtx, method, msg)
}

// ff-only merge is accomplished by calling Git.UpdateRef with the force
// parameter set to false, and the new commit hash for the base branch's
// pointer
func (m *GitHubMerger) ffOnlyMerge(ctx context.Context, pullCtx pull.Context) (string, error) {
	base, _ := pullCtx.Branches()

	ref, _, err := m.client.Git.GetRef(ctx, pullCtx.Owner(), pullCtx.Repo(), fmt.Sprintf("refs/heads/%s", base))
	if err != nil {
		return "", errors.Wrap(err, "could not get git reference of PR base branch")
	}

	headCommitSHA := pullCtx.HeadSHA()
	ref.Object.SHA = &headCommitSHA

	newRef, _, err := m.client.Git.UpdateRef(ctx, pullCtx.Owner(), pullCtx.Repo(), ref, false)
	if err != nil {
		return "", errors.Wrap(err, "could not perform ff-only merge")
	}

	if newRef.GetObject().GetSHA() != headCommitSHA {
		return "", fmt.Errorf("expected reference to be updated to SHA %s, but instead it points to %s", headCommitSHA, newRef.GetObject().GetSHA())
	}

	return headCommitSHA, nil
}

func (m *GitHubMerger) defaultMerge(ctx context.Context, pullCtx pull.Context, method MergeMethod, msg CommitMessage) (string, error) {
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
	resp, err := m.client.Git.DeleteRef(ctx, pullCtx.Owner(), pullCtx.Repo(), fmt.Sprintf("refs/heads/%s", head))
	// Disregard the error since Github may delete the branch before we do
	if resp.StatusCode == http.StatusUnprocessableEntity {
		return nil
	}
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

// DetermineMergeMethod determines which merge method to use when merging the PR
func DetermineMergeMethod(ctx context.Context, pullCtx pull.Context, mergeConfig MergeConfig) (MergeMethod, error) {
	logger := zerolog.Ctx(ctx)

	base, _ := pullCtx.Branches()
	mergeMethod := mergeConfig.Method

	if branchMergeMethod, ok := mergeConfig.BranchMethod[base]; ok {
		mergeMethod = branchMergeMethod
	}

	for _, method := range mergeConfig.MergeMethods {
		triggered, reason, err := IsMergeMethodTriggered(ctx, pullCtx, method.Trigger)
		if err != nil {
			err = errors.Wrapf(err, "Failed to determine if merge method '%s' is triggered", method.Method)
			return "", err
		}

		if triggered {
			mergeMethod = method.Method
			logger.Debug().Msgf("%s method is triggered because %s matched", mergeMethod, reason)
			break
		}
	}

	if !isValidMergeMethod(mergeMethod) {
		mergeMethod = MergeCommit
	}

	return mergeMethod, nil
}

// MergePR merges a pull request if all conditions are met. It logs any errors
// that it encounters.
func MergePR(ctx context.Context, pullCtx pull.Context, merger Merger, mergeConfig MergeConfig) {
	logger := zerolog.Ctx(ctx)

	mergeMethod, err := DetermineMergeMethod(ctx, pullCtx, mergeConfig)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to determine merge method")
		return
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
			logger.Error().Err(err).Msg("Failed to calculate commit message")
			return
		}
		commitMsg.Message = message

		title, err := calculateCommitTitle(ctx, pullCtx, *opt)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to calculate commit title")
			return
		}
		commitMsg.Title = title
	}

	var attempts int
	var merged, retry bool
	for {
		merged, retry = attemptMerge(ctx, pullCtx, merger, mergeMethod, commitMsg)
		if merged || !retry {
			break
		}

		attempts++
		if attempts >= MaxPullRequestPollCount {
			logger.Error().Msgf("Failed to merge pull request after %d attempts", attempts)
			return
		}
		time.Sleep(4 * time.Second)
	}

	_, head := pullCtx.Branches()
	if merged {
		if mergeConfig.DeleteAfterMerge {
			attemptDelete(ctx, pullCtx, head, merger)
		} else {
			logger.Debug().Msgf("Not deleting refs/heads/%s, delete after merge is not enabled", head)
		}
	}
}

// attemptMerge attempts to merge a pull request, logging any errors and
// returing flags to show if the merge suceeded and if a retry is needed.
func attemptMerge(ctx context.Context, pullCtx pull.Context, merger Merger, method MergeMethod, msg CommitMessage) (merged, retry bool) {
	logger := zerolog.Ctx(ctx)

	mergeState, err := pullCtx.MergeState(ctx)
	if err != nil {
		logger.Error().Err(err).Msgf("Failed to get merge state for %q", pullCtx.Locator())
		return false, false
	}

	if mergeState.Closed {
		logger.Debug().Msg("Pull request already closed")
		return false, false
	}

	if mergeState.Mergeable == nil {
		logger.Debug().Msg("Pull request mergeability not yet known")
		return false, true
	}

	if !*mergeState.Mergeable {
		logger.Debug().Msg("Pull request is not mergeable")
		return false, false
	}

	logger.Info().Msgf("Attempting to merge pull request with method %s", method)
	sha, err := merger.Merge(ctx, pullCtx, method, msg)
	if err != nil {
		gerr, ok := errors.Cause(err).(*github.ErrorResponse)
		if !ok {
			logger.Error().Err(err).Msg("Failed to merge pull request")
			return false, true
		}

		switch gerr.Response.StatusCode {
		case http.StatusMethodNotAllowed:
			if gerr.Message == "Base branch was modified. Review and try the merge again." {
				logger.Info().Msg("Base branch was modified, retrying")
				return false, true
			}
			logger.Info().Msgf("Merge rejected due to unsatisfied condition: %q", gerr.Message)
			return false, false
		case http.StatusConflict:
			logger.Info().Msgf("Merge rejected due to being invalid: %q", gerr.Message)
			return false, false
		default:
			logger.Error().Msgf("Merge failed with unexpected status: %d: %q", gerr.Response.StatusCode, gerr.Message)
			return false, true
		}
	}

	logger.Info().Msgf("Successfully merged pull request as SHA %s", sha)
	return true, false
}

// attemptDelete attempts to delete a pull request branch, logging any errors
// and returning true if successful.
func attemptDelete(ctx context.Context, pullCtx pull.Context, head string, merger Merger) bool {
	logger := zerolog.Ctx(ctx)

	if strings.ContainsRune(head, ':') {
		// skip forks because the app doesn't have permission to do the delete
		logger.Debug().Msg("Pull Request is from a fork, not deleting")
		return false
	}

	ref := fmt.Sprintf("refs/heads/%s", head)

	// check other open PRs to make sure that nothing is trying to merge into the ref we're about to delete
	isTargeted, err := pullCtx.IsTargeted(ctx)
	if err != nil {
		logger.Error().Err(err).Msgf("Unabled to determine if %s is targeted by other pull requests", ref)
		return false
	}
	if isTargeted {
		logger.Info().Msgf("Unable to delete %s after merging %q because there are open PRs against it", ref, pullCtx.Locator())
		return false
	}

	logger.Info().Msgf("Attempting to delete ref %s", ref)
	if err := merger.DeleteHead(ctx, pullCtx); err != nil {
		logger.Error().Err(err).Msgf("Failed to delete %s", ref)
		return false
	}

	logger.Info().Msgf("Successfully deleted %s after merging %q", ref, pullCtx.Locator())
	return true
}

func isValidMergeMethod(input MergeMethod) bool {
	return input == SquashAndMerge || input == RebaseAndMerge || input == MergeCommit || input == FastForwardOnly
}

func calculateCommitMessage(ctx context.Context, pullCtx pull.Context, option SquashOptions) (string, error) {
	// As of go-github v30, using the empty string as the commit message
	// selects the default GitHub behavior for the merge mode. To actually
	// clear the message, we use a non-empty string that GitHub will still
	// interpret as empty.
	const (
		defaultMessage = ""
		emptyMessage   = " "
	)

	commitMessage := defaultMessage
	switch option.Body {
	case PullRequestBody:
		if option.MessageDelimiter != "" {
			var quotedDelimiter = regexp.QuoteMeta(option.MessageDelimiter)
			var rString = fmt.Sprintf(`(?sm:(%s\s*)^(.*)$(\s*%s))`, quotedDelimiter, quotedDelimiter)
			matcher, err := regexp.Compile(rString)
			if err != nil {
				return "", errors.Wrap(err, "failed to compile message delimiter regex")
			}

			if m := matcher.FindStringSubmatch(pullCtx.Body()); len(m) == 4 && m[2] != "" {
				commitMessage = m[2]
			} else {
				commitMessage = emptyMessage
			}
		} else {
			commitMessage = pullCtx.Body()
		}
	case EmptyBody:
		commitMessage = emptyMessage
	case SummarizeCommits:
		// Summarizing commits is the default behavior for squash merges
		commitMessage = defaultMessage
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
