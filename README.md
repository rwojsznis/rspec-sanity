# rspec-sanity

Flaky tests reporter that wraps your `rspec` call designed for the CI systems. [See this repo for a fully-working example](https://github.com/rwojsznis/rspec-sanity-example/).

### Motivation

Create an opinionated statically typed wrapper on top of `rspec` instead of creating ad-hoc spaghetti of shell scripts every time at every company that is dealing with a similar problem.

### Status

In working state, battle tested on few projects - _gets the job done_. Supports Github Issues and JIRA (with strong assumptions about how flakies are reported).

### How to use it?

1. Drop binary from the [releases](https://github.com/rwojsznis/rspec-sanity/releases) section into your system's `PATH`
1. Ensure you have configured Rspec's [example_status_persistence_file_path](https://rubydoc.info/gems/rspec-core/RSpec%2FCore%2FConfiguration:example_status_persistence_file_path)
1. Create `.rspec-sanity.toml` configuration file (more details below)
1. From the root project directory execute `rspec-sanity run [test files]` - in the exact same manner same as you would run `rspec [test files]`

Your `[test files]` will be executed _up to two times_ - if something that failed passed on the 2nd attempt it means it's flaky and will be reported as a JIRA ticket/Github issue according to your configuration.

#### Alternative installation method (Debian/Ubuntu)

If you prefer installing binary via `apt` you can use grab `deb` from [gemfury](https://gemfury.com/). Deb packages are generated as part of the release process so they will be always up to date with the Github releases.

```
curl -fsSL https://apt.fury.io/rspec-sanity/gpg.key | gpg --dearmor | sudo tee /etc/apt/trusted.gpg.d/rspec-sanity.gpg > /dev/null
echo "deb [signed-by=/etc/apt/trusted.gpg.d/rspec-sanity.gpg] https://apt.fury.io/rspec-sanity/ * *" | sudo tee /etc/apt/sources.list.d/rspec-sanity.list
sudo apt-get update && sudo apt-get install rspec-sanity
```

### Configuration syntax

By default the app will try to look up `.rspec-sanity.toml` - which can be configured with `--config` switch.

```toml
# defined how to load rspec command
command = "bundle exec rspec"

# arguments that will be passed to your command on the first attempt
arguments = "--format progress --format RspecJunitFormatter -o tmp/rspec/rspec.xml --force-color"

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

# reopen GH issue if it was closed when adding new report?
reopen = true

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
# There is a strong assumption that every JIRA ticket will be
# reported to an epic issue
epic_id = "PROD-1"
# issue type, can vary from project to project
# can be found in JIRA project settings
task_type_id = "10001"
# ID of the JIRA project
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

To authorize with Github you need to set `RSPEC_SANITY_GITHUB_TOKEN` ENV variable - [it can be a personal token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token) or newly [introduced fine-grained token](https://github.blog/2022-10-18-introducing-fine-grained-personal-access-tokens-for-github/) - just make you have access to **write** issues.


### JIRA

To authorize with JIRA you need to set following ENV variables:

- `RSPEC_SANITY_JIRA_TOKEN` - JIRA [personal access token](https://confluence.atlassian.com/enterprise/using-personal-access-tokens-1026032365.html) - I recommend creating a dedicated JIRA user for this purposes
- `RSPEC_SANITY_JIRA_USER` - email address of the token owner
- `RSPEC_SANITY_JIRA_HOST` - full JIRA instance address, with a protocol (`https://`)

#### Creating a test issue

To check your configuration you run `rspec-sanity verify`.

### Todos / nice to haves

- proper interfaces for better tests
- Github-related tests [with go-github-mock](https://github.com/migueleliasweb/go-github-mock)
- uploading artifacts associated with flaky tests (eg. screenshots captured with [capybara-screenshot](https://github.com/mattheworiordan/capybara-screenshot))
- auto-generating `bisect` command for flaky test replication attempt (we would have to grab used rspec seed)
