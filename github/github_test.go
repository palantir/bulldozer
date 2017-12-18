// Copyright 2017 Palantir Technologies, Inc.
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

package github

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	mux    *http.ServeMux
	client *Client
	server *httptest.Server
)

func setup() {
	// test server
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	logger := logrus.New().WithField("deliveryID", "randomDelivery")
	client = &Client{logger, context.TODO(), github.NewClient(nil)}
	url, _ := url.Parse(server.URL + "/")
	client.BaseURL = url
	client.UploadURL = url
}

// teardown closes the test HTTP server.
func teardown() {
	server.Close()
}

func testMethod(t *testing.T, r *http.Request, want string) {
	if got := r.Method; got != want {
		t.Errorf("Request method: %v, want %v", got, want)
	}
}

type values map[string]string

func testFormValues(t *testing.T, r *http.Request, values values) {
	want := url.Values{}
	for k, v := range values {
		want.Set(k, v)
	}

	_ = r.ParseForm()
	if got := r.Form; !reflect.DeepEqual(got, want) {
		t.Errorf("Request parameters: %v, want %v", got, want)
	}
}

func testBody(t *testing.T, r *http.Request, want string) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Errorf("Error reading request body: %v", err)
	}
	if got := string(b); got != want {
		t.Errorf("request Body is %s, want %s", got, want)
	}
}

func fakeUser(login string) *github.User {
	return &github.User{
		Login: github.String(login),
	}
}

func fakeRepository(name string) *github.Repository {
	return &github.Repository{
		Owner:    fakeUser("o"),
		Name:     github.String(name),
		FullName: github.String("o/r"),
	}
}

func fakePullRequestBranch() *github.PullRequestBranch {
	return &github.PullRequestBranch{
		Repo: fakeRepository("r"),
	}
}

func fakePullRequest(number int) *github.PullRequest {
	return &github.PullRequest{
		Base:   fakePullRequestBranch(),
		Number: github.Int(number),
	}
}

