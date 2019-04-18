// Copyright 2017 alertmanager-github-receiver Authors
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
//////////////////////////////////////////////////////////////////////////////
package issues_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-github/github"
	"github.com/m-lab/alertmanager-github-receiver/issues"
	"github.com/m-lab/go/rtx"
)

// Global vars for tests.
//
// Tests should register handlers on testMux which provide mock responses for
// the Github API method used by the method under test.
var (

	// testMux is the HTTP request multiplexer used with the test server.
	testMux *http.ServeMux

	// testServer is a test HTTP server used to provide mock API responses.
	testServer *httptest.Server
)

// setupServer starts a new http test server and returns the test server URL.
func setupServer() *url.URL {
	// test server.
	testMux = http.NewServeMux()
	testServer = httptest.NewServer(testMux)

	// Test server URL is guaranteed to parse successfully.
	// The github client library requires that the URL end with a slash.
	url, _ := url.Parse(testServer.URL + "/")
	return url
}

// teardownServer stops the test server.
func teardownServer() {
	testServer.Close()
}

func TestClient_CreateIssue(t *testing.T) {
	tests := []struct {
		name       string
		org        string
		repo       string
		title      string
		body       string
		alertLabel string
		extra      []string
		want       *github.Issue
		wantErr    bool
	}{
		{
			name:       "success",
			org:        "fake-org",
			repo:       "fake-repo",
			title:      "fake title",
			body:       "fake issue body",
			alertLabel: "alert:boom:",
			extra:      []string{"extra", "labels"},
			want:       &github.Issue{Number: github.Int(1)},
		},
		{
			name:       "create-returns-error",
			org:        "fake-org",
			repo:       "fake-repo",
			title:      "fake title",
			body:       "fake issue body",
			alertLabel: "alert:boom:",
			extra:      []string{"extra", "labels"},
			want:       nil,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := issues.NewClient(
				tt.org,
				"FAKE-AUTH-TOKEN",
				tt.alertLabel,
			)
			c.GithubClient.BaseURL = setupServer()
			defer teardownServer()

			testMux.HandleFunc("/repos/"+tt.org+"/"+tt.repo+"/issues", func(w http.ResponseWriter, r *http.Request) {
				v := &github.IssueRequest{}
				json.NewDecoder(r.Body).Decode(v)
				authToken := r.Header.Get("Authorization")
				if !strings.Contains(authToken, "FAKE-AUTH-TOKEN") {
					t.Errorf("Request does not contain bearer token")
				}
				if *v.Title != tt.title {
					t.Errorf("Request title = %+v, want %+v", *v.Title, tt.title)
				}
				if *v.Body != tt.body {
					t.Errorf("Request body = %+v, want %+v", *v.Body, tt.body)
				}
				if tt.wantErr {
					w.WriteHeader(http.StatusNotFound)
					fmt.Fprint(w, `error`)
				} else {
					fmt.Fprint(w, `{"number":1}`)
				}
			})

			got, err := c.CreateIssue(tt.repo, tt.title, tt.body, tt.extra)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.CreateIssue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateIssue returned %+v, want %+v", got, tt.want)
			}
		})
	}
}

var (
	result = `{
		"url": "https://api.github.com/repos/batterseapower/pinyin-toolkit/issues/132",
		"html_url": "https://github.com/batterseapower/pinyin-toolkit/issues/132",
		"id": 35802,
		"number": 132,
		"title": "Line Number Indexes Beyond 20 Not Displayed",
		"state": "open"
	  }`
	listResults = `
{
	"total_count": 2,
	"incomplete_results": true,
	"items": [` + result + `]
}`
)

