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

// Report runs the changelog check over each document and aggregates the
// diagnostics into the lean stickler-json report.
func Report(read FileReader, files []string) goyze.Report {
	report := goyze.Report{}
	total := findingCount(0)
	for _, file := range files {
		data, err := read(file)
		if err != nil {
			// Contained, exactly as an unreadable document is. A file the gate
			// cannot open destroying every other file's findings is the same
			// outage the not-text repair fixed, reached by another door — and
			// the sibling analyzers already answer this way.
			report.Diagnostics = append(report.Diagnostics, unreadable(Path(file), err))
			continue
		}
		found := documentDiagnostics(Path(file), Source(data))
		total += findingCount(len(found))
		if findingCount(len(report.Diagnostics)) < reportLimit {
			report.Diagnostics = append(report.Diagnostics, found...)
		}
	}
	if total > reportLimit {
		report.Diagnostics = append(report.Diagnostics, runTruncation(total))
	}
	return report
}

// runTruncation is the finding that stands for everything past the run's limit,
// carrying the true total so nothing is silently lost.
func runTruncation(found findingCount) goyze.Diagnostic {
	return diagnostic("", 1, finding(fmt.Sprintf(runTruncationMessage, found, reportLimit)))
}

// runTruncationMessage formats the finding that stands for the rest of a run.
const runTruncationMessage = "%d changelog findings across this run, of which %d are reported; a tree with this " +
	"many is one problem rather than that many, and carrying them all costs more memory than reading it did"

// unreadable is the finding for a document the analyzer could not open at all.
func unreadable(file Path, cause error) goyze.Diagnostic {
	return diagnostic(file, 1, finding(fmt.Sprintf(unreadableMessage, ErrReadFile.With(cause, "path", string(file)))))
}

// documentDiagnostics is one document's findings, with an unreadable document
// reported as a finding of its own rather than raised as the whole run's error.
func documentDiagnostics(file Path, source Source) []goyze.Diagnostic {
	diags, err := Diagnostics(file, source)
	if err != nil {
		return []goyze.Diagnostic{diagnostic(file, 1, finding(fmt.Sprintf(unreadableMessage, err)))}
	}
	return diags
}