func TestLastReviewFromUser(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/pulls/1/reviews", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `
		[
			{
				"id": 1,
				"user": {
					"login": "octocat"
				}
			},
			{
				"id": 2,
				"user": {
					"login": "u"
				}
			},
			{
				"id": 3,
				"user": {
					"login": "u"
				}
			}
		]`)
	})

	lastReview, err := client.LastReviewFromUser(fakePullRequest(1), fakeUser("u"))
	require.Nil(t, err)

	want := &github.PullRequestReview{
		ID:   github.Int(2),
		User: fakeUser("u"),
	}
	if !reflect.DeepEqual(lastReview, want) {
		t.Errorf("PullRequests.ListReviews returned %+v, want %+v", lastReview, want)
	}
}

func TestHasLabel(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/issues/1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `
		{
			"labels": [
				{
					"id": 1,
					"name": "label_1"
				},
				{
					"id": 2,
					"name": "label_2"
				},
				{
					"id": 3,
					"name": "label_3"
				},
				{
					"id": 4,
					"name": "CAPITAL_LABEL"
				}
			]
		}`)
	})

	pr := &github.PullRequest{
		Base: &github.PullRequestBranch{
			Repo: &github.Repository{
				Owner: &github.User{
					Login: github.String("o"),
				},
				Name: github.String("r"),
			},
		},
		Number: github.Int(1),
	}

	hasLabels, err := client.HasLabels(pr, []string{"label_1", "label_2"})
	require.Nil(t, err)
	assert.True(t, hasLabels)

	hasLabels, err = client.HasLabels(pr, []string{"unknown_label"})
	require.Nil(t, err)
	assert.False(t, hasLabels)

	hasLabels, err = client.HasLabels(pr, []string{"unknown_label"})
	require.Nil(t, err)
	assert.False(t, hasLabels)

	hasLabels, err = client.HasLabels(pr, []string{})
	require.Nil(t, err)
	assert.False(t, hasLabels)

	hasLabels, err = client.HasLabels(pr, []string{"capital_label"})
	require.Nil(t, err)
	assert.True(t, hasLabels)
}

func TestLastStatusForContext(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/commits/1234/statuses", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testFormValues(t, r, values{
			"per_page": "100",
		})

		fmt.Fprint(w, `
		[
			{
				"id": 1,
				"context": "context_1",
				"state": "success"
			},
			{
				"id": 2,
				"context": "context_2",
				"state": "pending"
			},
			{
				"id": 3,
				"context": "context_1",
				"state": "success"
			}
		]`)
	})

	repo := fakeRepository("r")
	sha := "1234"

	status, err := client.LastStatusForContext(repo, sha, "context_1")
	require.Nil(t, err)
	assert.Equal(t, "success", status)

	status, err = client.LastStatusForContext(repo, sha, "context_2")
	require.Nil(t, err)
	assert.Equal(t, "pending", status)

	status, err = client.LastStatusForContext(repo, sha, "inexistent_context")
	require.NotNil(t, err)
	assert.Equal(t, "", status)
}

func TestAllPullRequestForSHA(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/pulls", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `
		[
			{
				"id": 1,
				"number": 1,
				"state": "open",
				"head": {
					"ref": "ref_1",
					"sha": "1234"
				}
			},
			{
				"id": 3,
				"number": 3,
				"state": "open",
				"head": {
					"ref": "ref_3",
					"sha": "12345"
				}
			}
		]`)
	})

	mux.HandleFunc("/repos/o/r/pulls/1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `
		{
			"id": 1,
			"number": 1,
			"state": "open",
			"head": {
				"ref": "ref_1",
				"sha": "1234"
			},
			"mergeable": true
		}`)
	})

	want := &github.PullRequest{
		ID:     github.Int(1),
		Number: github.Int(1),
		State:  github.String("open"),
		Head: &github.PullRequestBranch{
			Ref: github.String("ref_1"),
			SHA: github.String("1234"),
		},
		Mergeable: github.Bool(true),
	}
	pr, err := client.PullRequestForSHA(fakeRepository("r"), "1234")
	require.Nil(t, err)
	if !reflect.DeepEqual(pr, want) {
		t.Errorf("PullRequestForSHA returned %+v, want %+v", pr, want)
	}

	pr, err = client.PullRequestForSHA(fakeRepository("r"), "inexistent_sha")
	require.Nil(t, err)
	require.Nil(t, pr)
}

func TestShaStatusSuccess(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/branches/develop/protection", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `
		{
		  "url": "https://github.com",
		  "required_status_checks": {
			"url": "https://github.com",
			"strict": true,
			"contexts": [
			  "context_1",
			  "context_2"
			],
			"contexts_url": "https://github.com"
		  },
		  "enforce_admins": {
			"url": "https://github.com",
			"enabled": true
		  }
		}
		`)
	})
	mux.HandleFunc("/repos/o/r/commits/develop/status", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `
		{
		  "state": "success",
		  "statuses": [
			{
			  "id": 15263,
			  "state": "success",
			  "context": "context_1"
			},
			{
			  "id": 15265,
			  "state": "success",
			  "context": "context_2"
			}
		  ]
		}
		`)
	})
	mux.HandleFunc("/repos/o/r/branches/develop/protection/required_status_checks", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `
		{
		  "contexts": [
			"context_1",
			"context_2"
		  ]
		}
		`)
	})
	mux.HandleFunc("/repos/o/r/commits/develop/statuses", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `
		[
		  {
			"state": "success",
			"context": "context_1"
		  },
		  {
			"state": "success",
			"context": "context_2"
		  },
		  {
			"state": "pending",
			"context": "context_1"
		  }
		]
		`)
	})

	pr := &github.PullRequest{
		Number: github.Int(1),
		Base: &github.PullRequestBranch{
			Repo: fakeRepository("r"),
			Ref:  github.String("develop"),
		},
	}
	status, err := client.ShaStatus(pr, "develop")
	require.Nil(t, err)
	assert.Equal(t, true, status)
}

func TestShaStatusFailure(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/branches/develop/protection", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `
		{
		  "url": "https://github.com",
		  "required_status_checks": {
			"url": "https://github.com",
			"strict": true,
			"contexts": [
			  "context_1",
			  "context_2"
			],
			"contexts_url": "https://github.com"
		  },
		  "enforce_admins": {
			"url": "https://github.com",
			"enabled": true
		  }
		}
		`)
	})
	mux.HandleFunc("/repos/o/r/commits/develop/status", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `
		{
		  "state": "success",
		  "statuses": [
			{
			  "id": 15263,
			  "state": "success",
			  "context": "context_1"
			},
			{
			  "id": 15265,
			  "state": "success",
			  "context": "context_2"
			}
		  ]
		}
		`)
	})
	mux.HandleFunc("/repos/o/r/branches/develop/protection/required_status_checks", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `
		{
		  "contexts": [
			"context_1",
			"context_2"
		  ]
		}
		`)
	})
	mux.HandleFunc("/repos/o/r/commits/develop/statuses", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `
		[
		  {
			"state": "failed",
			"context": "context_1"
		  },
		  {
			"state": "success",
			"context": "context_2"
		  },
		  {
			"state": "pending",
			"context": "context_1"
		  }
		]
		`)
	})

	pr := &github.PullRequest{
		Number: github.Int(1),
		Base: &github.PullRequestBranch{
			Repo: fakeRepository("r"),
			Ref:  github.String("develop"),
		},
	}
	status, err := client.ShaStatus(pr, "develop")
	require.Nil(t, err)
	assert.Equal(t, false, status)
}

func TestConfigFileSuccess(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/contents/.bulldozer.yml", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{
		  "type": "file",
		  "encoding": "base64",
		  "content": "bW9kZTogd2hpdGVsaXN0DQpzdHJhdGVneTogc3F1YXNoDQpkZWxldGVBZnRlck1lcmdlOiB0cnVlDQp1cGRhdGVTdHJhdGVneTogZGVmZXJUb1BS",
		  "name": ".bulldozer.yml",
		  "path": ".bulldozer.yml"
		}`)
	})

	want := &BulldozerFile{
		Mode:             "whitelist",
		MergeStrategy:    "squash",
		UpdateStrategy:   UpdateStrategyDeferToPR,
		DeleteAfterMerge: true,
	}
	configFile, err := client.ConfigFile(fakeRepository("r"), "develop")
	require.Nil(t, err)

	if !reflect.DeepEqual(configFile, want) {
		t.Errorf("ConfigFile returned %+v, want %+v", configFile, want)
	}
}

