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

// TestTheRunTruncationNamesTheFirstFileItDropped pins the attribution, which
// nothing verified: removing the guard that stops collecting moved the notice
// from the FIRST dropped document to the last, and the sentence it carries —
// "findings from X onward are omitted" — became false for every document
// between them.
func TestTheRunTruncationNamesTheFirstFileItDropped(t *testing.T) {
	t.Parallel()
	crowded := strings.Repeat("## Changelog\n\n", 1200)
	files := make([]string, 0, 14)
	contents := map[string]string{}
	for i := range 14 {
		name := fmt.Sprintf("d%02d.md", i)
		files = append(files, name)
		contents[name] = crowded
	}

	report := docfiles.Report(reader(contents), files)

	last := report.Diagnostics[len(report.Diagnostics)-1]
	assert.Contains(t, last.Message, "onward are omitted")
	assert.Equal(t, "d09.md", last.Path, "the FIRST document whose findings were dropped, not the last")
	assert.Contains(t, last.Message, "d09.md")
	assert.Contains(t, last.Message, "16800 changelog findings", "and the true total across the run")
}

// TestADocumentExactlyAtItsCapIsNotTruncated pins the per-document boundary. A
// document with exactly as many findings as the cap allows has lost nothing, so
// appending a notice saying "1000 findings, of which 1000 are reported" tells a
// reader something was withheld when nothing was. The test above it uses five
// thousand and the one below twelve; neither sits at the edge, which is the
// argument the size-limit test already makes for itself.
func TestADocumentExactlyAtItsCapIsNotTruncated(t *testing.T) {
	t.Parallel()
	atCap := strings.Repeat("## Changelog\n", 1000)

	diags, err := docfiles.Diagnostics("doc.md", docfiles.Source(atCap))

	require.NoError(t, err)
	require.Len(t, diags, 1000, "every finding, and no notice")
	for _, diag := range diags {
		assert.NotContains(t, diag.Message, "of which", "nothing was withheld, so nothing says it was")
	}
}

// TestTheRunTruncationNamesTheFileWhoseFindingsWereActuallyDropped pins the run
// boundary at an EXACT fit, which is where the attribution goes wrong. Twenty
// documents of five hundred findings fill the run's allowance precisely; the
// next document is the first to lose anything, and naming the one before it
// reports "findings from d19.md onward are omitted" when every one of d19.md's
// was reported.
func TestTheRunTruncationNamesTheFileWhoseFindingsWereActuallyDropped(t *testing.T) {
	t.Parallel()
	full := strings.Repeat("## Changelog\n", 500)
	files := make([]string, 0, 21)
	contents := map[string]string{}
	for i := range 20 {
		name := fmt.Sprintf("d%02d.md", i)
		files = append(files, name)
		contents[name] = full
	}
	files = append(files, "extra.md")
	contents["extra.md"] = "## Changelog\n"

	report := docfiles.Report(reader(contents), files)

	last := report.Diagnostics[len(report.Diagnostics)-1]
	assert.Contains(t, last.Message, "onward are omitted")
	assert.EqualValues(t, "extra.md", last.Path, "the first file that actually lost a finding")
}

// TestAnUnparseableDocumentCountsAsOneFinding pins the count on the branch
// beside the one that IS pinned. A file that cannot be read counts as one
// finding rather than none — its sibling says so and has a test — and a file
// that cannot be DECODED reached the same conclusion with nothing behind it, so
// the run's reported total could silently lose every such file.
func TestAnUnparseableDocumentCountsAsOneFinding(t *testing.T) {
	t.Parallel()
	contents := map[string]string{"bad.md": "\xff\xfe not utf-8 \xff", "good.md": "## Changelog\n"}

	report := docfiles.Report(reader(contents), []string{"bad.md", "good.md"})

	require.Len(t, report.Diagnostics, 2, "the undecodable file is one finding, not none")
	assert.Contains(t, report.Diagnostics[0].Message, "bad.md")
}

// TestAnUndecodableDocumentIsCountedInTheRunTotal pins the count where it is
// actually observable. The run's reported total is the TRUE number of findings,
// not the number carried, so a file that contributes one finding and no
// diagnostics past the cap has to be counted like any other — the branch beside
// it, an unreadable file, is pinned and this one reached the same conclusion
// with nothing behind it.
func TestAnUndecodableDocumentIsCountedInTheRunTotal(t *testing.T) {
	t.Parallel()
	full := strings.Repeat("## Changelog\n", 500)
	files := make([]string, 0, 24)
	contents := map[string]string{}
	for i := range 21 {
		name := fmt.Sprintf("d%02d.md", i)
		files = append(files, name)
		contents[name] = full
	}
	for _, name := range []string{"bad-a.md", "bad-b.md"} {
		files = append(files, name)
		contents[name] = "\xff\xfe not utf-8 \xff"
	}

	report := docfiles.Report(reader(contents), files)

	last := report.Diagnostics[len(report.Diagnostics)-1]
	assert.Contains(t, last.Message, "onward are omitted")
	assert.Contains(t, last.Message, "10502 ",
		"twenty-one documents of five hundred, plus one for each file nobody could decode")
}
