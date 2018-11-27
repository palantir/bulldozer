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
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/palantir/bulldozer/pull"
)

// IsPRBlacklisted returns true if the PR is identified as blacklisted,
// false otherwise. Additionally, a description of the reason will be returned.
func IsPRBlacklisted(ctx context.Context, pullCtx pull.Context, config Signals) (bool, string, error) {
	labels, err := pullCtx.Labels(ctx)
	if err != nil {
		return true, "unable to list PR labels", err
	}

	if inSlice, idx := anyInSliceCaseInsensitive(labels, config.Labels); inSlice {
		return true, fmt.Sprintf("PR label matches one of specified blacklist labels: %q", config.Labels[idx]), nil
	}

	body, err := pullCtx.Body(ctx)
	if err != nil {
		return true, "unable to list PR body", err
	}

	comments, err := pullCtx.Comments(ctx)
	if err != nil {
		return true, "unable to list PR comments", err
	}

	if inSlice, idx := anyInSlice(comments, config.Comments); inSlice {
		return true, fmt.Sprintf("PR comment matches one of specified blacklist comments: %q", config.Comments[idx]), nil
	}

	for _, blacklistedComment := range config.Comments {
		if blacklistedComment == body {
			return true, fmt.Sprintf("PR body matches one of specified blacklist comments: %q", blacklistedComment), nil
		}
	}

	for _, blacklistedSubstring := range config.CommentSubstrings {
		for _, comment := range comments {
			if strings.Contains(comment, blacklistedSubstring) {
				return true, fmt.Sprintf("PR comment matches one of specified blacklist comment substrings: %q", blacklistedSubstring), nil
			}
		}

		if strings.Contains(body, blacklistedSubstring) {
			return true, fmt.Sprintf("PR body matches one of specified blacklist comment substrings: %q", blacklistedSubstring), nil
		}
	}

	for _, blacklistedSubstring := range config.PRBodySubstrings {
		if strings.Contains(body, blacklistedSubstring) {
			return true, fmt.Sprintf("PR body matches one of specified blacklist substrings: %q", blacklistedSubstring), nil
		}
	}

	return false, "no matching blacklist found", nil
}

// IsPRWhitelisted returns true if the PR is identified as whitelisted,
// false otherwise. Additionally, a description of the reason will be returned.
func IsPRWhitelisted(ctx context.Context, pullCtx pull.Context, config Signals) (bool, string, error) {
	labels, err := pullCtx.Labels(ctx)
	if err != nil {
		return false, "unable to list PR labels", err
	}

	if inSlice, idx := anyInSliceCaseInsensitive(labels, config.Labels); inSlice {
		return true, fmt.Sprintf("PR label matches one of specified whitelist labels: %q", config.Labels[idx]), nil
	}

	body, err := pullCtx.Body(ctx)
	if err != nil {
		return false, "unable to list PR body", err
	}

	comments, err := pullCtx.Comments(ctx)
	if err != nil {
		return false, "unable to list PR comments", err
	}

	if inSlice, idx := anyInSlice(comments, config.Comments); inSlice {
		return true, fmt.Sprintf("PR comment matches one of specified whitelist comments: %q", config.Comments[idx]), nil
	}

	for _, whitelistedComment := range config.Comments {
		if whitelistedComment == body {
			return true, fmt.Sprintf("PR body matches one of specified whitelist comments: %q", whitelistedComment), nil
		}
	}

	for _, whitelistedSubstring := range config.CommentSubstrings {
		for _, comment := range comments {
			if strings.Contains(comment, whitelistedSubstring) {
				return true, fmt.Sprintf("PR comment matches one of specified whitelist comment substrings: %q", whitelistedSubstring), nil
			}
		}

		if strings.Contains(body, whitelistedSubstring) {
			return true, fmt.Sprintf("PR body matches one of specified whitelist comment substrings: %q", whitelistedSubstring), nil
		}
	}

	for _, whitelistedSubstring := range config.PRBodySubstrings {
		if strings.Contains(body, whitelistedSubstring) {
			return true, fmt.Sprintf("PR body matches one of specified whitelist substrings: %q", whitelistedSubstring), nil
		}
	}

	return false, "no matching whitelist found", nil
}

func anyInSlice(testValues []string, elements []string) (bool, int) {
	for _, testValue := range testValues {
		for index, element := range elements {
			if testValue == element {
				return true, index
			}
		}
	}
	return false, -1
}

func anyInSliceCaseInsensitive(testValues []string, elements []string) (bool, int) {
	for _, testValue := range testValues {
		for index, element := range elements {
			if strings.EqualFold(testValue, element) {
				return true, index
			}
		}
	}

	return false, -1
}

// setDifference returns all elements in set1 that
// are not in set2.
func setDifference(set1, set2 []string) []string {
	m2 := make(map[string]struct{})
	for _, s2 := range set2 {
		m2[s2] = struct{}{}
	}

	seen := make(map[string]struct{})
	var result []string
	for _, s1 := range set1 {
		if _, ok := m2[s1]; !ok {
			if _, alreadySeen := seen[s1]; !alreadySeen {
				result = append(result, s1)
				seen[s1] = struct{}{}
			}
		}
	}
	return result
}

// ShouldMergePR TODO: may want to return a richer type than bool
func ShouldMergePR(ctx context.Context, pullCtx pull.Context, mergeConfig MergeConfig) (bool, error) {
	logger := zerolog.Ctx(ctx)

	if mergeConfig.Blacklist.Enabled() {
		blacklisted, reason, err := IsPRBlacklisted(ctx, pullCtx, mergeConfig.Blacklist)
		if err != nil {
			return false, errors.Wrap(err, "failed to determine if pull request is blacklisted")
		}
		if blacklisted {
			logger.Debug().Msgf("%s is deemed not mergeable because blacklisting is enabled and %s", pullCtx.Locator(), reason)
			return false, nil
		}
	}

	if mergeConfig.Whitelist.Enabled() {
		whitelisted, reason, err := IsPRWhitelisted(ctx, pullCtx, mergeConfig.Whitelist)
		if err != nil {
			return false, errors.Wrap(err, "failed to determine if pull request is whitelisted")
		}
		if !whitelisted {
			logger.Debug().Msgf("%s is deemed not mergeable because whitelisting is enabled and no whitelist signal detected", pullCtx.Locator())
			return false, nil
		}

		logger.Debug().Msgf("%s is whitelisted because whitelisting is enabled and %s", pullCtx.Locator(), reason)
	}

	requiredStatuses, err := pullCtx.RequiredStatuses(ctx)
	if err != nil {
		return false, errors.Wrap(err, "failed to determine required Github status checks")
	}
	requiredStatuses = append(requiredStatuses, mergeConfig.RequiredStatuses...)

	successStatuses, err := pullCtx.CurrentSuccessStatuses(ctx)
	if err != nil {
		return false, errors.Wrap(err, "failed to determine currently successful status checks")
	}

	unsatisfiedStatuses := setDifference(requiredStatuses, successStatuses)
	if len(unsatisfiedStatuses) > 0 {
		logger.Debug().Msgf("%s is deemed not mergeable because of unfulfilled status checks: [%s]", pullCtx.Locator(), strings.Join(unsatisfiedStatuses, ","))
		return false, nil
	}

	// Ignore required reviews and try a merge (which may fail with a 4XX).

	return true, nil
}
