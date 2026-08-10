package docfiles_test

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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

	report := docfiles.Report(read, []string{"a.md", "b.md", "CHANGELOG.md"})

	require.Len(t, report.Diagnostics, 2)
	assert.Equal(t, "a.md", report.Diagnostics[0].Path)
	assert.Equal(t, "CHANGELOG.md", report.Diagnostics[1].Path)
}

// TestReportContainsAReadFailureToItsOwnFile pins that a document the gate
// cannot open is reported rather than fatal. One unreadable file destroying
// every other file's findings is the same outage the not-text repair fixed,
// reached by another door — and the sibling analyzers already answer this way.
func TestReportContainsAReadFailureToItsOwnFile(t *testing.T) {
	t.Parallel()

	read := func(path string) ([]byte, error) {
		if path == "locked.md" {
			return nil, errUnreadable
		}
		return []byte("## Changelog\n"), nil
	}

	report := docfiles.Report(read, []string{"locked.md", "notes.md"})

	paths := map[string]string{}
	for _, d := range report.Diagnostics {
		paths[d.Path] += d.Message
	}
	assert.Contains(t, paths["locked.md"], "cannot be analyzed as a document")
	assert.Contains(t, paths["locked.md"], docfiles.ErrReadFile.Error())
	assert.ErrorIs(t, docfiles.ErrReadFile.With(errUnreadable, "path", "locked.md"), docfiles.ErrReadFile)
	assert.Contains(t, paths["notes.md"], "section is a changelog", "its neighbour keeps its finding")
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

	report := docfiles.Report(read, []string{"blob.md", "notes.md"})

	require.Len(t, report.Diagnostics, 2)
	assert.Contains(t, report.Diagnostics[0].Message, "cannot be analyzed as a document")
	assert.Equal(t, "notes.md", report.Diagnostics[1].Path, "its neighbour keeps its finding")
}

// TestReportOfNoFilesIsAnEmptyReport pins the trivial case explicitly.
func TestReportOfNoFilesIsAnEmptyReport(t *testing.T) {
	t.Parallel()

	report := docfiles.Report(reader(nil), nil)

	assert.Empty(t, report.Diagnostics)
}

// TestReportLimitBoundsTheRunAndNeverLosesTheCount pins the bound the per-document limit does not
// provide. A document limit bounds a document; it does not bound a tree — two
// thousand documents each at their own limit produced 490 MB of report and
// 2.3 GB resident from 31 MB of input, which is the failure the per-document
// limit exists to prevent, reached by a route it does not cover.
func TestReportLimitBoundsTheRunAndNeverLosesTheCount(t *testing.T) {
	t.Parallel()

	files := make([]string, 0, 30)
	contents := map[string]string{}
	for i := range 30 {
		name := "d" + strconv.Itoa(i) + ".md"
		files = append(files, name)
		contents[name] = strings.Repeat("## Changelog\n", 500)
	}

	report := docfiles.Report(reader(contents), files)

	assert.LessOrEqual(t, len(report.Diagnostics), 10_001, "the run is bounded, not just each document")
	last := report.Diagnostics[len(report.Diagnostics)-1]
	assert.Contains(t, last.Message, "15000 changelog findings across this run")
	assert.Contains(t, last.Message, "10000 are reported")
}

// TestARunUnderTheLimitIsReportedInFull pins the other side, so the bound never
// quietly truncates an ordinary tree.
func TestARunUnderTheLimitIsReportedInFull(t *testing.T) {
	t.Parallel()

	report := docfiles.Report(
		reader(map[string]string{"a.md": "## Changelog\n", "b.md": "## Changelog\n"}),
		[]string{"a.md", "b.md"},
	)

	assert.Len(t, report.Diagnostics, 2)
}

// TestAnUnreadableDocumentIsOneFindingAndTheRunContinues pins the containment
// AND its arithmetic. One file the gate cannot open must not empty the report,
// must not produce two findings for one file, and must count toward the same
// total every other finding does — three properties that were each true of a
// different counter, and so of none of them together.
func TestAnUnreadableDocumentIsOneFindingAndTheRunContinues(t *testing.T) {
	t.Parallel()

	report := docfiles.Report(func(path string) ([]byte, error) {
		if path == "locked.md" {
			return nil, os.ErrPermission
		}
		return []byte("## Changelog\n"), nil
	}, []string{"locked.md", "good.md"})

	require.Len(t, report.Diagnostics, 2, "one finding for the unreadable file, one for the readable one")
	assert.Equal(t, "locked.md", report.Diagnostics[0].Path)
	assert.Contains(t, report.Diagnostics[0].Message, "cannot be analyzed as a document")
	assert.Equal(t, "good.md", report.Diagnostics[1].Path)
}

// TestAReadFailureCannotCrowdOutARealFinding pins the hole between the three
// counters. Read failures were appended past the limit, the limit was measured
// against a slice they had filled, and the truncation notice fired on a total
// they never incremented — so a tree of unreadable files silently discarded a
// real changelog and exited zero, which is the one outcome this file exists to
// prevent.
func TestAReadFailureCannotCrowdOutARealFinding(t *testing.T) {
	t.Parallel()

	files := make([]string, 0, 12_000)
	for i := 0; i < 11_000; i++ {
		files = append(files, fmt.Sprintf("locked-%d.md", i))
	}
	files = append(files, "CHANGELOG.md")

	report := docfiles.Report(func(path string) ([]byte, error) {
		if strings.HasPrefix(path, "locked-") {
			return nil, os.ErrPermission
		}
		return []byte("## Changelog\n"), nil
	}, files)

	last := report.Diagnostics[len(report.Diagnostics)-1]
	assert.Contains(t, last.Message, "onward are omitted", "the run says so rather than passing over it")
	assert.NotEmpty(t, last.Path, "a diagnostic the runner cannot attribute is one it can only ignore")
	assert.Contains(t, last.Message, "11002 changelog findings", "the true total, not the reported one")
}
