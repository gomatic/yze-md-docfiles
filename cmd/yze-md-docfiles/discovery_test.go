package main

// What this command claims, and nothing else. The walk itself — the symlinked
// root, the identity of a path reached two ways, the tree that cannot be read,
// the size bound, the ignore filter — belongs to the shared discovery and is
// proven there, once, rather than three times in three ways that disagreed.

import (
	"path/filepath"
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
