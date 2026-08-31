package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v90/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGithubReporterCreatesIssueForNewFlakyFile(t *testing.T) {
	var request github.CreateIssueRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/issues":
			_, _ = w.Write([]byte(`{"total_count":0,"items":[]}`))
		case "/repos/owner/repo/issues":
			require.Equal(t, http.MethodPost, r.Method)
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			_, _ = w.Write([]byte(`{"number":7,"title":"spec/flaky_spec.rb"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	reporter := githubTestReporter(t, server)
	err := reporter.ReportFlaky([]RspecExample{{Id: "spec/flaky_spec.rb[1:1]", Status: "failed"}})

	require.NoError(t, err)
	assert.Equal(t, "spec/flaky_spec.rb", request.GetTitle())
	assert.Contains(t, request.GetBody(), "spec/flaky_spec.rb[1:1]")
	assert.Equal(t, []string{"flaky"}, request.Labels)
}

func TestGithubReporterCommentsOnAndReopensClosedIssue(t *testing.T) {
	var comment github.IssueComment
	reopened := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/search/issues":
			_, _ = w.Write([]byte(`{"total_count":1,"items":[{"number":7,"title":"spec/flaky_spec.rb","state":"closed"}]}`))
		case r.URL.Path == "/repos/owner/repo/issues/7/comments":
			require.NoError(t, json.NewDecoder(r.Body).Decode(&comment))
			_, _ = w.Write([]byte(`{"id":1}`))
		case r.URL.Path == "/repos/owner/repo/issues/7" && r.Method == http.MethodPatch:
			reopened = true
			_, _ = w.Write([]byte(`{"number":7,"state":"open"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	reporter := githubTestReporter(t, server)
	reporter.config.Reopen = true
	err := reporter.ReportFlaky([]RspecExample{{Id: "spec/flaky_spec.rb[1:2]", Status: "failed"}})

	require.NoError(t, err)
	assert.Contains(t, comment.GetBody(), "spec/flaky_spec.rb[1:2]")
	assert.True(t, reopened)
}

func TestGithubReporterFallsBackToFirstSearchResult(t *testing.T) {
	commented := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/issues":
			_, _ = w.Write([]byte(`{"total_count":1,"items":[{"number":7,"title":"similar issue","state":"open"}]}`))
		case "/repos/owner/repo/issues/7/comments":
			commented = true
			_, _ = w.Write([]byte(`{"id":1}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	err := githubTestReporter(t, server).ReportFlaky([]RspecExample{{Id: "spec/flaky_spec.rb[1:1]"}})
	require.NoError(t, err)
	assert.True(t, commented)
}

func TestGithubReporterReturnsSearchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	err := githubTestReporter(t, server).ReportFlaky([]RspecExample{{Id: "spec/flaky_spec.rb[1:1]"}})
	assert.Error(t, err)
}

func TestGithubReporterInitAndVerify(t *testing.T) {
	var request github.CreateIssueRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		_, _ = w.Write([]byte(`{"number":7,"title":"Test Issue","html_url":"https://example.com/issues/7"}`))
	}))
	defer server.Close()

	reporter := NewGithubReporter(&GithubConfig{
		Owner: "owner", Repo: "repo", Template: `{{range .Examples}}{{.Id}} {{end}}`, token: "token",
	})
	require.NoError(t, reporter.Init())
	reporter.client = githubTestClient(t, server)

	require.NoError(t, reporter.Verify())
	assert.Equal(t, "Test Issue", request.GetTitle())
	assert.Contains(t, request.GetBody(), "some/test-example.rb:1:2")
}

func TestGithubReporterVerifyReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	err := githubTestReporter(t, server).Verify()
	assert.Error(t, err)
}

func TestGithubReporterReturnsCreateAndCommentErrors(t *testing.T) {
	tests := []struct {
		name   string
		search string
	}{
		{name: "create", search: `{"total_count":0,"items":[]}`},
		{name: "comment", search: `{"total_count":1,"items":[{"number":7,"title":"spec/flaky_spec.rb","state":"open"}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/search/issues" {
					_, _ = w.Write([]byte(tt.search))
					return
				}
				http.Error(w, "write failed", http.StatusInternalServerError)
			}))
			defer server.Close()

			err := githubTestReporter(t, server).ReportFlaky([]RspecExample{{Id: "spec/flaky_spec.rb[1:1]"}})
			assert.Error(t, err)
		})
	}
}

func TestGithubReporterDoesNotReopenWhenDisabled(t *testing.T) {
	reopened := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/issues":
			_, _ = w.Write([]byte(`{"total_count":1,"items":[{"number":7,"title":"spec/flaky_spec.rb","state":"closed"}]}`))
		case "/repos/owner/repo/issues/7/comments":
			_, _ = w.Write([]byte(`{"id":1}`))
		case "/repos/owner/repo/issues/7":
			reopened = true
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	reporter := githubTestReporter(t, server)
	reporter.config.Reopen = false
	require.NoError(t, reporter.ReportFlaky([]RspecExample{{Id: "spec/flaky_spec.rb[1:1]"}}))
	assert.False(t, reopened)
}

func githubTestReporter(t *testing.T, server *httptest.Server) *GithubReporter {
	t.Helper()
	return &GithubReporter{
		config: &GithubConfig{
			Owner:    "owner",
			Repo:     "repo",
			Labels:   []string{"flaky"},
			Template: `Flaky examples:{{range .Examples}} {{.Id}}{{end}}`,
		},
		client: githubTestClient(t, server),
	}
}

func githubTestClient(t *testing.T, server *httptest.Server) *github.Client {
	t.Helper()
	baseURL := server.URL + "/"
	client, err := github.NewClient(
		github.WithHTTPClient(server.Client()),
		github.WithURLs(&baseURL, &baseURL),
	)
	require.NoError(t, err)
	return client
}
