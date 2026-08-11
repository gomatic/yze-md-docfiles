package main

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	goyze "github.com/gomatic/go-yze"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	docfiles "github.com/gomatic/yze-md-docfiles"
)

// TestMain neutralises the one seam that reaches outside this process. The real
// ignore filter runs `git check-ignore`, so every test here spawned git —
// twenty-four processes across the package — and the filter's verdict then
// depended on whatever repository the temporary directory happened to sit in.
// It fails open, so nothing was flaky; it was simply never exercised
// deterministically, and a unit test that shells out is an integration test
// wearing one's name.
func TestMain(m *testing.M) {
	files.CheckIgnore = func(goyze.RepoDir, []string) (map[string]bool, error) {
		return map[string]bool{}, nil
	}
	os.Exit(m.Run())
}

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

// TestRunFailsOnMissingPath pins that a path that does not exist is an error,
// not an empty success.
func TestRunFailsOnMissingPath(t *testing.T) {
	assert.Equal(t, 1, run([]string{filepath.Join(t.TempDir(), "absent.md")}))
}

// TestARunWithNoPathsIsAFailure pins that being given nothing is an error, not a
// clean pass. A runner whose root placeholder expands to nothing would otherwise
// green the gate over a repository no analyzer ever looked at — indistinguishable,
// at the exit code, from one it read and found clean.
func TestARunWithNoPathsIsAFailure(t *testing.T) {
	assert.Equal(t, 1, run(nil))
	// The error the CODE produces, not one the assertion builds. Asserting that
	// `ErrNoPaths.With(nil)` matches `ErrNoPaths` exercises the error helper and
	// holds whatever run returns — swapping this failure's sentinel for any
	// other left the whole suite green.
	assert.ErrorIs(t, report(nil), docfiles.ErrNoPaths)
	assert.ErrorIs(t, report([]string{}), docfiles.ErrNoPaths)
}

// TestAnUnreadableTreeIsReportedRatherThanLost pins that the command ATTACHES
// what the walk could not enter. The shared discovery hands those back; a
// command that dropped them on the floor would pass every other test while
// losing exactly the tree an unchecked document hides in.
func TestAnUnreadableTreeIsReportedRatherThanLost(t *testing.T) {
	dir := t.TempDir()
	writeDoc(t, dir, "README.md", banned)
	locked := filepath.Join(dir, "locked")
	require.NoError(t, os.Mkdir(locked, 0o750))
	original := files.WalkDir
	files.WalkDir = deniedTree(locked)
	t.Cleanup(func() { files.WalkDir = original })
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{dir}))
	out := buf.String()
	assert.Contains(t, out, "locked", "the tree is reported rather than passed over")
	assert.Contains(t, out, "changelog", "and every other document keeps its findings")
}

// deniedTree walks for real but reports one path as unreadable.
//
// The failure is injected through the SEAM rather than with chmod, because a
// test that removes read permission does not test what it says on a machine
// whose user ignores permissions: the gate's own CI container runs as ROOT.
func deniedTree(at string) func(string, fs.WalkDirFunc) error {
	return func(root string, walk fs.WalkDirFunc) error {
		return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if path == at {
				return walk(path, entry, os.ErrPermission)
			}
			return walk(path, entry, err)
		})
	}
}

// TestANamedNonRegularFileIsRefusedRatherThanRead pins that a FIFO or device
// named outright is an error carrying its own sentinel. It skips the walk's
// guard, and READING one hangs the gate forever instead of failing it — the
// single outcome nobody can diagnose from a stuck CI job.
func TestANamedNonRegularFileIsRefusedRatherThanRead(t *testing.T) {
	dir := t.TempDir()
	pipe := filepath.Join(dir, "notes.md")
	require.NoError(t, syscall.Mkfifo(pipe, 0o600))

	_, err := discovery().Expand([]string{pipe})

	require.Error(t, err)
	assert.ErrorIs(t, err, goyze.ErrNotRegularFile)
	assert.Equal(t, 1, run([]string{pipe}), "and the command reports the failure")
}

// TestRunReportsAnUnreadableDocumentWithoutFailingTheRun pins the containment
// at the command's edge: the document becomes a finding of its own and the run
// continues, so a single locked file cannot empty the report.
func TestRunReportsAnUnreadableDocumentWithoutFailingTheRun(t *testing.T) {
	dir := t.TempDir()
	file := writeDoc(t, dir, "README.md", banned)
	original := readFile
	readFile = func(string) ([]byte, error) { return nil, errors.New("read failed") }
	t.Cleanup(func() { readFile = original })
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{file}))
	assert.Contains(t, buf.String(), "cannot be analyzed as a document")
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
