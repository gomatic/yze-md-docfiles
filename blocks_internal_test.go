package docfiles

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsTitleCandidateRejectsEveryLineThatCannotBeATitle names the exclusions
// the doc comment claims are absolute. Each one is a line that sits exactly
// where a title sits — directly above a delimiter, with no blank line between —
// so admitting any of them turns the delimiter into an underline and the block
// it opened never opens.
func TestIsTitleCandidateRejectsEveryLineThatCannotBeATitle(t *testing.T) {
	t.Parallel()

	for name, text := range map[string]line{
		"blank":           "",
		"attribute":       "[source,go]",
		"block title":     ".Example listing",
		"dash delimiter":  "----",
		"equals delimter": "====",
		"dot delimiter":   "....",
		"comment block":   "////",
	} {
		assert.False(t, isTitleCandidate(text), "%s is not a title", name)
	}

	for name, text := range map[string]line{
		"ordinary words":  "Changelog",
		"leading dots":    "..a directive",
		"three dashes":    "---",
		"bracket midline": "see [1] for details",
	} {
		assert.True(t, isTitleCandidate(text), "%s could be a title", name)
	}
}

// TestWithoutCodeSpansRemovesOnlyClosedSpans names the CommonMark rule the doc
// comment states: a span closes on a backtick run of the same length, and a run
// that never closes is literal text. Getting the second half wrong would make
// every line holding one stray backtick disappear.
func TestWithoutCodeSpansRemovesOnlyClosedSpans(t *testing.T) {
	t.Parallel()

	for name, span := range map[string]struct{ in, want line }{
		"single":        {"a `code` b", "a  b"},
		"double":        {"a ``co`de`` b", "a  b"},
		"unclosed":      {"a ` b <!--", "a ` b <!--"},
		"uneven closer": {"a ``code` b", "a ``code` b"},
		"two spans":     {"`x` and `y`", " and "},
		"none":          {"plain text", "plain text"},
		"empty":         {"", ""},
	} {
		assert.Equal(t, span.want, withoutCodeSpans(span.in), "%s", name)
	}
}

// TestClosingRunReportsNotFoundForAnUnclosedSpan names the sentinel the doc
// comment promises. A span left open must be reported as absent rather than as
// an offset, because an offset would slice past the text it came from.
//
// The text given is what FOLLOWS the opening run, which is how withoutCodeSpans
// calls it — the offset returned is relative to that, not to the whole line.
func TestClosingRunReportsNotFoundForAnUnclosedSpan(t *testing.T) {
	t.Parallel()

	assert.Equal(t, notFound, closingRun(" never closed", 1))
	assert.Equal(t, notFound, closingRun(" only one ` here", 2), "a shorter run does not close a longer one")
	assert.Equal(t, offset(4), closingRun("code` rest", 1))
	assert.Equal(t, offset(4), closingRun("code`` rest", 2), "a run of the same length closes it")
}

// TestLeavesCommentOpenReadsTheLineOnce names the scan and the contradiction it
// replaced. Two separate questions — "is there an opener outside a code span?"
// and "is there a closer?" — were asked of DIFFERENT texts, so a comment opened
// and closed on one line stayed open forever while the identical span on a
// continuation line closed it.
func TestLeavesCommentOpenReadsTheLineOnce(t *testing.T) {
	t.Parallel()

	for name, text := range map[string]struct {
		in   line
		open bool
	}{
		"plain prose":         {"just words", false},
		"opened":              {"text <!--", true},
		"opened and closed":   {"<!-- a -->", false},
		"closed then opened":  {"<!-- a --> tail <!--", true},
		"closer in a span":    {"<!-- see `-->`", false},
		"opener in a span":    {"shown `<!--` only", false},
		"unclosed span first": {"stray ` then <!--", true},
		"empty":               {"", false},
	} {
		assert.Equal(t, text.open, leavesCommentOpen(text.in), "%s", name)
	}
}
