package docfiles

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileFindingsContainsAReadFailureToItsOwnFile names the containment the
// doc comment claims is absolute: one file the gate cannot open yields exactly
// one finding, against that file, and never raises the whole run's error.
func TestFileFindingsContainsAReadFailureToItsOwnFile(t *testing.T) {
	t.Parallel()

	found := fileFindings(func(string) ([]byte, error) { return nil, os.ErrPermission }, "locked.md")

	require.Len(t, found, 1)
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