func TestClient_ListOpenIssues(t *testing.T) {
	var issue github.Issue
	err := json.Unmarshal([]byte(result), &issue)
	rtx.Must(err, "Failed to unmarshal issue: %q", result)

	tests := []struct {
		name       string
		org        string
		alertLabel string
		want       []*github.Issue
		wantErr    bool
	}{
		{
			name:       "success",
			org:        "fake-org",
			alertLabel: "alert",
			want: []*github.Issue{
				&issue,
				&issue,
			},
		},
		{
			name:       "list-returns-error",
			org:        "fake-org",
			alertLabel: "alert",
			want:       nil,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := issues.NewClient(
				tt.org,
				"FAKE-AUTH-TOKEN",
				tt.alertLabel,
			)
			c.GithubClient.BaseURL = setupServer()
			defer teardownServer()

			count := 0
			testMux.HandleFunc("/search/issues", func(w http.ResponseWriter, r *http.Request) {
				if tt.wantErr {
					w.WriteHeader(http.StatusNotFound)
					fmt.Fprint(w, `error`)
					return
				}
				// r.ParseForm()
				if count == 0 {
					w.Header().Set("Link", `<https://api.github.com/resource?page=2>; rel="next"`)
				}
				count++
				w.Write([]byte(listResults))
			})

			got, err := c.ListOpenIssues()
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.ListOpenIssues() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.ListOpenIssues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func xxxTestCloseIssue(t *testing.T) {
	client := issues.NewClient("owner", "FAKE-AUTH-TOKEN", "alert:boom:")
	client.GithubClient.BaseURL = setupServer()
	defer teardownServer()

	u := "https://api.github.com/repos/fake-owner/fake-repo"
	testMux.HandleFunc("/repos/fake-owner/fake-repo/issues/1", func(w http.ResponseWriter, r *http.Request) {
		v := &github.IssueRequest{}
		err := json.NewDecoder(r.Body).Decode(v)
		if err != nil {
			t.Fatal(err)
		}

		// Fake result.
		fmt.Fprintf(w, `{"number":1, "repository_url":"%s"}`, u)
	})

	openIssue := &github.Issue{Number: github.Int(1), RepositoryURL: &u}

	closedIssue, err := client.CloseIssue(openIssue)
	if err != nil {
		t.Errorf("CloseIssue returned error: %v", err)
	}

	if !reflect.DeepEqual(openIssue, closedIssue) {
		t.Errorf("CloseIssue returned %+v, want %+v", closedIssue, openIssue)
	}
}

func TestClient_CloseIssue(t *testing.T) {
	tests := []struct {
		name    string
		org     string
		issue   *github.Issue
		want    *github.Issue
		wantErr bool
	}{
		{
			name: "success",
			org:  "fake-org",
			issue: &github.Issue{
				Number:        github.Int(1),
				RepositoryURL: github.String("https://api.github.com/repos/fake-org/fake-repo"),
			},
			want: &github.Issue{
				Number:        github.Int(1),
				RepositoryURL: github.String("https://api.github.com/repos/fake-org/fake-repo"),
			},
		},
		{
			name:    "error-empty-repository-url",
			org:     "fake-org",
			issue:   &github.Issue{RepositoryURL: nil}, // Empty repostiry url.
			wantErr: true,
		},
		{
			name:    "error-repository-parse-url",
			org:     "fake-org",
			issue:   &github.Issue{RepositoryURL: github.String("-://bad-url.com")}, // URL fails to parse.
			wantErr: true,
		},
		{
			name:    "error-repository-wrong-field-count",
			org:     "fake-org",
			issue:   &github.Issue{RepositoryURL: github.String("https://api.github.com/fake-org/fake-repo")}, // Too many fields.
			wantErr: true,
		},
		{
			name: "error-close-returns-error",
			org:  "fake-org",
			issue: &github.Issue{
				Number:        github.Int(1),
				RepositoryURL: github.String("https://api.github.com/repos/fake-org/fake-repo"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := issues.NewClient(
				tt.org,
				"FAKE-AUTH-TOKEN",
				"",
			)
			c.GithubClient.BaseURL = setupServer()
			defer teardownServer()

			u := "https://api.github.com/repos/fake-org/fake-repo"
			testMux.HandleFunc("/repos/fake-org/fake-repo/issues/1", func(w http.ResponseWriter, r *http.Request) {
				// TODO: add rate limit headers in response to trigger a RateLimitError.
				if tt.wantErr {
					w.WriteHeader(http.StatusNotFound)
					fmt.Fprint(w, `error`)
					return
				}
				v := &github.IssueRequest{}
				err := json.NewDecoder(r.Body).Decode(v)
				if err != nil {
					t.Fatal(err)
				}
				fmt.Fprintf(w, `{"number":1, "repository_url":"%s"}`, u)
			})

			got, err := c.CloseIssue(tt.issue)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.CloseIssue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.CloseIssue() = %v, want %v", got, tt.want)
			}
		})
	}
}
