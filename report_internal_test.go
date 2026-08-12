package docfiles

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSectionFindingsContainsAReadFailureToItsOwnFile names the containment the
// doc comment claims is absolute: one file the gate cannot open yields exactly
// one finding, against that file, and never raises the whole run's error.
func TestSectionFindingsContainsAReadFailureToItsOwnFile(t *testing.T) {
	t.Parallel()

	found, held := sectionFindings(func(string) ([]byte, error) { return nil, os.ErrPermission }, "locked.md")

	require.Len(t, found, 1)
	assert.Equal(t, findingCount(1), held, "one unreadable file counts as one finding, not none")
	assert.Equal(t, "locked.md", found[0].Path)
	assert.Contains(t, found[0].Message, "cannot be analyzed as a document")
}

// TestTheSectionHalfNeverRepeatsWhatTheNameHalfSaid pins the seam between the
// walk's two lists. The name half judges NAMES and the section half reads FILES,
// so a `CHANGELOG.md` that cannot be opened must yield ONE finding here — the
// read failure — and get its ban from the other list. Both halves answering for
// the name is how one banned file becomes two findings, and every alias of it
// another two.
func TestTheSectionHalfNeverRepeatsWhatTheNameHalfSaid(t *testing.T) {
	t.Parallel()

	found, held := sectionFindings(func(string) ([]byte, error) { return nil, os.ErrPermission }, "docs/CHANGELOG.md")

	require.Len(t, found, 1)
	assert.Equal(t, findingCount(1), held)
	assert.Contains(t, found[0].Message, "cannot be analyzed as a document")
	assert.NotContains(t, found[0].Message, "must not be committed", "the name is the other list's question")
}

// TestReasonIsTheCauseOrUnopenableWhenThereIsNone names both arms of reason,
// which its doc comment claims are each reached, and the constant the first arm
// returns. Only one of them had anything behind it:
// every read and decode failure carries a cause, so the second arm is walked by
// most of this file's tests, while the first is reached only through Unreadable
// — the walk's own report of a path it did not attempt to open. Left
// unasserted, a nil cause would have rendered `%!s(<nil>)` where the reason
// belongs, and no test in the package would have seen it.
func TestReasonIsTheCauseOrUnopenableWhenThereIsNone(t *testing.T) {
	t.Parallel()

	assert.Equal(t, unopenable, reason(nil), "a path never opened has no error to quote")
	assert.Equal(t, os.ErrPermission.Error(), reason(os.ErrPermission), "and a real cause is quoted verbatim")
	assert.NotContains(t, reason(nil), "%!", "least of all a formatting verb")
}

// TestRunTruncationAlwaysCarriesAPath names the property the doc comment states:
// a diagnostic with an empty path is one the runner cannot attribute, baseline
// or ratchet, so this one is attributed to the file its run stopped at.
func TestRunTruncationAlwaysCarriesAPath(t *testing.T) {
	t.Parallel()

	diag := runTruncation("docs/CHANGELOG.md", 12_345, 9_991)

	assert.Equal(t, "docs/CHANGELOG.md", diag.Path)
	assert.Equal(t, Rule, diag.Rule)
	assert.Contains(t, diag.Message, "12345")
	assert.Contains(t, diag.Message, "9991 diagnostics are carried",
		"what the report CARRIES, not the limit — a run past its bound carries truncation notices too, "+
			"and naming the constant said 10000 over a report holding 9991 findings and nine notices")
	assert.Contains(t, diag.Message, "docs/CHANGELOG.md")
}

// TestTheRunTotalIsTheDocumentsCountNotTheReportedOne pins the number the
// truncation notice names. A document past its OWN limit hands back a truncated
// slice, so a run summing slices counted this analyzer's truncation instead of
// the documents' findings — it announced 11011 over a tree holding 16500, in the
// very sentence that says the true count is named.
func TestTheRunTotalIsTheDocumentsCountNotTheReportedOne(t *testing.T) {
	t.Parallel()
	crowded := strings.Repeat("## Changelog\n\n", int(findingLimit)+500)

	_, held := sectionFindings(func(string) ([]byte, error) { return []byte(crowded), nil }, "many.md")

	assert.Equal(t, findingCount(int(findingLimit)+500), held,
		"every heading the document holds, not the thousand that fit in the report")
}
