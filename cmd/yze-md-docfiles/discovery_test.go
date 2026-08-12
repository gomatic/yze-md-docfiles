package main

// What this command claims, and nothing else. The walk itself — the symlinked
// root, the identity of a path reached two ways, the tree that cannot be read,
// the size bound, the ignore filter — belongs to the shared discovery and is
// proven there, once, rather than three times in three ways that disagreed.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	goyze "github.com/gomatic/go-yze"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDiscoveryClaimsOnlyProse pins which files a walk reads: the extensions a
// changelog is written in, and the extensionless canonical spelling. Source
// code that manages the concept is not an instance of it.
//
// The `.rst` and `.adoc` documents are named `CHANGELOG` rather than carrying a
// banned section, because the analyzer judges those two by NAME alone — it does
// not parse them. Asserting on a section inside one would assert nothing about
// discovery: the file would be claimed, read, and correctly found silent, which
// is indistinguishable from never having been claimed at all.
func TestDiscoveryClaimsOnlyProse(t *testing.T) {
	dir := t.TempDir()
	writeDoc(t, dir, "notes.md", banned)
	writeDoc(t, dir, "CHANGELOG.rst", "")
	writeDoc(t, dir, "UPPER.MD", banned)
	writeDoc(t, dir, "CHANGELOG.adoc", "")
	writeDoc(t, dir, "guide.markdown", banned)
	writeDoc(t, dir, "notes.txt", banned)
	writeDoc(t, dir, "changelog.go", banned)
	writeDoc(t, dir, "image.png", banned)
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{dir}))
	out := buf.String()
	for _, claimed := range []string{
		"notes.md", "CHANGELOG.rst", "UPPER.MD", "CHANGELOG.adoc", "guide.markdown", "notes.txt",
	} {
		assert.Contains(t, out, claimed, "%s is prose", claimed)
	}
	assert.NotContains(t, out, "changelog.go", "code is not prose")
	assert.NotContains(t, out, "image.png")
}

// TestEveryProseExtensionIsClaimedByTheWalk pins the claim predicate directly,
// so an extension can be dropped from it without the finding it would have
// produced being the only thing that notices. The walk must claim every
// extension EITHER half of the rule judges — the section half parses four of
// them, and the name half judges two more that it never opens.
func TestEveryProseExtensionIsClaimedByTheWalk(t *testing.T) {
	for _, name := range []string{
		"notes.md", "notes.markdown", "notes.txt", "notes.rst", "notes.adoc",
		"NOTES.MD", "notes.RST",
	} {
		assert.True(t, isDocument(goyze.FilePath(filepath.Join("docs", name))),
			"%q is prose this rule reads", name)
	}

	for _, name := range []string{"changelog.go", "image.png", "Makefile", "notes.html"} {
		assert.False(t, isDocument(goyze.FilePath(filepath.Join("docs", name))),
			"%q is not", name)
	}
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

// TestEveryDotlessChangelogSpellingIsDiscovered pins the whole stem set rather
// than a sample. Six of the nine had nothing behind them: each could be dropped
// with the suite green, and each drop makes a real changelog file invisible to
// every invocation — a silent pass on the one thing this rule exists to find.
func TestEveryDotlessChangelogSpellingIsDiscovered(t *testing.T) {
	for _, stem := range []string{
		"CHANGELOG", "change-log", "change_log", "change log",
		"CHANGES", "releasenotes", "release-notes", "release_notes", "release notes",
	} {
		assert.True(t, isDocument(goyze.FilePath(filepath.Join("docs", stem))),
			"%q is a changelog whatever it is called next", stem)
	}

	for _, stem := range []string{"changelog-policy", "changes-guide", "readme", "notes"} {
		assert.False(t, isDocument(goyze.FilePath(filepath.Join("docs", stem))),
			"%q is a document about the subject, not the file", stem)
	}
}

// TestACommittedSymlinkNamedChangelogIsReportedOnce pins the escape the walk's
// two lists exist for, and the consumer decision that makes them worth having.
//
// `ln -s docs/versions.md CHANGELOG.md` is a committed entry — mode 120000,
// which survives a clone — and resolving it before judging deleted the evidence:
// the banned NAME never reached the rule, and the innocent document behind it
// reported whatever it held. So the name half judges every SPELLING and the
// section half reads every FILE, and this asserts both halves at once: the link
// earns exactly one ban, and its target's sections are reported exactly once
// rather than a second time under the second name.
func TestACommittedSymlinkNamedChangelogIsReportedOnce(t *testing.T) {
	dir := t.TempDir()
	writeDoc(t, dir, "docs/versions.md", banned)
	require.NoError(t, os.Symlink(filepath.Join(dir, "docs", "versions.md"), filepath.Join(dir, "CHANGELOG.md")))
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{dir}))
	out := buf.String()

	assert.Equal(t, 1, strings.Count(out, "must not be committed"),
		"the link's name is banned, once — the walk reaches it by one spelling")
	assert.Contains(t, out, "CHANGELOG.md must not be committed", "and the ban names the link, not its target")
	assert.Equal(t, 1, strings.Count(out, "section is a changelog"),
		"while the document behind it is read once, not once per name it has")
}

// TestADanglingSymlinkNamedChangelogIsStillBanned pins the shape that reaches
// neither list. A link resolving to nothing is not a file the walk can read and
// not a name it can hand to the section half, so it arrives as an unreadable
// PATH — and saying only that it could not be read answers a question nobody
// asked about an entry whose whole problem is its name.
//
// The consumer promotes it: every unreadable path is judged by name before it is
// reported as unreadable. That decision was taken for the size and encoding
// guards, where a name is knowable with zero bytes read, and this is the same
// rule reaching one step further out rather than a case handled specially.
func TestADanglingSymlinkNamedChangelogIsStillBanned(t *testing.T) {
	dir := t.TempDir()
	writeDoc(t, dir, "README.md", banned)
	require.NoError(t, os.Symlink(filepath.Join(dir, "absent.md"), filepath.Join(dir, "CHANGELOG.md")))
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{dir}))
	out := buf.String()

	assert.Contains(t, out, "CHANGELOG.md must not be committed", "the name is knowable with zero bytes read")
	assert.Contains(t, out, "cannot be analyzed as a document", "and the unreadable path is still reported")
	assert.Contains(t, out, "README.md", "while every other document keeps its findings")
}

// TestASymlinkToADirectoryNamedChangelogIsStillBanned pins the last spelling of
// the same entry. `ln -s docs CHANGELOG.md` is a committed link to a DIRECTORY,
// which the walk names rather than descends — so it appears in no list of files
// at all, and its name went unmentioned while every document behind it was
// reported under its own path.
func TestASymlinkToADirectoryNamedChangelogIsStillBanned(t *testing.T) {
	dir := t.TempDir()
	writeDoc(t, dir, "docs/guide.md", banned)
	require.NoError(t, os.Symlink(filepath.Join(dir, "docs"), filepath.Join(dir, "CHANGELOG.md")))
	buf := swapStdout(t)

	require.Equal(t, 0, run([]string{dir}))
	out := buf.String()

	assert.Contains(t, out, "CHANGELOG.md must not be committed", "the link's own name is a changelog")
	assert.Equal(t, 1, strings.Count(out, "section is a changelog"),
		"and the tree behind it is read once, under its own path")
}
