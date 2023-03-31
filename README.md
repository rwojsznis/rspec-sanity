# rspec-sanity

Flaky tests reporter that wraps your `rspec` call designed for the CI systems. [See this repo for fully-working example](https://github.com/rwojsznis/rspec-sanity-example/).

### Motivation

Create opinionated statically typed wrapper on top of `rspec` instead of creating ad-hoc spaghetti of shell scripts every time at every company that is dealing with similar problem.

### Status

Working proof of concept. Some tests and terrible code quality. Supports Github Issues and JIRA (with strong assumptions about how flakies are reported).

### How to use it?

1. Drop binary from the releases section into your system's `PATH`
1. Ensure you have configured Rspec's [example_status_persistence_file_path](https://rubydoc.info/gems/rspec-core/RSpec%2FCore%2FConfiguration:example_status_persistence_file_path)
1. Create `.rspec-sanity.toml` configuration file (more details below)
1. From the root project directory execute `rspec-sanity run [test files]` - in exact same manner same as you would run `rspec [test files]`

Your `[test files]` will be executed _up to two times_ - if something that failed passed on 2nd attempt it means it's flaky and will be reported as jira ticket/Github issue according to your configuration.


### Configuration syntax

By default app will try to lookup `.rspec-sanity.toml` - which can be configured with `--config` switch.

```toml
# defined how to load rspec command
command = "bundle exec rspec"

# arguments that will be passed to your command on the first attempt
arguments = "--format progress --format RspecJunitFormatter -o ~/rspec/rspec.xml --force-color"

# arguments used for the 2nd attempt (re-run)
rerun_arguments = "--format documentation --force-color"

# file path defined for example_status_persistence_file_path in Rspec
persistence_file = "spec/examples.txt"

# Right now you can use github or jira reporters
# only one will be picked up
[github]
owner = "rwojsznis"
repo = "rspec-sanity"
# optional labels
labels = ['flaky-spec']
# Under .Env you will find all available env variables on the system
# Here I'm using some handy stuff defined by CircleCI
template = '''
Failed build: {{ .Env.CIRCLE_BUILD_URL }}
Node: {{ .Env.CIRCLE_NODE_INDEX }}
Branch: {{ .Env.CIRCLE_BRANCH }}

| Example |
| --- |
{{- range .Examples}}
| {{ .Id }} |
{{- end}}
'''

[jira]
# There is a strong assumption that every jira ticket will be
# reported to a epic issue
epic_id = "PROD-1"
# issue type, can vary from project to project
# can be found in jira project settings
task_type_id = "10001"
# ID of the jira project
project_id = "PROD"
# optional labels
labels = ['flaky-spec']
template = '''
Failed build: {{ .Env.CIRCLE_BUILD_URL }}
Node: {{ .Env.CIRCLE_NODE_INDEX }}
Branch: {{ .Env.CIRCLE_BRANCH }}

| Example |
{{- range .Examples}}
| {{ .Id }} |
{{- end}}
'''
```

### Additional configuration per reporter

#### Github

To authorize with Github you need to setup `RSPEC_SANITY_GITHUB_TOKEN` - [it can be a personal token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token) or newly [introduced fine-grained token](https://github.blog/2022-10-18-introducing-fine-grained-personal-access-tokens-for-github/) - just make you you have access to **write** issues.


### JIRA
To authorize with JIRA you need to setup few more things:

- `RSPEC_SANITY_JIRA_TOKEN` - jira [personal access token](https://confluence.atlassian.com/enterprise/using-personal-access-tokens-1026032365.html) - I recommend creating a dedicated JIRA user for this purposes
- `RSPEC_SANITY_JIRA_USER` - email address of token owner
- `RSPEC_SANITY_JIRA_HOST` - your jira instance address, with protocol (`https://`)

#### Creating a test issue

To check your configuration you can run `rspec-sanity verify`.
