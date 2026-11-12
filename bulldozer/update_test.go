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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v93/github"
	"github.com/palantir/bulldozer/pull/pulltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdatePRRequiresHeadInBaseRepository(t *testing.T) {
	const headSHA = "8dd53b0e5c0d0a1b6b3c9e2f4a7d8c1e0f9b2a3c"

	tests := map[string]struct {
		BaseRepo     *github.Repository
		HeadRepo     *github.Repository
		ExpectUpdate bool
	}{
		"sameRepository": {
			BaseRepo:     &github.Repository{ID: new(int64(1))},
			HeadRepo:     &github.Repository{ID: new(int64(1))},
			ExpectUpdate: true,
		},
		"sameRepositoryThatIsAFork": {
			BaseRepo:     &github.Repository{ID: new(int64(1)), Fork: new(true)},
			HeadRepo:     &github.Repository{ID: new(int64(1)), Fork: new(true)},
			ExpectUpdate: true,
		},
		"headInFork": {
			BaseRepo:     &github.Repository{ID: new(int64(1))},
			HeadRepo:     &github.Repository{ID: new(int64(2)), Fork: new(true)},
			ExpectUpdate: false,
		},
		"headInParentOfFork": {
			BaseRepo:     &github.Repository{ID: new(int64(1)), Fork: new(true)},
			HeadRepo:     &github.Repository{ID: new(int64(2))},
			ExpectUpdate: false,
		},
		"headRepositoryDeleted": {
			BaseRepo:     &github.Repository{ID: new(int64(1))},
			HeadRepo:     nil,
			ExpectUpdate: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var merges []github.RepositoryMergeRequest

			mux := http.NewServeMux()
			mux.HandleFunc("GET /api/v3/repos/testorg/testrepo/compare/{basehead}", func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"behind_by": 1}`))
			})
			mux.HandleFunc("POST /api/v3/repos/testorg/testrepo/merges", func(w http.ResponseWriter, r *http.Request) {
				var merge github.RepositoryMergeRequest
				if err := json.NewDecoder(r.Body).Decode(&merge); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				merges = append(merges, merge)
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"sha": "deadbeef"}`))
			})

			srv := httptest.NewServer(mux)
			defer srv.Close()

			client, err := github.NewClient(github.WithEnterpriseURLs(srv.URL, srv.URL))
			require.NoError(t, err)

			pullCtx := &pulltest.MockPullContext{
				OwnerValue: "testorg",
				RepoValue:  "testrepo",
			}
			pr := &github.PullRequest{
				State: new("open"),
				Base:  &github.PullRequestBranch{Ref: new("develop"), Repo: test.BaseRepo},
				Head:  &github.PullRequestBranch{Ref: new("feature"), SHA: new(headSHA), Repo: test.HeadRepo},
			}

			updated := UpdatePR(context.Background(), pullCtx, client, pr, UpdateConfig{}, "develop")
			assert.Equal(t, test.ExpectUpdate, updated)

			if test.ExpectUpdate {
				assert.Equal(t, []github.RepositoryMergeRequest{{Base: "feature", Head: "develop"}}, merges)
			} else {
				assert.Empty(t, merges, "the head branch is not in the base repository and must not be merged into")
			}
		})
	}
}
