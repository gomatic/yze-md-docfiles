package docfiles

import (
	"fmt"

	errs "github.com/gomatic/go-error"
	goyze "github.com/gomatic/go-yze"
)

// errReadFile names the read failure carried inside the finding an unreadable
// document produces.
//
// It is UNEXPORTED because it is not an error any caller can receive. It is
// interpolated into [unreadableMessage] by [unreadable], which returns a
// diagnostic; [Report] returns no error at all. Exported, it advertised a
// sentinel that `errors.Is` could never match, and the only test that could
// name it had to build the expected value out of the constant it compared
// against — a claim about the error helper, not about this package, and one
// deleted for measuring nothing. A sentinel with no consumer is a contract the
// package never fulfills; the read failure's contract is the MESSAGE, which is
// what the report actually carries and what the test now pins.
const errReadFile errs.Const = "cannot read documentation file"

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
const unreadableMessage = "cannot be analyzed as a document: %v; the gate cannot vouch for a file it could not " +
	"read, so this is reported rather than passed over"

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
func Unreadable(paths []string) []goyze.Diagnostic {
	diags := make([]goyze.Diagnostic, 0, len(paths))
	for _, path := range paths {
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
		return []goyze.Diagnostic{unreadable(file, err)}, 1
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

// unreadable is the finding for a document the analyzer could not open at all.
func unreadable(file Path, cause error) goyze.Diagnostic {
	return diagnostic(file, 1, finding(fmt.Sprintf(unreadableMessage, errReadFile.With(cause, "path", string(file)))))
}

// documentDiagnostics is one document's findings, with an unreadable document
// reported as a finding of its own rather than raised as the whole run's error.
func documentDiagnostics(file Path, source Source) ([]goyze.Diagnostic, findingCount) {
	diags, held, err := countedDiagnostics(file, source)
	if err != nil {
		return []goyze.Diagnostic{diagnostic(file, 1, finding(fmt.Sprintf(unreadableMessage, err)))}, 1
	}
	return diags, held
}
