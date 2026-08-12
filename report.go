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

// Names is every spelling the walk reached, one entry per NAME, symlinks left
// unresolved. Files is what the walk found to read, one entry per FILE.
//
// They are two types rather than two `[]string` parameters because they are two
// different questions with two different right answers, and swapping them at a
// call site would compile: the name half would judge one spelling per inode and
// stop seeing the aliases, and the content half would read one document once per
// name it has.
type (
	Names []string
	Files []string
)

// Report runs the changelog check over a walk's names and its files, and
// aggregates the diagnostics into the lean stickler-json report.
//
// THE TWO HALVES READ TWO DIFFERENT LISTS, and that is the whole reason both are
// parameters. The name half judges NAMES, because there the name IS the finding:
// a symlink named `CHANGELOG.md` pointing at an innocent document is a banned
// name in the repository, it survives a clone as mode 120000, and resolving it to
// its target deletes the evidence. The section half reads FILES, one spelling per
// inode, because analyzing one document twice under two names reports one defect
// as two. Driving both halves off one list had to be wrong for one of them, and
// it was wrong for the one that cannot be worked around.
//
// Every diagnostic a run produces passes through ONE counter, ONE limit and ONE
// truncation notice. There were three, and they disagreed. Read failures were
// appended with no limit at all; the limit was tested against the length of a
// slice those failures had already filled; and the notice fired on a total they
// never incremented. A tree holding ten thousand unreadable files and one real
// changelog reported the ten thousand, dropped the changelog, announced
// nothing, and exited zero — a finding lost in silence, which is the one
// outcome this file exists to prevent.
func Report(read FileReader, names Names, files Files) goyze.Report {
	found := run{}
	for _, name := range names {
		banned := nameFindings(Path(name))
		found = found.add(Path(name), banned, findingCount(len(banned)))
	}
	for _, file := range files {
		sections, held := sectionFindings(read, Path(file))
		found = found.add(Path(file), sections, held)
	}
	return found.report()
}

// run is one report being built: what it has collected, the TRUE number of
// findings it has seen, and the first path whose findings it had to drop.
type run struct {
	truncatedAt Path
	diagnostics []goyze.Diagnostic
	total       findingCount
}

// add takes one path's findings into the run — counting all of them, and
// collecting as many as the run's limit still has room for.
//
// The count is the TRUE one, not the collected one: a document past its own
// limit hands back a truncated slice, and summing slices counted this analyzer's
// truncation instead of the documents' findings.
func (r run) add(at Path, found []goyze.Diagnostic, held findingCount) run {
	r.total += held
	if r.truncatedAt != "" {
		// Past the limit the run keeps COUNTING but stops collecting, so the
		// total it reports is the true one.
		return r
	}
	room := reportLimit - findingCount(len(r.diagnostics))
	if findingCount(len(found)) > room {
		r.diagnostics = append(r.diagnostics, found[:room]...)
		r.truncatedAt = at
		return r
	}
	r.diagnostics = append(r.diagnostics, found...)
	return r
}

// report is the run as the stickler runner consumes it, with the notice that
// accounts for anything the limit dropped.
func (r run) report() goyze.Report {
	if r.truncatedAt != "" {
		r.diagnostics = append(r.diagnostics, runTruncation(r.truncatedAt, r.total))
	}
	return goyze.Report{Diagnostics: r.diagnostics}
}

// sectionFindings is one FILE's content findings, whether it could be read or
// not. It never repeats what the NAME half already said — the name is judged
// from the walk's list of names, and a file reached by one name would otherwise
// be banned twice.
//
// A file the gate cannot open becomes ONE finding against that file and the run
// continues, exactly as an unparseable one does — a single blob mis-claimed by
// discovery can never take every other file's findings down with it.
func sectionFindings(read FileReader, file Path) ([]goyze.Diagnostic, findingCount) {
	data, err := read(string(file))
	if err != nil {
		return []goyze.Diagnostic{unreadable(file, err)}, 1
	}
	diags, held, err := sectionDiagnostics(file, Source(data))
	if err != nil {
		return append(diags, unreadable(file, err)), held + 1
	}
	return diags, held
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
