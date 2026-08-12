package docfiles

import (
	"fmt"

	errs "github.com/gomatic/go-error"
	goyze "github.com/gomatic/go-yze"
)

// ErrNotRegularFile reports a named path whose contents cannot be read as a
// document. It IS the shared sentinel, not a second one beside it: the refusal
// is raised by the discovery this command walks with, and nothing in this module
// ever returned the local copy — a caller holding it matched only because the two
// happened to carry the same text, so rewording either would have broken
// `errors.Is` for everyone. That is the defect [ErrTooLarge] was already repaired
// for, one declaration away.
const ErrNotRegularFile = goyze.ErrNotRegularFile

// FileReader reads a file's bytes; injected so aggregation is testable without
// the filesystem.
type FileReader func(path string) ([]byte, error)

// unreadableMessage formats the finding for a document that could not be read
// as prose.
//
// It does NOT name the path. [goyze.Diagnostic.Path] already carries it in a
// field a runner can read, and interpolating it here said the same thing twice
// — once structured, once buried in a sentence — which is the duplication the
// file finding's own message was corrected for.
//
// The reason it interpolates is a STRING, not an error. No caller receives an
// error for this condition: [unreadable] returns a diagnostic and [Report]
// returns a report. It used to be assembled from an `errs.Const` that no
// `errors.Is` could reach, which is message text wearing an error type — and
// yze/errtested was right to say so.
const unreadableMessage = "cannot be analyzed as a document: %s; the gate cannot vouch for a file it could not " +
	"read, so this is reported rather than passed over"

// unopenable is the reason given for a path the walk never attempted to open —
// a directory it could not enter, a FIFO, a device, a link resolving to
// nothing. There is genuinely no error to quote for those, and formatting a nil
// one would show a reader the verb `%!s(<nil>)` where the reason belongs.
const unopenable = "the walk could not read this path"

// A document that cannot be read, or whose bytes are not text, becomes ONE
// finding against that file and the run continues — so a single blob
// mis-claimed by discovery, or one file the gate cannot open, can never take
// every other file's findings down with it. Nothing is passed over in silence,
// which is the one outcome a gate must never produce.

// ErrNoPaths reports a run given nothing to analyze. A runner whose root
// placeholder expands to nothing would otherwise green the gate over a
// repository no analyzer ever looked at.
const ErrNoPaths errs.Const = "no paths to analyze"

// Unreadable is the finding for each path the walk could not read: a directory
// it could not enter, and a file it could not have read had it tried — a FIFO,
// a device, a link resolving to nothing. Both are REPORTED rather than skipped,
// because a path the gate cannot open is where an unchecked one would hide, and
// the run still yields every other file's findings.
//
// What the path's NAME says is reported alongside, and first. The walk handing
// back a path it could not open is still the walk saying that path EXISTS, so a
// `CHANGELOG.md` behind a broken symlink is a changelog in the repository — and
// saying only that it could not be read would answer a question nobody asked.
func Unreadable(paths []string) []goyze.Diagnostic {
	diags := make([]goyze.Diagnostic, 0, len(paths))
	for _, path := range paths {
		diags = append(diags, nameFindings(Path(path))...)
		diags = append(diags, unreadable(Path(path), nil))
	}
	return diags
}

