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

// TestCodeSpansIsLinearInTheLineLength pins the COST, not just the answer. The
// spans used to be answered per position by scanning forward for a matching run,
// which rescans the whole line for every run that never closes — so a line of
// runs of strictly increasing length, which closes none of them, was superlinear:
// eight megabytes of it, a size the limit deliberately admits, took 27 seconds
// and reported nothing.
func TestCodeSpansIsLinearInTheLineLength(t *testing.T) {
	t.Parallel()
	var builder strings.Builder
	for run := 1; builder.Len() < 1<<20; run++ {
		_, _ = builder.WriteString(strings.Repeat("`", run) + "x")
	}
	hostile := line(builder.String())

	done := make(chan bool, 1)
	go func() { done <- leavesCommentOpen(hostile) }()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("scanning one line did not finish")
	}
}

// TestNextRunOfLengthMatchesOnlyAnExactLength pins the CommonMark rule directly:
// a span closes on a backtick string of EXACTLY the same length, so a longer or
// shorter run is not a closer and the span stays open.
func TestNextRunOfLengthMatchesOnlyAnExactLength(t *testing.T) {
	t.Parallel()
	runs := backtickRuns(line("`a``b`c```"))
	byLength := runsByLength(runs)

	closer, isClosed := nextRunOfLength(byLength, runs, 0)
	require.True(t, isClosed, "the one-backtick run closes on the next one-backtick run")
	assert.Equal(t, runIndex(2), closer, "skipping the two-backtick run between them")

	_, isTwoClosed := nextRunOfLength(byLength, runs, 1)
	assert.False(t, isTwoClosed, "and the two-backtick run finds no two-backtick closer")
}
