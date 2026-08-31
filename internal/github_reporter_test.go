package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v50/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGithubReporterCreatesIssueForNewFlakyFile(t *testing.T) {
	var request github.IssueRequest
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
	assert.Equal(t, []string{"flaky"}, *request.Labels)
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

func TestGithubReporterReturnsSearchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	err := githubTestReporter(t, server).ReportFlaky([]RspecExample{{Id: "spec/flaky_spec.rb[1:1]"}})
	assert.Error(t, err)
}

func githubTestReporter(t *testing.T, server *httptest.Server) *GithubReporter {
	t.Helper()
	client := github.NewClient(server.Client())
	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client.BaseURL = baseURL
	client.UploadURL = baseURL

	return &GithubReporter{
		config: &GithubConfig{
			Owner:    "owner",
			Repo:     "repo",
			Labels:   []string{"flaky"},
			Template: `Flaky examples:{{range .Examples}} {{.Id}}{{end}}`,
		},
		client: client,
	}
}
