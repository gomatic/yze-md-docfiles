package docfiles

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestCodeSpansFindsEveryClosedSpanAndNoOpenOne names nextRunOfLength and pins
// the CommonMark rule that
// makes hiding a comment opener safe: a span closes on a backtick run of exactly
// the same length, and a run that never closes is literal text.
func TestCodeSpansFindsEveryClosedSpanAndNoOpenOne(t *testing.T) {
	t.Parallel()

	for name, span := range map[string]struct {
		spans map[int]width
		text  string
	}{
		"single":        {map[int]width{2: 6}, "a `code` b"},
		"double":        {map[int]width{2: 9}, "a ``co`de`` b"},
		"unclosed":      {map[int]width{}, "a ` b"},
		"uneven closer": {map[int]width{}, "a ``code` b"},
		"two spans":     {map[int]width{0: 3, 4: 3}, "`x` `y`"},
		"none":          {map[int]width{}, "plain"},
		"empty":         {map[int]width{}, ""},
	} {
		assert.Equal(t, span.spans, codeSpans(line(span.text)), "%s", name)
	}
}

// TestCodeSpansIsLinearInTheLineLength pins the COST, not just the answer, on
// BOTH shapes that made it superlinear — and the second is the one that matters,
// because it is what ordinary prose looks like.
//
// Answering per position by scanning forward rescans the whole line for every
// run that never closes, so runs of strictly increasing length were superlinear:
// eight megabytes took 27 seconds and reported nothing.
//
// Indexing the runs by length fixed that shape and not the other one. A line
// that is nothing but code spans has every run at the SAME length, so the
// openers sit at candidate positions 0, 2, 4, … and each restarted from the
// front of that length's list — a megabyte took 39 seconds, 112 times slower
// than the increasing-length line at half the size. A test named for linearity
// pinned the old bug's absence rather than the property in its name.
func TestCodeSpansIsLinearInTheLineLength(t *testing.T) {
	t.Parallel()

	for name, hostile := range map[string]line{
		"increasing runs": increasingRuns(1 << 20),
		"equal runs":      line(strings.Repeat("`x", 1<<19)),
	} {
		done := make(chan bool, 1)
		go func() { done <- leavesCommentOpen(hostile) }()

		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatalf("%s: scanning one line did not finish", name)
		}
	}
}

// increasingRuns is a line of backtick runs of strictly increasing length, which
// closes none of them.
func increasingRuns(size int) line {
	var builder strings.Builder
	for run := 1; builder.Len() < size; run++ {
		_, _ = builder.WriteString(strings.Repeat("`", run) + "x")
	}
	return line(builder.String())
}

// TestTheCloserMatchesOnlyAnExactLength pins the CommonMark rule directly:
// a span closes on a backtick string of EXACTLY the same length, so a longer or
// shorter run is not a closer and the span stays open.
func TestTheCloserMatchesOnlyAnExactLength(t *testing.T) {
	t.Parallel()
	runs := backtickRuns(line("`a``b`c```"))
	byLength := runsByLength(runs)

	closer, isClosed := searching(byLength).after(runs, 0)
	require.True(t, isClosed, "the one-backtick run closes on the next one-backtick run")
	assert.Equal(t, runIndex(2), closer, "skipping the two-backtick run between them")

	_, isTwoClosed := searching(byLength).after(runs, 1)
	assert.False(t, isTwoClosed, "and the two-backtick run finds no two-backtick closer")
}
