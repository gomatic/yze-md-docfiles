package docfiles_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	docfiles "github.com/gomatic/yze-md-docfiles"
)

// TestUnreadableCannotSkipAPathAndSaysItsNameFirst pins the two claims
// [docfiles.Unreadable]'s doc makes, which nothing else exercised: that a path
// the walk could not open is REPORTED rather than skipped, and that what the
// path's NAME says is reported alongside it, and FIRST.
//
// Both halves are load-bearing and neither is the other's consequence. Skipping
// is the failure the first claim exists to refuse — a path the gate cannot open
// is exactly where an unchecked one would hide, so a silent drop turns the one
// path that defeated the walk into the one path nobody hears about. Ordering is
// the second: a `CHANGELOG.md` behind a broken symlink is a changelog in the
// repository, and answering only "could not be read" answers a question nobody
// asked. A reordering that put the tool failure first would still emit both
// findings, so only an assertion on the ORDER catches it.
//
// The list mixes a banned name with an innocent one deliberately: the innocent
// path proves the walk failure alone is reported for a name that accuses
// nobody, and the banned one proves the name survives a reader that never saw
// a byte.
func TestUnreadableCannotSkipAPathAndSaysItsNameFirst(t *testing.T) {
	t.Parallel()

	const (
		banned   = "docs/CHANGELOG.md"
		innocent = "docs/locked"
	)

	diags := docfiles.Unreadable([]string{banned, innocent})

	require.Len(t, diags, 3, "the banned path yields its name and its failure; the innocent path yields its failure")

	assert.EqualValues(t, banned, diags[0].Path, "the banned path's NAME finding comes first")
	assert.Contains(t, diags[0].Message, "CHANGELOG.md must not be committed",
		"and it is the ban, not the tool failure")

	assert.EqualValues(t, banned, diags[1].Path, "its walk failure follows its name")
	assert.Contains(t, diags[1].Message, "cannot be analyzed as a document",
		"reported rather than skipped")

	assert.EqualValues(t, innocent, diags[2].Path, "and no path is dropped for accusing nobody")
	assert.Contains(t, diags[2].Message, "cannot be analyzed as a document")

	for i, diag := range diags {
		assert.Equal(t, 1, diag.Line, "diagnostic %d is anchored at the document's first line", i)
	}

	reported := map[string]bool{}
	for _, diag := range diags {
		reported[diag.Path] = true
	}
	assert.Equal(t, map[string]bool{banned: true, innocent: true}, reported,
		"every path handed in is answered for; none is skipped")
}

// TestUnreadableAnswersForEveryPathItIsHanded pins the skip half on its own, at
// a width no hand-written case list would cover, because "reported rather than
// skipped" is a claim about ALL paths and a three-element example is evidence
// about three. Every path here is innocent, so the only finding each can draw
// is its walk failure — which makes a dropped path a missing diagnostic rather
// than a changed one.
func TestUnreadableAnswersForEveryPathItIsHanded(t *testing.T) {
	t.Parallel()

	paths := []string{"a", "b/c", "d/e/f", "g.md", "h/i.markdown", "j/k/l/m"}

	diags := docfiles.Unreadable(paths)

	require.Len(t, diags, len(paths), "one finding per path, no path dropped and none invented")
	for i, path := range paths {
		assert.EqualValues(t, path, diags[i].Path, "path %d is answered for, in the order it was handed in", i)
		assert.Contains(t, diags[i].Message, "the walk could not read this path",
			"a path never attempted carries the walk's own reason, not a filesystem error")
	}
}
