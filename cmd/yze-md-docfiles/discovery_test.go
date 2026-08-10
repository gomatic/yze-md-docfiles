package main

// What this command claims, and nothing else. The walk itself — the symlinked
// root, the identity of a path reached two ways, the tree that cannot be read,
// the size bound, the ignore filter — belongs to the shared discovery and is
// proven there, once, rather than three times in three ways that disagreed.

import (
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
