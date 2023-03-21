package internal

import (
	"context"
	"fmt"
	"log"

	"github.com/google/go-github/v50/github"
	"golang.org/x/exp/slices"
	"golang.org/x/oauth2"
)

type GithubReporter struct {
	config *GithubConfig
	client *github.Client
}

func NewGithubReporter(gc *GithubConfig) *GithubReporter {
	return &GithubReporter{
		config: gc,
	}
}

func (gr *GithubReporter) Init() error {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: gr.config.GetToken()},
	)
	tc := oauth2.NewClient(ctx, ts)

	gr.client = github.NewClient(tc)
	return nil
}

func (gr *GithubReporter) Verify() error {
	log.Println("[github] Verifying reporter")

	testFlakies := []RspecExample{
		{Id: "some/test-example.rb:1:2"},
		{Id: "some/test-example.rb:10:2"},
	}
	template, err := RenderTemplate(gr.config.Template, testFlakies)
	if err != nil {
		return err
	}

	issue, err := gr._createIssue("Test Issue", template, gr.config.Labels)

	if err != nil {
		return err
	}

	log.Printf("[github] Created test issue: %s", *issue.HTMLURL)
	
	return nil
}

func (gr *GithubReporter) ReportFlaky(flakies []RspecExample) error {
	issueTitle := flakies[0].Filename()
	query := fmt.Sprintf("\"%s\" in:title repo:%s/%s is:issue",
		issueTitle,
		gr.config.Owner,
		gr.config.Repo,
	)

	results, _, err := gr.client.Search.Issues(context.Background(), query, &github.SearchOptions{
		ListOptions: github.ListOptions{
			Page:    0,
			PerPage: 10,
		},
	})

	if err != nil {
		return err
	}

	if *results.Total == 0 {
		log.Println("[github] No issues found, creating new one")
		return gr.createIssue(flakies)
	} else {

		idx := slices.IndexFunc(results.Issues, func(c *github.Issue) bool {
			return *c.Title == flakies[0].Filename()
		})

		if idx == -1 {
			log.Println("[github] Can't find exact match, doing fallback")
			idx = 0
		}

		log.Printf("[github] Adding comment to issue %s", *(results.Issues[idx]).Title)

		return gr.addIssueComment(results.Issues[idx], flakies)
	}
}

func (gr *GithubReporter) addIssueComment(issue *github.Issue, flakies []RspecExample) error {
	body, err := RenderTemplate(gr.config.Template, flakies)
	if err != nil {
		return err
	}

	comment := &github.IssueComment{
		Body: github.String(body),
	}

	_, _, err = gr.client.Issues.CreateComment(
		context.Background(),
		gr.config.Owner,
		gr.config.Repo,
		*issue.Number,
		comment,
	)

	if err != nil {
		return err
	}

	return nil
}

func (gr *GithubReporter) createIssue(flakies []RspecExample) error {
	body, err := RenderTemplate(gr.config.Template, flakies)
	if err != nil {
		return err
	}

	issue, err := gr._createIssue(flakies[0].Filename(), body, gr.config.Labels)

	if err != nil {
		return err
	}

	log.Printf("[github] Created new issue: %s", *issue.Title)

	return nil
}

func (gr *GithubReporter) _createIssue(title string, body string, labels []string) (*github.Issue, error) {
	if len(labels) == 0 {
		labels = make([]string, 0)
	}

	issue := &github.IssueRequest{
		Title:  github.String(title),
		Body:   github.String(body),
		Labels: &labels,
	}

	newIssue, _, err := gr.client.Issues.Create(
		context.Background(),
		gr.config.Owner,
		gr.config.Repo,
		issue,
	)

	return newIssue, err
}
