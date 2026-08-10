package main

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDiscoveryClaimsOnlyProse pins which files a walk reads: the extensions a
// changelog is written in, and the extensionless canonical spelling. Source
// code that manages the concept is not an instance of it.
func TestDiscoveryClaimsOnlyProse(t *testing.T) {
	dir := t.TempDir()
	writeDoc(t, dir, "notes.md", banned)
	// Each in the spelling ITS OWN family uses: reStructuredText has no
	// marker-run heading, so writing `## Changelog` there proved the file was
	// claimed only for as long as one pattern was wrongly shared by all three.
	writeDoc(t, dir, "notes.rst", "Changelog\n=========\n")
	writeDoc(t, dir, "UPPER.MD", banned)
	writeDoc(t, dir, "guide.adoc", "== Changelog\n")
	writeDoc(t, dir, "guide.markdown", banned)
	writeDoc(t, dir, "notes.txt", banned)
	writeDoc(t, dir, "changelog.go", banned)
	writeDoc(t, dir, "image.png", banned)
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{dir}))
	out := buf.String()
	for _, claimed := range []string{"notes.md", "notes.rst", "UPPER.MD", "guide.adoc", "guide.markdown", "notes.txt"} {
		assert.Contains(t, out, claimed, "%s is prose", claimed)
	}
	assert.NotContains(t, out, "changelog.go", "code is not prose")
	assert.NotContains(t, out, "image.png")
}

// TestDiscoveryClaimsTheExtensionlessChangelog pins the canonical Unix
// spelling. It is the commonest form of the very thing this rule bans, and
// requiring an extension made it invisible to every real invocation while the
// unit tests advertised it as covered.
func TestDiscoveryClaimsTheExtensionlessChangelog(t *testing.T) {
	dir := t.TempDir()
	writeDoc(t, dir, "CHANGELOG", "")
	writeDoc(t, dir, "CHANGES", "")
	writeDoc(t, dir, "Makefile", banned)
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{dir}))
	out := buf.String()
	assert.Contains(t, out, "CHANGELOG")
	assert.Contains(t, out, "CHANGES")
	assert.NotContains(t, out, "Makefile", "an extensionless file that is not a changelog is not prose")
}

// TestDiscoverySkipsSomebodyElsesProse pins the pruning: a dependency's own
// changelog is its business, and reporting it tells this repository to delete a
// file it does not own. A Hugo theme and a nested checkout are somebody else's
// too, and `testdata` is where this family proves its analyzers in BOTH
// directions — a fixture that must contain a violation is not one.
func TestDiscoverySkipsSomebodyElsesProse(t *testing.T) {
	dir := t.TempDir()
	writeDoc(t, dir, "README.md", banned)
	writeDoc(t, dir, "node_modules/dep/CHANGELOG.md", "")
	writeDoc(t, dir, "vendor/dep/CHANGELOG.md", "")
	writeDoc(t, dir, "themes/paper/CHANGELOG.md", "")
	writeDoc(t, dir, "testdata/fixture.md", banned)
	writeDoc(t, dir, ".venv/lib/pkg/CHANGELOG.md", "")
	writeDoc(t, dir, ".git/notes.md", banned)
	writeDoc(t, dir, "submodule/.git/HEAD", "ref: refs/heads/main\n")
	writeDoc(t, dir, "submodule/CHANGELOG.md", "")
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{dir}))
	out := buf.String()
	assert.Contains(t, out, "README.md")
	for _, skipped := range []string{"node_modules", "vendor", "themes", "testdata", ".git/", "submodule", ".venv"} {
		assert.NotContains(t, out, skipped, "%s is not this repository's prose", skipped)
	}
}

// TestPrunedDirNeverPrunesTheWalkRoot pins that naming a pruned directory
// outright still analyzes it. Applying the prune list to the walk root answered
// a deliberate request with a silent clean pass, which is the one result a gate
// must never invent.
func TestPrunedDirNeverPrunesTheWalkRoot(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "node_modules")
	writeDoc(t, dir, "CHANGELOG.md", "")
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{dir}))
	assert.Contains(t, buf.String(), "CHANGELOG.md")
}

