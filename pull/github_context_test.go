// Copyright 2026 Palantir Technologies, Inc.
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
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-github/v90/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type requestCounter struct {
	mu     sync.Mutex
	counts map[string]int
}

func (rc *requestCounter) count(path string) int {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.counts[path]
}

func newEmptyResultsContext(t *testing.T) (*GithubContext, *requestCounter) {
	rc := &requestCounter{counts: make(map[string]int)}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v3")
		rc.mu.Lock()
		rc.counts[path]++
		rc.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(path, "/status"):
			_, _ = w.Write([]byte(`{"statuses": []}`))
		case strings.HasSuffix(path, "/check-runs"):
			_, _ = w.Write([]byte(`{"total_count": 0, "check_runs": []}`))
		default:
			_, _ = w.Write([]byte(`[]`))
		}
	}))
	t.Cleanup(srv.Close)

	client, err := github.NewClient(github.WithEnterpriseURLs(srv.URL, srv.URL))
	require.NoError(t, err)

	pr := &github.PullRequest{
		Number: new(1),
		Base: &github.PullRequestBranch{
			Repo: &github.Repository{
				Name:  new("testrepo"),
				Owner: &github.User{Login: new("testorg")},
			},
		},
		Head: &github.PullRequestBranch{SHA: new("abc123")},
	}
	return NewGithubContext(client, pr).(*GithubContext), rc
}

func TestCommentsCachesEmptyResult(t *testing.T) {
	ghc, rc := newEmptyResultsContext(t)
	ctx := context.Background()

	for range 3 {
		comments, err := ghc.Comments(ctx)
		require.NoError(t, err)
		assert.Empty(t, comments)
	}

	assert.Equal(t, 1, rc.count("/repos/testorg/testrepo/pulls/1/comments"))
	assert.Equal(t, 1, rc.count("/repos/testorg/testrepo/issues/1/comments"))
}

func TestCurrentSuccessStatusesCachesEmptyResult(t *testing.T) {
	ghc, rc := newEmptyResultsContext(t)
	ctx := context.Background()

	for range 3 {
		statuses, err := ghc.CurrentSuccessStatuses(ctx)
		require.NoError(t, err)
		assert.Empty(t, statuses)
	}

	assert.Equal(t, 1, rc.count("/repos/testorg/testrepo/commits/abc123/status"))
	assert.Equal(t, 1, rc.count("/repos/testorg/testrepo/commits/abc123/check-runs"))
}
