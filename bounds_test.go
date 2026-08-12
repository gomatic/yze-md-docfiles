package docfiles_test

// What a report carries when there is too much of it or when it could not be
// read at all, and where a finding about a whole file points. Every boundary
// here was reachable by an off-by-one, and every whole-file line number was
// unasserted.

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	docfiles "github.com/gomatic/yze-md-docfiles"
)

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

	report := reported(reader(contents), files)

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

	report := reported(reader(contents), files)

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

	report := reported(reader(contents), []string{"bad.md", "good.md"})

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

	report := reported(reader(contents), files)

	last := report.Diagnostics[len(report.Diagnostics)-1]
	assert.Contains(t, last.Message, "onward are omitted")
	assert.Contains(t, last.Message, "10502 ",
		"twenty-one documents of five hundred, plus one for each file nobody could decode")
}

// TestEveryWholeFileFindingNamesTheFirstLine pins the position of the findings
// that address a file rather than a heading. The column was pinned and the LINE
// was not, for any of them: five separate mutations moved a file-level finding
// to line 2 with the suite green, and one of them put a finding at line 2 of a
// document that has no line 2 at all.
func TestEveryWholeFileFindingNamesTheFirstLine(t *testing.T) {
	t.Parallel()
	crowded := strings.Repeat("## Changelog\n", 1200)

	named, err := docfiles.Diagnostics("docs/CHANGELOG.md", "")
	require.NoError(t, err)
	require.Len(t, named, 1, "an empty changelog file is still a changelog file")
	assert.Equal(t, 1, named[0].Line, "a document with no lines has no line 2 to point at")

	truncated, err := docfiles.Diagnostics("doc.md", docfiles.Source(crowded))
	require.NoError(t, err)
	assert.Equal(t, 1, truncated[len(truncated)-1].Line, "the per-document truncation notice")

	report := reported(reader(map[string]string{"bad.md": "\xff\xfe"}), []string{"bad.md"})
	require.Len(t, report.Diagnostics, 1)
	assert.Equal(t, 1, report.Diagnostics[0].Line, "a document nobody could decode")

	unreadable := docfiles.Unreadable([]string{"locked"})
	require.Len(t, unreadable, 1, "one tree, one finding — not two")
	assert.Equal(t, 1, unreadable[0].Line)
	assert.EqualValues(t, "locked", unreadable[0].Path)
}

// TestTheRunTruncationNoticeNamesTheFirstLineToo pins the last of the five, which
// needs a run large enough to overflow.
func TestTheRunTruncationNoticeNamesTheFirstLineToo(t *testing.T) {
	t.Parallel()
	full := strings.Repeat("## Changelog\n", 600)
	files := make([]string, 0, 20)
	contents := map[string]string{}
	for i := range 20 {
		name := fmt.Sprintf("d%02d.md", i)
		files = append(files, name)
		contents[name] = full
	}

	report := reported(reader(contents), files)

	last := report.Diagnostics[len(report.Diagnostics)-1]
	require.Contains(t, last.Message, "onward are omitted")
	assert.Equal(t, 1, last.Line)
}

// TestAnUnreadableChangelogStillSaysItMustNotExist pins the guard ordering,
// which is a contract rather than a reorder. The size and text guards describe a
// file's CONTENTS and were deciding a rule that depends only on its NAME, so an
// oversized or undecodable `CHANGELOG.md` was told the analyzer could not read
// it — true, unhelpful, and about the wrong problem. Both are said now: the ban
// AND the tool failure, because a file the gate could not read is still a fact
// worth reporting.
func TestAnUnreadableChangelogStillSaysItMustNotExist(t *testing.T) {
	t.Parallel()

	for name, unreadable := range map[string]struct {
		wantErr error
		source  string
	}{
		"too large":       {docfiles.ErrTooLarge, strings.Repeat("x", int(docfiles.SizeLimit)+1)},
		"not utf-8":       {docfiles.ErrNotText, "\xff\xfe not text \xff"},
		"not utf-8 alone": {docfiles.ErrNotText, "\xff"},
	} {
		diags, err := docfiles.Diagnostics("docs/CHANGELOG.md", docfiles.Source(unreadable.source))

		require.ErrorIs(t, err, unreadable.wantErr, "%s: the failure is still raised", name)
		require.Len(t, diags, 1, "%s: and the ban is still stated", name)
		assert.Equal(t, 1, diags[0].Line, "%s: at the document's first line", name)
		assert.Contains(t, diags[0].Message, "CHANGELOG.md must not be committed", name)
	}
}

// TestAnInnocentUnreadableDocumentInventsNothing pins the other half of that
// contract, and it is the half a careless reorder breaks: nothing was read, so
// nothing about the document's CONTENTS can have been determined. A file whose
// name is nobody's changelog yields the tool failure and not one finding more.
func TestAnInnocentUnreadableDocumentInventsNothing(t *testing.T) {
	t.Parallel()

	for name, unreadable := range map[string]struct {
		wantErr error
		source  string
	}{
		"too large": {docfiles.ErrTooLarge, strings.Repeat("## Changelog\n", int(docfiles.SizeLimit)/13+1)},
		"not utf-8": {docfiles.ErrNotText, "## Changelog\n\xff\xfe"},
	} {
		diags, err := docfiles.Diagnostics("docs/notes.md", docfiles.Source(unreadable.source))

		require.ErrorIs(t, err, unreadable.wantErr, name)
		assert.Empty(t, diags, "%s: a document nobody read has no sections anybody can name", name)
	}
}

// TestAChangelogNameOutlivesEveryReaderThatCouldNotOpenIt pins the same contract
// at the two entry points that never see a byte at all: the reader refusing a
// file, and the walk handing back a path it could not enter. Each of those is
// still the gate saying that path EXISTS, and a `CHANGELOG.md` behind a broken
// symlink or a permission error is a changelog in the repository.
func TestAChangelogNameOutlivesEveryReaderThatCouldNotOpenIt(t *testing.T) {
	t.Parallel()

	report := reported(func(string) ([]byte, error) { return nil, os.ErrPermission },
		[]string{"docs/CHANGELOG.md", "docs/notes.md"})

	require.Len(t, report.Diagnostics, 3, "the ban, its unreadable finding, and the innocent file's")
	assert.Contains(t, report.Diagnostics[0].Message, "must not be committed", "the ban comes first")
	assert.Contains(t, report.Diagnostics[1].Message, "cannot be analyzed as a document")
	assert.Equal(t, "docs/notes.md", report.Diagnostics[2].Path)
	assert.NotContains(t, report.Diagnostics[2].Message, "must not be committed",
		"and an innocent file nobody could open earns no ban")

	walked := docfiles.Unreadable([]string{"docs/CHANGELOG.md", "docs/locked"})

	require.Len(t, walked, 3)
	assert.Contains(t, walked[0].Message, "must not be committed")
	assert.Contains(t, walked[1].Message, "cannot be analyzed as a document")
	assert.Equal(t, "docs/locked", walked[2].Path)
}
