package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCommandExitsWithSuccessfulRspecStatus(t *testing.T) {
	config := writeCLIConfig(t, executable(t, "true"), filepath.Join(t.TempDir(), "examples.txt"))
	exitCode := -1

	err := newApp(func(code int) { exitCode = code }).Run([]string{
		"rspec-sanity", "--config", config, "run", "spec/models",
	})

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
}

func TestRunCommandPropagatesFailedRspecStatusWhenRerunIsSkipped(t *testing.T) {
	config := writeCLIConfig(t, executable(t, "false"), filepath.Join(t.TempDir(), "examples.txt"))
	exitCode := -1

	err := newApp(func(code int) { exitCode = code }).Run([]string{
		"rspec-sanity", "--config", config, "--skip-rerun", "run", "spec/models",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, exitCode)
}

func TestRunCommandReportsDetectedFlakies(t *testing.T) {
	dir := t.TempDir()
	persistenceFile := filepath.Join(dir, "examples.txt")
	scriptFile := filepath.Join(dir, "rspec")
	script := `#!/bin/sh
cat > "$1" <<EOF
example_id | status | run_time |
-----------|--------|----------|
spec/flaky_spec.rb[1:1] | $([ "$RSPEC_SANITY_ATTEMPT" = 1 ] && printf failed || printf passed) | 0.1 |
EOF
[ "$RSPEC_SANITY_ATTEMPT" = 1 ] && exit 1
exit 0
`
	require.NoError(t, os.WriteFile(scriptFile, []byte(script), 0o700))
	config := writeCLIConfig(t, fmt.Sprintf("%s %s", scriptFile, persistenceFile), persistenceFile)
	exitCode := -1

	err := newApp(func(code int) { exitCode = code }).Run([]string{
		"rspec-sanity", "--config", config, "run", "spec/models",
	})

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
}

func TestRunCommandValidatesArgumentsAndConfig(t *testing.T) {
	config := writeCLIConfig(t, executable(t, "true"), filepath.Join(t.TempDir(), "examples.txt"))

	err := newApp(func(int) {}).Run([]string{"rspec-sanity", "--config", config, "run"})
	assert.EqualError(t, err, "no test files or directories specified")

	err = newApp(func(int) {}).Run([]string{"rspec-sanity", "--config", "missing.toml", "run", "spec"})
	assert.Error(t, err)
}

func TestVerifyCommandUsesConfiguredReporter(t *testing.T) {
	config := writeCLIConfig(t, executable(t, "true"), filepath.Join(t.TempDir(), "examples.txt"))

	err := newApp(func(int) {}).Run([]string{"rspec-sanity", "--config", config, "verify"})
	assert.NoError(t, err)
}

func executable(t *testing.T, name string) string {
	t.Helper()
	path, err := exec.LookPath(name)
	require.NoError(t, err)
	return path
}

func writeCLIConfig(t *testing.T, command, persistenceFile string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	contents := fmt.Sprintf("command = %q\npersistence_file = %q\n", command, persistenceFile)
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return path
}
