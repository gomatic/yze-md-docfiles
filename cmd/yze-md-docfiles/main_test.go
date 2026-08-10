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

// TestResolvedRootNeverAnswersASymlinkedDirectoryWithNothing pins that a deliberate request is
// answered. The walk lstats its own root, so a symlink to a directory reported
// itself as a non-directory and the walk yielded NOTHING — a silent clean pass,
// which is the one result a gate must never invent.
func TestResolvedRootNeverAnswersASymlinkedDirectoryWithNothing(t *testing.T) {
	base := t.TempDir()
	writeDoc(t, base, "tree/CHANGELOG.md", "")
	link := filepath.Join(base, "linkdir")
	require.NoError(t, os.Symlink(filepath.Join(base, "tree"), link))
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{link}))
	assert.Contains(t, buf.String(), "CHANGELOG.md")
}

// TestResolvedRootKeepsAnOrdinaryPathAsGiven pins the other half: only a
// SYMLINKED root is resolved. Resolving every root rewrote ordinary paths too —
// on this platform a temp directory sits under a symlinked prefix — so one
// document reached by two arguments was reported under two names.
func TestResolvedRootKeepsAnOrdinaryPathAsGiven(t *testing.T) {
	dir := t.TempDir()
	file := writeDoc(t, dir, "CHANGELOG.md", "")
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{dir, file}))
	assert.Equal(t, 1, bytes.Count(buf.Bytes(), []byte("hand-maintained changelog")),
		"the walked path and the named path are the same document")
}

// TestResolvedRootFallsBackWhenASymlinkCannotBeResolved pins the arm that keeps
// an unresolvable root from becoming a crash or a silent pass: the walk
// proceeds with the path as given, so whatever the walk itself can reach is
// still reported rather than the whole argument being dropped.
func TestResolvedRootFallsBackWhenASymlinkCannotBeResolved(t *testing.T) {
	base := t.TempDir()
	writeDoc(t, base, "tree/CHANGELOG.md", "")
	link := filepath.Join(base, "linkdir")
	require.NoError(t, os.Symlink(filepath.Join(base, "tree"), link))

	original := evalSymlinks
	evalSymlinks = func(string) (string, error) { return "", errors.New("cannot resolve") }
	t.Cleanup(func() { evalSymlinks = original })
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{link}), "an unresolvable root is walked as given, not abandoned")
	assert.NotContains(t, buf.String(), "CHANGELOG.md",
		"the walk lstats that root and finds a symlink, which is exactly why resolving it matters")
}

// TestWithinSizeLimitRefusesBeforeOpening pins the bound that actually bounds.
// Asking after the read was no bound at all — a 2 GiB document cost 4.3 GB
// resident, its own size to read and again to convert, before the limit that
// refused it was ever consulted. The size comes from the directory entry.
func TestWithinSizeLimitRefusesBeforeOpening(t *testing.T) {
	dir := t.TempDir()
	writeDoc(t, dir, "small.md", banned)
	huge := filepath.Join(dir, "huge.md")
	require.NoError(t, os.WriteFile(huge, nil, 0o600))
	require.NoError(t, os.Truncate(huge, docfiles.SizeLimit+1))
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{dir}))
	out := buf.String()
	assert.Contains(t, out, "small.md")
	assert.NotContains(t, out, "huge.md", "never opened, so never read and never reported")
}

// TestWithinSizeLimitFallsBackToStat pins the arm taken when the walk cannot
// describe an entry: the file is measured directly, and if it cannot be
// measured at all it is READ rather than dropped in silence — a gate must not
// skip a file because it could not size it.
func TestWithinSizeLimitFallsBackToStat(t *testing.T) {
	dir := t.TempDir()
	small := writeDoc(t, dir, "notes.md", banned)

	assert.True(t, withinSizeLimit(entryPath(small), entryWithoutInfo{name: "notes.md"}),
		"measured by stat when the entry cannot describe itself")

	original := statPath
	statPath = func(string) (os.FileInfo, error) { return nil, errors.New("cannot stat") }
	t.Cleanup(func() { statPath = original })

	assert.True(t, withinSizeLimit(entryPath(small), entryWithoutInfo{name: "notes.md"}),
		"and read rather than silently skipped when it cannot be measured at all")
}

// TestOneDocumentReachedTwoWaysIsReportedOnce pins that identity is the file.
func TestOneDocumentReachedTwoWaysIsReportedOnce(t *testing.T) {
	dir := t.TempDir()
	target := writeDoc(t, dir, "real/CHANGELOG.md", "")
	link := filepath.Join(dir, "link.md")
	require.NoError(t, os.Symlink(target, link))
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{target, link}))
	assert.Equal(t, 1, bytes.Count(buf.Bytes(), []byte("hand-maintained changelog")))
}

// TestANamedDocumentIsAnalyzedEvenWhenGitIgnoresIt pins the filter's scope: it
// stops a WALK claiming files the repository does not own, and does not
// overrule an author who asked about one outright.
func TestANamedDocumentIsAnalyzedEvenWhenGitIgnoresIt(t *testing.T) {
	dir := t.TempDir()
	named := writeDoc(t, dir, "var/CHANGELOG.md", "")

	original := checkIgnore
	checkIgnore = func(repoDir, []string) (map[string]bool, error) {
		return map[string]bool{named: true}, nil
	}
	t.Cleanup(func() { checkIgnore = original })
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{named}))
	assert.Contains(t, buf.String(), "CHANGELOG.md")
}

// TestCanonicalFallsBackToTheSpelling pins that a path which cannot be resolved
// keeps its own spelling as its identity, so the document is still analyzed
// rather than dropped for being unidentifiable.
func TestCanonicalFallsBackToTheSpelling(t *testing.T) {
	dir := t.TempDir()
	file := writeDoc(t, dir, "CHANGELOG.md", "")

	original := evalSymlinks
	evalSymlinks = func(string) (string, error) { return "", errors.New("cannot resolve") }
	t.Cleanup(func() { evalSymlinks = original })
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{file}))
	assert.Contains(t, buf.String(), "CHANGELOG.md")
}
