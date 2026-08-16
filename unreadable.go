package docfiles

import (
	"fmt"

	goyze "github.com/gomatic/go-yze"
)

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

// unreadable is the finding for a document the analyzer could not read: one the
// walk could not open, one the reader refused, and one whose bytes are not
// prose.
//
// ONE condition reaches a reader ONE way. A second layer used to format its own
// copy of [unreadableMessage], so the same failure was worded by whichever layer
// happened to notice it — and that second path, which never touched a sentinel
// at all, is what showed the first one did not need one.
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
