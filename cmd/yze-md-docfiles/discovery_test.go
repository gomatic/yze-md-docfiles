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

	docfiles "github.com/gomatic/yze-md-docfiles"
)

// TestDiscoveryClaimsOnlyProse pins which files a walk reads: the extensions a
// changelog is written in, and the extensionless canonical spelling. Source
// code that manages the concept is not an instance of it.
func TestDiscoveryClaimsOnlyProse(t *testing.T) {
	dir := t.TempDir()
	writeDoc(t, dir, "notes.md", banned)
	writeDoc(t, dir, "notes.rst", banned)
	writeDoc(t, dir, "UPPER.MD", banned)
	writeDoc(t, dir, "guide.adoc", banned)
	writeDoc(t, dir, "guide.markdown", banned)
	writeDoc(t, dir, "changelog.go", banned)
	writeDoc(t, dir, "image.png", banned)
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{dir}))
	out := buf.String()
	for _, claimed := range []string{"notes.md", "notes.rst", "UPPER.MD", "guide.adoc", "guide.markdown"} {
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

// entryWithoutInfo is a directory entry the walk cannot describe.
type entryWithoutInfo struct{ name string }

func (e entryWithoutInfo) Name() string             { return e.name }
func (entryWithoutInfo) IsDir() bool                { return false }
func (entryWithoutInfo) Type() fs.FileMode          { return 0 }
func (entryWithoutInfo) Info() (fs.FileInfo, error) { return nil, errors.New("no info") }

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
