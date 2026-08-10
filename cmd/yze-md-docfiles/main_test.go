package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	goyze "github.com/gomatic/go-yze"
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

// TestANamedDocumentIsAnalyzedEvenWhenGitIgnoresIt pins the filter's scope: it
// stops a WALK claiming files the repository does not own, and does not
// overrule an author who asked about one outright.
func TestANamedDocumentIsAnalyzedEvenWhenGitIgnoresIt(t *testing.T) {
	dir := t.TempDir()
	named := writeDoc(t, dir, "var/CHANGELOG.md", "")

	original := checkIgnore
	checkIgnore = func(goyze.RepoDir, []string) (map[string]bool, error) {
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

// TestOneDocumentReachedByTwoSpellingsIsReportedOnce pins that identity is the
// FILE. EvalSymlinks resolves links without absolutising, so `CHANGELOG.md` and
// `$PWD/CHANGELOG.md` were two identities for one document — and the symlink
// half of the rule was pinned by nothing, because the link in the old test was
// named `link.md`, which can never produce a file finding at all.
func TestOneDocumentReachedByTwoSpellingsIsReportedOnce(t *testing.T) {
	dir := t.TempDir()
	target := writeDoc(t, dir, "CHANGELOG.md", "")
	link := filepath.Join(dir, "CHANGELOG-link.md")
	require.NoError(t, os.Symlink(target, link))

	relative, err := filepath.Rel(dir, target)
	require.NoError(t, err)
	t.Chdir(dir)
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{target, relative, "./" + relative}))
	assert.Equal(t, 1, bytes.Count(buf.Bytes(), []byte("hand-maintained changelog")),
		"one file, however it is spelled")
}

// TestANamedDocumentIsNotFilteredWhateverTheArgumentOrder pins the rule against
// the ordering that broke it: a directory listed first claimed the file, and
// the ignore filter then deleted the author's explicit request.
func TestANamedDocumentIsNotFilteredWhateverTheArgumentOrder(t *testing.T) {
	dir := t.TempDir()
	ignored := writeDoc(t, dir, "var/CHANGELOG.md", "")

	original := checkIgnore
	checkIgnore = func(goyze.RepoDir, []string) (map[string]bool, error) {
		return map[string]bool{ignored: true}, nil
	}
	t.Cleanup(func() { checkIgnore = original })

	for name, args := range map[string][]string{
		"directory first": {filepath.Join(dir, "var"), ignored},
		"named first":     {ignored, filepath.Join(dir, "var")},
	} {
		buf := swapStdout(t)
		require.Equal(t, 0, run(args), name)
		assert.Contains(t, buf.String(), "CHANGELOG.md", "%s: the author asked for it by name", name)
	}
}

// TestCanonicalFallsBackWhenAPathCannotBeMadeAbsolute pins the first arm of
// identity: a path the working directory makes unresolvable keeps its own
// spelling, so the document is still analyzed rather than dropped for being
// unidentifiable.
func TestCanonicalFallsBackWhenAPathCannotBeMadeAbsolute(t *testing.T) {
	dir := t.TempDir()
	file := writeDoc(t, dir, "CHANGELOG.md", "")

	original := absolutePath
	absolutePath = func(string) (string, error) { return "", os.ErrInvalid }
	t.Cleanup(func() { absolutePath = original })
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{file}))
	assert.Contains(t, buf.String(), "CHANGELOG.md", "still analyzed, never dropped for being unidentifiable")
}