// TestDiscoveryFollowsASymlinkToADocument pins that a linked document is read.
// The walk reports the LINK's mode, so testing that dropped the file the author
// actually wrote — silently, which is the part that matters.
func TestDiscoveryFollowsASymlinkToADocument(t *testing.T) {
	dir := t.TempDir()
	target := writeDoc(t, dir, "real/NOTES.md", banned)
	require.NoError(t, os.Symlink(target, filepath.Join(dir, "link.md")))
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{dir}))
	assert.Contains(t, buf.String(), "link.md")
}

// TestDiscoverySkipsNonRegularFilesInATree pins the other half of that guard:
// one FIFO in a scanned tree must not hang the whole run.
func TestDiscoverySkipsNonRegularFilesInATree(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, syscall.Mkfifo(filepath.Join(dir, "pipe.md"), 0o600))
	writeDoc(t, dir, "README.md", banned)
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{dir}))
	out := buf.String()
	assert.Contains(t, out, "README.md")
	assert.NotContains(t, out, "pipe.md")
}

// TestRunFailsWhenWalkErrors pins that a walk failure aborts rather than
// reporting whatever it collected first.
func TestRunFailsWhenWalkErrors(t *testing.T) {
	original := walkDir
	walkDir = func(_ string, _ fs.WalkDirFunc) error { return errors.New("walk failed") }
	t.Cleanup(func() { walkDir = original })

	assert.Equal(t, 1, run([]string{t.TempDir()}))
}

// TestRunFailsWhenTheWalkCallbackErrors pins the other half of walk failure: an
// entry the walk could not stat aborts the run rather than being skipped.
func TestRunFailsWhenTheWalkCallbackErrors(t *testing.T) {
	original := walkDir
	walkDir = func(root string, fn fs.WalkDirFunc) error {
		return fn(root, nil, errors.New("entry failed"))
	}
	t.Cleanup(func() { walkDir = original })

	assert.Equal(t, 1, run([]string{t.TempDir()}))
}

// TestEverySpellingOfOnePathIsOneDocument pins that deduplication normalises.
// Keyed on the raw string, `a.md`, `./a.md` and `sub/../a.md` were three
// documents, telling an author there were three changelogs to delete.
func TestEverySpellingOfOnePathIsOneDocument(t *testing.T) {
	dir := t.TempDir()
	writeDoc(t, dir, "sub/CHANGELOG.md", "")
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{
		filepath.Join(dir, "sub", "CHANGELOG.md"),
		dir + "/sub/./CHANGELOG.md",
		dir + "/sub/../sub/CHANGELOG.md",
		filepath.Join(dir, "sub") + string(filepath.Separator),
	}))
	assert.Equal(t, 1, bytes.Count(buf.Bytes(), []byte("hand-maintained changelog")))
}

// TestDocumentsAreReportedInTheOrderTheyWereFound pins the ordering the
// deduplication must preserve: a report that reorders findings makes two runs
// over the same tree produce different output.
func TestDocumentsAreReportedInTheOrderTheyWereFound(t *testing.T) {
	dir := t.TempDir()
	first := writeDoc(t, dir, "a.md", banned)
	second := writeDoc(t, dir, "b.md", banned)
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{first, second}))
	assert.Less(t, bytes.Index(buf.Bytes(), []byte("a.md")), bytes.Index(buf.Bytes(), []byte("b.md")))
}

// TestDiscoverySkipsASymlinkToAFifo pins that following a link means following
// it to what it POINTS AT: a link to a FIFO blocks on open exactly as a bare
// one does, and the guard has to see through the link to catch it.
func TestDiscoverySkipsASymlinkToAFifo(t *testing.T) {
	dir := t.TempDir()
	pipe := filepath.Join(dir, "pipe")
	require.NoError(t, syscall.Mkfifo(pipe, 0o600))
	require.NoError(t, os.Symlink(pipe, filepath.Join(dir, "link.md")))
	writeDoc(t, dir, "README.md", banned)
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{dir}))
	out := buf.String()
	assert.Contains(t, out, "README.md")
	assert.NotContains(t, out, "link.md")
}

// TestAReportedPathIsCleaned pins the spelling an author is shown. A path
// reaching the report with `..` still in it names a file correctly and reads as
// a different one, and the same document named two ways used to be reported
// twice.
func TestAReportedPathIsCleaned(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0o750))
	writeDoc(t, dir, "CHANGELOG.md", banned)
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{dir + "/sub/../CHANGELOG.md"}))

	assert.NotContains(t, buf.String(), "sub", "the path is cleaned before it is reported")
	assert.Contains(t, buf.String(), "CHANGELOG.md")
}
