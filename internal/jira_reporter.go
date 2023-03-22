package internal

import (
	"context"
	"fmt"
	"net/http"

	"log"

	jira "github.com/andygrunwald/go-jira/v2/cloud"
	"golang.org/x/exp/slices"
)

type JiraReporter struct {
	config *JiraConfig
	client *jira.Client
}

func NewJiraReporter(jc *JiraConfig) *JiraReporter {
	return &JiraReporter{
		config: jc,
	}
}

func (jr *JiraReporter) Init() error {
	tp := jira.BasicAuthTransport{
		Username: jr.config.GetUser(),
		APIToken: jr.config.GetToken(),
	}

	client, err := jira.NewClient(
		jr.config.GetHost(),
		tp.Client(),
	)

	if err != nil {
		return err
	}

	jr.client = client

	return nil
}

func (jr *JiraReporter) Verify() error {
	log.Println("[jira] Verifying reporter")
	testFlakies := []RspecExample{
		{Id: "some/test-example.rb:1:2"},
		{Id: "some/test-example.rb:10:2"},
	}
	template, err := RenderTemplate(jr.config.Template, testFlakies)
	if err != nil {
		return err
	}

	issue, err := jr._createIssue("Test Issue", template, jr.config.Labels)

	if err != nil {
		return err
	}

	log.Printf("[jira] Created test issue: %s/browse/%s", jr.config.host, issue.Key)

	return nil
}

func (jr *JiraReporter) ReportFlaky(flakies []RspecExample) error {
	issueTitle := flakies[0].Filename()

	query := fmt.Sprintf(
		`project = %s AND ("Epic Link" = %s OR parent = %s) AND text ~ "\"%s\""`,
		jr.config.ProjectId,
		jr.config.EpicId,
		jr.config.EpicId,
		issueTitle,
	)

	issues, _, err := jr.client.Issue.Search(
		context.Background(),
		query,
		&jira.SearchOptions{
			MaxResults: 10,
		})

	if err != nil {
		log.Println("[jira] Error searching for issues")
		return err
	}

	if len(issues) == 0 {
		log.Println("No issues found, creating new one")
		return jr.createIssue(flakies)
	} else {
		idx := slices.IndexFunc(issues, func(c jira.Issue) bool {
			return c.Fields.Summary == flakies[0].Filename()
		})

		if idx == -1 {
			log.Println("Can't find exact match, doing fallback")
			idx = 0
		}

		return jr.addIssueComment(&issues[idx], flakies)
	}
}

type JiraSimpleComment struct {
	Body string `json:"body"`
}

// https://github.com/andygrunwald/go-jira/issues/604
func (jr *JiraReporter) addComment(issue *jira.Issue, body string) error {
	comment := JiraSimpleComment{
		Body: body,
	}

	apiEndpoint := fmt.Sprintf("rest/api/2/issue/%s/comment", issue.ID)
	req, err := jr.client.NewRequest(context.Background(), http.MethodPost, apiEndpoint, comment)

	if err != nil {
		return err
	}
	responseComment := new(jira.Comment)
	_, err = jr.client.Do(req, responseComment)
	if err != nil {
		return err
	}
	return nil
}

func (jr *JiraReporter) addIssueComment(issue *jira.Issue, flakies []RspecExample) error {
	body, err := RenderTemplate(jr.config.Template, flakies)
	if err != nil {
		return err
	}

	err = jr.addComment(issue, body)
	if err != nil {
		return err
	}

	log.Printf("[jira] Added comment to issue: %s", issue.Key)

	return nil
}

func (jr *JiraReporter) createIssue(flakies []RspecExample) error {
	body, err := RenderTemplate(jr.config.Template, flakies)
	if err != nil {
		return err
	}

	newIssue, err := jr._createIssue(
		flakies[0].Filename(),
		body,
		jr.config.Labels,
	)

	if err != nil {
		return err
	}

	log.Printf("[jira] Created new issue: %s", newIssue.Key)

	return nil
}

func (jr *JiraReporter) _createIssue(title string, body string, labels []string) (*jira.Issue, error) {
	if len(labels) == 0 {
		labels = make([]string, 0)
	}

	issue := jira.Issue{
		Fields: &jira.IssueFields{
			Description: body,
			Type: jira.IssueType{
				ID: jr.config.TaskTypeId,
			},
			Project: jira.Project{
				Key: jr.config.ProjectId,
			},
			Parent:  &jira.Parent{Key: jr.config.EpicId},
			Summary: title,
			Labels:  labels,
		},
	}

	newIssue, _, err := jr.client.Issue.Create(
		context.Background(),
		&issue,
	)

	return newIssue, err
}
