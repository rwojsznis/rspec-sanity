package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	jira "github.com/andygrunwald/go-jira/v2/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJiraReporterCreatesIssueForNewFlakyFile(t *testing.T) {
	var created jira.Issue
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "user", username)
		assert.Equal(t, "token", password)

		switch r.URL.Path {
		case "/rest/api/2/search":
			assert.Contains(t, r.URL.Query().Get("jql"), `text ~ "\"spec/flaky_spec.rb\""`)
			assert.Equal(t, "10", r.URL.Query().Get("maxResults"))
			_, _ = w.Write([]byte(`{"issues":[],"total":0}`))
		case "/rest/api/2/issue":
			require.Equal(t, http.MethodPost, r.Method)
			require.NoError(t, json.NewDecoder(r.Body).Decode(&created))
			_, _ = w.Write([]byte(`{"id":"10002","key":"APP-2"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	reporter := jiraTestReporter(t, server)
	err := reporter.ReportFlaky([]RspecExample{{Id: "spec/flaky_spec.rb[1:1]", Status: "failed"}})

	require.NoError(t, err)
	require.NotNil(t, created.Fields)
	assert.Equal(t, "spec/flaky_spec.rb", created.Fields.Summary)
	assert.Contains(t, created.Fields.Description, "spec/flaky_spec.rb[1:1]")
	assert.Equal(t, "APP", created.Fields.Project.Key)
	assert.Equal(t, "10001", created.Fields.Type.ID)
	assert.Equal(t, "EPIC-1", created.Fields.Parent.Key)
	assert.Equal(t, []string{"flaky"}, created.Fields.Labels)
}

func TestJiraReporterCommentsOnExactExistingIssue(t *testing.T) {
	var comment JiraSimpleComment
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/2/search":
			_, _ = w.Write([]byte(`{"issues":[` +
				`{"id":"1","key":"APP-1","fields":{"summary":"another issue"}},` +
				`{"id":"2","key":"APP-2","fields":{"summary":"spec/flaky_spec.rb"}}],"total":2}`))
		case "/rest/api/2/issue/2/comment":
			require.Equal(t, http.MethodPost, r.Method)
			require.NoError(t, json.NewDecoder(r.Body).Decode(&comment))
			_, _ = w.Write([]byte(`{"id":"3","body":"created"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	err := jiraTestReporter(t, server).ReportFlaky([]RspecExample{{Id: "spec/flaky_spec.rb[1:2]"}})

	require.NoError(t, err)
	assert.Contains(t, comment.Body, "spec/flaky_spec.rb[1:2]")
}

func TestJiraReporterReturnsSearchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	err := jiraTestReporter(t, server).ReportFlaky([]RspecExample{{Id: "spec/flaky_spec.rb[1:1]"}})
	assert.Error(t, err)
}

func TestJiraReporterVerifyCreatesTestIssue(t *testing.T) {
	var created jira.Issue
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/rest/api/2/issue", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&created))
		_, _ = w.Write([]byte(`{"id":"10003","key":"APP-3"}`))
	}))
	defer server.Close()

	err := jiraTestReporter(t, server).Verify()

	require.NoError(t, err)
	assert.Equal(t, "Test Issue", created.Fields.Summary)
	assert.Contains(t, created.Fields.Description, "some/test-example.rb:1:2")
}

func jiraTestReporter(t *testing.T, server *httptest.Server) *JiraReporter {
	t.Helper()
	config := &JiraConfig{
		EpicId:     "EPIC-1",
		ProjectId:  "APP",
		TaskTypeId: "10001",
		Labels:     []string{"flaky"},
		Template:   `Flaky examples:{{range .Examples}} {{.Id}}{{end}}`,
		user:       "user",
		token:      "token",
		host:       strings.TrimSuffix(server.URL, "/") + "/",
	}
	reporter := NewJiraReporter(config)
	require.NoError(t, reporter.Init())
	return reporter
}