func TestConfigFileInvalid(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/contents/.bulldozer.yml", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{
		  "type": "file",
		  "encoding": "base64",
		  "content": "invalidContent",
		  "name": ".bulldozer.yml",
		  "path": ".bulldozer.yml"
		}`)
	})

	configFile, err := client.ConfigFile(fakeRepository("r"), "develop")
	require.NotNil(t, err)
	assert.Empty(t, configFile, ".bulldozer.yml not empty")
}

func TestOperationModeInvalid(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/contents/.bulldozer.yml", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{
		  "type": "file",
		  "encoding": "base64",
		  "content": "bW9kZTogaW52YWxpZApzdHJhdGVneTogc3F1YXNoCmRlbGV0ZUFmdGVyTWVy\nZ2U6IHRydWUK\n",
		  "name": ".bulldozer.yml",
		  "path": ".bulldozer.yml"
		}`)
	})

	branch := &github.PullRequestBranch{
		Ref:  github.String("develop"),
		Repo: fakeRepository("r"),
	}
	operationMode, err := client.OperationMode(branch)
	require.NotNil(t, err)
	assert.Empty(t, operationMode, "operationMode should be empty")
}

func TestMergeMethodAllowed(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{
		  "allow_squash_merge": true,
		  "allow_merge_commit": true,
		  "allow_rebase_merge": true
		}`)
	})

	mux.HandleFunc("/repos/o/r/contents/.bulldozer.yml", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{
		  "type": "file",
		  "encoding": "base64",
		  "content": "bW9kZTogd2hpdGVsaXN0CnN0cmF0ZWd5OiBzcXVhc2gKZGVsZXRlQWZ0ZXJN\nZXJnZTogdHJ1ZQo=\n",
		  "name": ".bulldozer.yml",
		  "path": ".bulldozer.yml"
		}`)
	})

	branch := &github.PullRequestBranch{
		Ref:  github.String("develop"),
		Repo: fakeRepository("r"),
	}
	mergeMethod, err := client.MergeMethod(branch)
	require.Nil(t, err)
	assert.Equal(t, SquashMethod, mergeMethod)
}

func TestMergeMethodDisallowed(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{
		  "allow_merge_commit": true,
		  "allow_squash_merge": false,
		  "allow_rebase_merge": true
		}`)
	})

	mux.HandleFunc("/repos/o/r/contents/.bulldozer.yml", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{
		  "type": "file",
		  "encoding": "base64",
		  "content": "bW9kZTogd2hpdGVsaXN0CnN0cmF0ZWd5OiBzcXVhc2gKZGVsZXRlQWZ0ZXJN\nZXJnZTogdHJ1ZQo=\n",
		  "name": ".bulldozer.yml",
		  "path": ".bulldozer.yml"
		}`)
	})

	branch := &github.PullRequestBranch{
		Ref:  github.String("develop"),
		Repo: fakeRepository("r"),
	}
	mergeMethod, err := client.MergeMethod(branch)
	require.Nil(t, err)
	assert.Equal(t, MergeMethod, mergeMethod)
}

func TestCommitMessage(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/pulls/1/commits", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `
			[
				{
					"commit": {
						"message": "1st commit msg"
					}
				},
				{
					"commit": {
						"message": "2nd commit msg"
					}
				},
				{
					"commit": {
						"message": "3rd commit msg"
					}
				}
			]
		`)

	})

	pr := fakePullRequest(1)
	commitMessages, err := client.CommitMessages(pr)
	require.Nil(t, err)
	require.Equal(t, []string{
		"* 1st commit msg", "* 2nd commit msg", "* 3rd commit msg",
	}, commitMessages)
}
