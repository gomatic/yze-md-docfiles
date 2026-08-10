package docfiles

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileFindingsContainsAReadFailureToItsOwnFile names the containment the
// doc comment claims is absolute: one file the gate cannot open yields exactly
// one finding, against that file, and never raises the whole run's error.
func TestFileFindingsContainsAReadFailureToItsOwnFile(t *testing.T) {
	t.Parallel()

	found, held := fileFindings(func(string) ([]byte, error) { return nil, os.ErrPermission }, "locked.md")

	require.Len(t, found, 1)
	assert.Equal(t, findingCount(1), held, "one unreadable file counts as one finding, not none")
	assert.Equal(t, "locked.md", found[0].Path)
	assert.Contains(t, found[0].Message, "cannot be analyzed as a document")
}

// TestRunTruncationAlwaysCarriesAPath names the property the doc comment states:
// a diagnostic with an empty path is one the runner cannot attribute, baseline
// or ratchet, so this one is attributed to the file its run stopped at.
func TestRunTruncationAlwaysCarriesAPath(t *testing.T) {
	t.Parallel()

	diag := runTruncation("docs/CHANGELOG.md", 12_345)

	assert.Equal(t, "docs/CHANGELOG.md", diag.Path)
	assert.Equal(t, Rule, diag.Rule)
	assert.Contains(t, diag.Message, "12345")
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

	_, held := fileFindings(func(string) ([]byte, error) { return []byte(crowded), nil }, "many.md")

	assert.Equal(t, findingCount(int(findingLimit)+500), held,
		"every heading the document holds, not the thousand that fit in the report")
}
