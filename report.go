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

// Report runs the changelog check over each document and aggregates the
// diagnostics into the lean stickler-json report.
//
// A read failure aborts with ErrReadFile, because a filesystem that will not
// answer is a tool failure rather than a property of any one file. A document
// whose bytes are not text is contained instead: it becomes one finding against
// that file and the run continues, so a single binary blob mis-claimed by
// discovery cannot take every other file's findings down with it. Either way
// nothing is passed over in silence, which is the one outcome a gate must never
// produce.
func Report(read FileReader, files []string) (goyze.Report, error) {
	report := goyze.Report{}
	for _, file := range files {
		data, err := read(file)
		if err != nil {
			return goyze.Report{}, ErrReadFile.With(err, "path", file)
		}
		report.Diagnostics = append(report.Diagnostics, documentDiagnostics(Path(file), Source(data))...)
	}
	return report, nil
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
