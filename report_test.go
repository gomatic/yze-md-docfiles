package docfiles_test

import (
	"testing"

	errs "github.com/gomatic/go-error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	docfiles "github.com/gomatic/yze-md-docfiles"
)

// errUnreadable stands in for whatever the filesystem refuses with.
const errUnreadable errs.Const = "unreadable"

// reader serves file contents from a map, refusing anything absent.
func reader(files map[string]string) docfiles.FileReader {
	return func(path string) ([]byte, error) {
		data, ok := files[path]
		if !ok {
			return nil, errUnreadable
		}
		return []byte(data), nil
	}
}

// TestReportAggregatesEveryDocumentsFindings pins that a run over several files
// yields all their findings, each naming the file it came from.
func TestReportAggregatesEveryDocumentsFindings(t *testing.T) {
	t.Parallel()

	read := reader(map[string]string{
		"a.md":         "## Changelog\n",
		"b.md":         "# Title\n",
		"CHANGELOG.md": "",
	})

	report, err := docfiles.Report(read, []string{"a.md", "b.md", "CHANGELOG.md"})

	require.NoError(t, err)
	require.Len(t, report.Diagnostics, 2)
	assert.Equal(t, "a.md", report.Diagnostics[0].Path)
	assert.Equal(t, "CHANGELOG.md", report.Diagnostics[1].Path)
}

// TestReportSurfacesAReadFailure pins that an unreadable file aborts the run
// with its own sentinel rather than being skipped into a clean result — and
// that the report it returns is EMPTY, so a caller that ignores the error
// cannot mistake a half-finished run for a finished one.
func TestReportSurfacesAReadFailure(t *testing.T) {
	t.Parallel()

	read := reader(map[string]string{"a.md": "## Changelog\n"})

	report, err := docfiles.Report(read, []string{"a.md", "missing.md"})

	require.Error(t, err)
	assert.ErrorIs(t, err, docfiles.ErrReadFile)
	assert.ErrorIs(t, err, errUnreadable, "the cause survives so the reason is visible")
	assert.Contains(t, err.Error(), "missing.md", "and the failure names WHICH file")
	assert.Empty(t, report.Diagnostics, "a failed run returns nothing, not a partial answer")
}

// TestReportContainsAnUnreadableDocumentToItsOwnFile pins the containment: a
// binary blob mis-claimed by discovery becomes one finding against that file
// and the run continues, so it cannot silence its neighbours.
func TestReportContainsAnUnreadableDocumentToItsOwnFile(t *testing.T) {
	t.Parallel()

	read := reader(map[string]string{
		"blob.md":  string([]byte{0xff, 0xfe, 0x00}),
		"notes.md": "## Changelog\n",
	})

	report, err := docfiles.Report(read, []string{"blob.md", "notes.md"})

	require.NoError(t, err, "one unreadable document is not the whole run's failure")
	require.Len(t, report.Diagnostics, 2)
	assert.Contains(t, report.Diagnostics[0].Message, "cannot be analyzed as a document")
	assert.Equal(t, "notes.md", report.Diagnostics[1].Path, "its neighbour keeps its finding")
}

// TestReportOfNoFilesIsAnEmptyReport pins the trivial case explicitly.
func TestReportOfNoFilesIsAnEmptyReport(t *testing.T) {
	t.Parallel()

	report, err := docfiles.Report(reader(nil), nil)

	require.NoError(t, err)
	assert.Empty(t, report.Diagnostics)
}
