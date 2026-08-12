package docfiles_test

// What the parse COSTS, not only what it answers.
//
// [docfiles.SizeLimit] admits eight megabytes, so every shape below is a file
// somebody could check in, and a rule that takes minutes over one is a rule
// whose gate gets disabled. The two backtick shapes were regression tests
// against this package's own hand-rolled code-span scanner; they are kept
// against the library that replaced it, because goldmark's inline parser
// reproduces the same superlinear blowup — a megabyte of `x` spans takes 43.9
// seconds through the full parser and 5 milliseconds through the block-only one
// this rule reads with (measured 2026-08-12). Deleting these would delete the
// only thing standing between the two.

import (
	"strings"
	"testing"
	"time"

	goyze "github.com/gomatic/go-yze"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	docfiles "github.com/gomatic/yze-md-docfiles"
)

// The budgets, which are two because the shapes below fail in two different
// ways and one number cannot catch both.
//
// A hostile LINE is superlinear or it is nothing: the shapes it covers measure
// 20 milliseconds here, 0.22 seconds under the race detector, and 43.9 seconds
// through the parser this rule declines to use. Ten seconds separates those by
// two orders of magnitude in each direction.
//
// A large DOCUMENT is merely large. Eight megabytes of banned headings is
// 645,277 nodes, and it measures 0.45 seconds — but the race detector
// instruments every one of those allocations and takes 8.6 seconds over it, on
// a runner already running the rest of the suite in parallel. A budget tight
// enough to be interesting for a line is a flake for a document, so this one is
// loose and still refuses the minutes a nesting blowup would cost.
const (
	hostileLineBudget   = 10 * time.Second
	largeDocumentBudget = 60 * time.Second
)

// TestAPathologicalLineIsNotSuperlinearInItsLength pins the COST on both shapes
// that made this package's own scanner quadratic, and the second is the one that
// matters, because it is what ordinary prose looks like: a line of code spans of
// equal length. A megabyte of it took 39 seconds in the hand-rolled scanner and
// 43.9 in goldmark's inline parser; the block grammar alone never looks at a
// backtick that does not open a fence.
func TestAPathologicalLineIsNotSuperlinearInItsLength(t *testing.T) {
	t.Parallel()

	for name, hostile := range map[string]string{
		"increasing runs": increasingRuns(1 << 20),
		"equal runs":      strings.Repeat("`x", 1<<19),
		"emphasis":        strings.Repeat("*a*_b_", (1<<20)/6),
	} {
		assert.Len(t, within(t, name, hostileLineBudget, "notes.md", hostile+"\n\n## Changelog\n\n"), 1,
			"%s: a line of punctuation opens no block, so the section after it is still read", name)
	}
}

// TestALargeOrdinaryDocumentIsReadInTime pins the shapes the size bound really
// admits: eight megabytes of prose, of blank lines, and of banned headings. The
// last is the one the per-document finding limit exists for, and the limit
// bounds the REPORT — this bounds the work.
func TestALargeOrdinaryDocumentIsReadInTime(t *testing.T) {
	t.Parallel()

	for name, huge := range map[string]string{
		"one long line":   strings.Repeat("x", 8<<20),
		"blank lines":     strings.Repeat("\n", 8<<20),
		"banned headings": strings.Repeat("## Changelog\n", (8<<20)/13),
	} {
		within(t, name, largeDocumentBudget, "notes.md", huge)
	}
}

// within runs the rule over one document and fails if it has not finished inside
// the budget. The analyzer cannot be cancelled, so the goroutine outlives a
// failure — which is why the failure is FATAL: a run that has blown its budget
// has nothing left to say, and letting the test continue would leave two
// pathological parses competing for the same machine.
func within(t *testing.T, name string, budget time.Duration, at, source string) []goyze.Diagnostic {
	t.Helper()
	type outcome struct {
		err   error
		diags []goyze.Diagnostic
	}
	done := make(chan outcome, 1)
	go func() {
		diags, err := docfiles.Diagnostics(docfiles.Path(at), docfiles.Source(source))
		done <- outcome{diags: diags, err: err}
	}()
	select {
	case got := <-done:
		require.NoError(t, got.err, name)
		return got.diags
	case <-time.After(budget):
		t.Fatalf("%s: reading one document did not finish within %s", name, budget)
		return nil
	}
}

// increasingRuns is a line of backtick runs of strictly increasing length, which
// closes none of them — the shape that made a forward-scanning code-span reader
// quadratic.
func increasingRuns(size int) string {
	var builder strings.Builder
	for run := 1; builder.Len() < size; run++ {
		_, _ = builder.WriteString(strings.Repeat("`", run) + "x")
	}
	return builder.String()
}