// Report runs the changelog check over each document and aggregates the
// diagnostics into the lean stickler-json report.
// Every diagnostic a run produces passes through ONE counter, ONE limit and ONE
// truncation notice. There were three, and they disagreed. Read failures were
// appended with no limit at all; the limit was tested against the length of a
// slice those failures had already filled; and the notice fired on a total they
// never incremented. A tree holding ten thousand unreadable files and one real
// changelog reported the ten thousand, dropped the changelog, announced
// nothing, and exited zero — a finding lost in silence, which is the one
// outcome this file exists to prevent.
func Report(read FileReader, files []string) goyze.Report {
	report := goyze.Report{}
	total := findingCount(0)
	truncatedAt := Path("")
	for _, file := range files {
		found, held := fileFindings(read, Path(file))
		// The TRUE count, not the reported one: a document past its own limit
		// hands back a truncated slice, and summing slices counted this run's
		// truncation instead of the documents' findings.
		total += held
		if truncatedAt != "" {
			// Past the limit the run keeps COUNTING but stops collecting, so
			// the total it reports is the true one.
			continue
		}
		room := reportLimit - findingCount(len(report.Diagnostics))
		if findingCount(len(found)) > room {
			report.Diagnostics = append(report.Diagnostics, found[:room]...)
			truncatedAt = Path(file)
			continue
		}
		report.Diagnostics = append(report.Diagnostics, found...)
	}
	if truncatedAt != "" {
		report.Diagnostics = append(report.Diagnostics, runTruncation(truncatedAt, total))
	}
	return report
}

// fileFindings is one file's diagnostics, whether it could be read or not.
//
// A file the gate cannot open becomes ONE finding against that file and the run
// continues, exactly as an unparseable one does — a single blob mis-claimed by
// discovery can never take every other file's findings down with it.
func fileFindings(read FileReader, file Path) ([]goyze.Diagnostic, findingCount) {
	data, err := read(string(file))
	if err != nil {
		// The name is still knowable: the reader refusing a file says nothing
		// about what it is called, and a locked `CHANGELOG.md` must not exist
		// whether or not anybody can open it.
		named := nameFindings(file)
		return append(named, unreadable(file, err)), findingCount(len(named)) + 1
	}
	return documentDiagnostics(file, Source(data))
}

// runTruncation is the finding that stands for everything past the run's limit,
// carrying the true total so nothing is silently lost.
//
// It is attributed to the FILE the run stopped collecting at, not to no file at
// all. A diagnostic with an empty path is one the runner cannot attribute,
// baseline or ratchet — the same objection that made an anonymous rule id
// invalid — and this one has a truthful path available: the document whose
// findings were the first to be dropped.
func runTruncation(at Path, found findingCount) goyze.Diagnostic {
	return diagnostic(at, 1, finding(fmt.Sprintf(runTruncationMessage, found, reportLimit, at)))
}

// unreadable is the finding for a document the analyzer could not read: one the
// walk could not open, one the reader refused, and one whose bytes are not
// prose.
//
// ONE condition reaches a reader ONE way. [documentDiagnostics] used to format
// its own copy of [unreadableMessage], so the same failure was worded by
// whichever layer happened to notice it — and that second path, which never
// touched a sentinel at all, is what showed the first one did not need one.
func unreadable(file Path, cause error) goyze.Diagnostic {
	return diagnostic(file, 1, finding(fmt.Sprintf(unreadableMessage, reason(cause))))
}

// reason is what the finding says went wrong, for a cause that may not exist.
//
// Both arms are reached and both are pinned: [Unreadable] reports paths it
// never attempted to open and takes the first, while every read and decode
// failure carries a real cause and takes the second.
func reason(cause error) string {
	if cause == nil {
		return unopenable
	}
	return cause.Error()
}

// documentDiagnostics is one document's findings, with an unreadable document
// reported as a finding of its own rather than raised as the whole run's error.
//
// The unreadable finding is APPENDED to whatever the rule could already say,
// rather than substituted for it. [countedDiagnostics] decides the file's name
// before it looks at a byte, so a `CHANGELOG.md` too large or too malformed to
// parse arrives here carrying its ban, and dropping that on the floor would
// leave an author told to investigate a reading problem instead of to delete a
// file.
func documentDiagnostics(file Path, source Source) ([]goyze.Diagnostic, findingCount) {
	diags, held, err := countedDiagnostics(file, source)
	if err != nil {
		return append(diags, unreadable(file, err)), held + 1
	}
	return diags, held
}
