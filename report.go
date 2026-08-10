package docfiles

import (
	"fmt"

	errs "github.com/gomatic/go-error"
	goyze "github.com/gomatic/go-yze"
)

// ErrReadFile reports that a document could not be read.
const ErrReadFile errs.Const = "cannot read documentation file"

// ErrNotRegularFile reports a named path whose contents cannot be read as
// prose. Reading one is not merely useless: a FIFO blocks forever on open and a
// character device never ends, so a single such argument hangs the gate instead
// of failing it.
const ErrNotRegularFile errs.Const = "not a regular file"

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
// reportLimit bounds how many findings ONE RUN carries.
//
// The per-document limit bounds a document; it does not bound a tree. Two
// thousand documents each at their own limit produced 490 MB of report and
// 2.3 GB resident from 31 MB of input — the same failure the per-document
// limit exists to prevent, reached by a route it does not cover. A run with
// this many findings is one problem too, and the true count is still named.
const reportLimit = 10_000

// ErrNoPaths reports a run given nothing to analyze. A runner whose root
// placeholder expands to nothing would otherwise green the gate over a
// repository no analyzer ever looked at.
const ErrNoPaths errs.Const = "no paths to analyze"

// Unreadable is the finding for each tree the walk could not enter. A directory
// the gate cannot descend into is reported rather than passed over, so nothing
// is lost in silence and the run still yields every other file's findings.
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

// runTruncationMessage formats the finding that stands for the rest of a run.
const runTruncationMessage = "%d changelog findings across this run, of which %d are reported; findings from %s " +
	"onward are omitted, because a tree with this many is one problem rather than that many"

// unreadable is the finding for a document the analyzer could not open at all.
func unreadable(file Path, cause error) goyze.Diagnostic {
	return diagnostic(file, 1, finding(fmt.Sprintf(unreadableMessage, ErrReadFile.With(cause, "path", string(file)))))
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
