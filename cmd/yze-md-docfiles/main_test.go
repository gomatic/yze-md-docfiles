package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	docfiles "github.com/gomatic/yze-md-docfiles"
)

// swapStdout captures what the command writes, restoring the real writer after
// so tests cannot leak into one another.
func swapStdout(t *testing.T) *bytes.Buffer {
	t.Helper()
	original := stdout
	buf := &bytes.Buffer{}
	stdout = buf
	t.Cleanup(func() { stdout = original })
	return buf
}

// writeDoc puts a file at a path relative to dir, creating the parents.
func writeDoc(t *testing.T, dir, rel, content string) string {
	t.Helper()
	path := filepath.Join(dir, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

// banned is a document carrying a changelog section.
const banned = "## Changelog\n\n- a thing\n"

// TestRunEmitsReportForDirectory pins the ordinary invocation: a tree is walked
// and its findings reach stdout as the report the runner consumes.
func TestRunEmitsReportForDirectory(t *testing.T) {
	dir := t.TempDir()
	writeDoc(t, dir, "README.md", banned)
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{dir}))
	assert.Contains(t, buf.String(), "yze/docfiles")
	assert.Contains(t, buf.String(), "changelog")
}

// TestRunAcceptsExplicitFile pins that a named file is analyzed verbatim,
// without the discovery rules a directory walk applies.
func TestRunAcceptsExplicitFile(t *testing.T) {
	dir := t.TempDir()
	file := writeDoc(t, dir, "CHANGELOG.md", "")
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{file}))
	assert.Contains(t, buf.String(), "hand-maintained changelog")
}

// TestDocumentsAreDeduplicatedAcrossOverlappingArguments pins deduplication. Passing a
// directory and a file inside it is ordinary, and reporting one document three
// times tells its author there are three changelogs to delete.
func TestDocumentsAreDeduplicatedAcrossOverlappingArguments(t *testing.T) {
	dir := t.TempDir()
	file := writeDoc(t, dir, "sub/CHANGELOG.md", "")
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{dir, filepath.Join(dir, "sub"), file}))
	assert.Equal(t, 1, bytes.Count(buf.Bytes(), []byte("hand-maintained changelog")))
}

// TestRunFailsOnMissingPath pins that a path that does not exist is an error,
// not an empty success.
func TestRunFailsOnMissingPath(t *testing.T) {
	assert.Equal(t, 1, run([]string{filepath.Join(t.TempDir(), "absent.md")}))
}

// TestANamedNonRegularFileIsRefusedRatherThanRead pins that a FIFO or device
// named outright is an error carrying its own sentinel. It skips the walk's
// guard, and READING one hangs the gate forever instead of failing it — the
// single outcome nobody can diagnose from a stuck CI job.
func TestANamedNonRegularFileIsRefusedRatherThanRead(t *testing.T) {
	dir := t.TempDir()
	pipe := filepath.Join(dir, "notes.md")
	require.NoError(t, syscall.Mkfifo(pipe, 0o600))

	_, err := documents([]string{pipe})

	require.Error(t, err)
	assert.ErrorIs(t, err, docfiles.ErrNotRegularFile)
	assert.Equal(t, 1, run([]string{pipe}), "and the command reports the failure")
}

// TestRunFailsWhenReadErrors pins that a document the analyzer cannot read
// aborts the run — the gate never passes over a file it did not see.
func TestRunFailsWhenReadErrors(t *testing.T) {
	dir := t.TempDir()
	file := writeDoc(t, dir, "README.md", banned)
	original := readFile
	readFile = func(string) ([]byte, error) { return nil, errors.New("read failed") }
	t.Cleanup(func() { readFile = original })

	assert.Equal(t, 1, run([]string{file}))
}

// failingWriter refuses every write, standing in for a closed pipe.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

// TestRunFailsWhenEncodeErrors pins that a report which cannot be written is a
// failure: exiting zero would tell the runner a check passed when its result
// never arrived.
func TestRunFailsWhenEncodeErrors(t *testing.T) {
	dir := t.TempDir()
	file := writeDoc(t, dir, "README.md", banned)
	original := stdout
	stdout = failingWriter{}
	t.Cleanup(func() { stdout = original })

	assert.Equal(t, 1, run([]string{file}))
}

// TestMainExits pins the entry point's wiring: main runs the command and exits
// with its status rather than swallowing it.
func TestMainExits(t *testing.T) {
	original, originalArgs := osExit, os.Args
	code := -1
	osExit = func(status int) { code = status }
	os.Args = []string{"yze-md-docfiles", filepath.Join(t.TempDir(), "absent.md")}
	t.Cleanup(func() { osExit, os.Args = original, originalArgs })

	main()

	assert.Equal(t, 1, code)
}
