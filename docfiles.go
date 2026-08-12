package docfiles

import (
	"strings"
	"unicode/utf8"

	goyze "github.com/gomatic/go-yze"
)

// Name is the analyzer's stable identifier — the suffix of its flat rule id and
// the key the yze suite catalogs it under.
const Name = "docfiles"

// Tool is the suite name stamped on every diagnostic.
const Tool = "yze"

// Rule is the stable, flat rule id every diagnostic carries: "yze/" + [Name].
const Rule = Tool + "/" + Name

// Category is the language group this analyzer belongs to, used by the yze
// suite to run it only when processing documentation.
const Category = "docs"

// Path is the file path stamped on each diagnostic's location.
type Path string

// Source is the text of one document.
type Source string

// finding is one diagnostic's rendered message.
type finding string

// fileMessage formats a banned-file finding.
//
// It addresses BOTH authors, because both write this file. It once read "is a
// hand-maintained changelog" and recommended generated release notes — a
// sentence that says nothing to a generator and reads as an instruction to
// switch tools rather than to delete a file. That wording was then mistaken for
// a REASON, and the file finding was exempted for anything carrying a generated
// claim. The message names the fault instead: the file is in the repository.
const fileMessage = "%s must not be committed to a repository, whoever or whatever wrote it; " +
	"git history is the changelog, and release notes belong to the tag and the release"

// headingMessage formats a banned-section finding.
const headingMessage = "the %q section is a changelog inside another document; " +
	"git history already records this, and the copy is the one that goes stale"

// Diagnostics reports the changelog findings for one document: the file itself
// when its name is a changelog, and every heading that opens one.
//
// A document too large to read yields [ErrTooLarge] and one whose bytes are not
// text yields [ErrNotText], so the caller surfaces a tool failure rather than a
// clean pass over a file nobody read.
//
// AN ERROR AND FINDINGS ARRIVE TOGETHER, and that is the contract rather than an
// oversight. What a document's NAME says is knowable from a directory entry, so
// it survives every reason the bytes could not be read — an oversized
// `CHANGELOG.md` that says only "the analyzer could not read this" tells its
// author to go looking for a problem in a file whose problem is that it exists.
// The error is not suppressed either: a file the gate could not read is still a
// fact worth reporting, and both are said.
func Diagnostics(at Path, source Source) ([]goyze.Diagnostic, error) {
	diags, _, err := countedDiagnostics(at, source)
	return diags, err
}

// countedDiagnostics is [Diagnostics] with the TRUE number of findings the
// document holds, which is not the number reported: the per-document limit
// truncates the slice, so a run summing the slices counted its own truncation
// rather than the documents. It reported "11011 findings" over a tree holding
// 16500, in the very sentence that says the true count is named.
func countedDiagnostics(at Path, source Source) ([]goyze.Diagnostic, findingCount, error) {
	// The NAME is decided FIRST, in a function that takes no bytes at all, and
	// nothing below can take it back. It is decided before the generated claim,
	// because a changelog is banned for EXISTING rather than for who typed it —
	// and reading the claim first made the rule silent for exactly the files it
	// most needs to catch, since release-please, git-cliff and goreleaser all
	// open their CHANGELOG.md with one. It is decided before the size and text
	// guards for the same reason one step further out: those describe the
	// CONTENTS, and a name is knowable with zero bytes read, so letting them
	// answer first told an author the analyzer had trouble reading a file whose
	// whole problem is that it is there.
	named := nameFindings(at)
	sections, held, err := sectionDiagnostics(at, source)
	return append(named, sections...), findingCount(len(named)) + held, err
}

// sectionDiagnostics is the half of the rule that reads BYTES: every heading in
// one document that opens a changelog section.
//
// It is a separate function because it is driven by a separate list. A walk
// hands back one entry per NAME and one entry per FILE, and this half asks for
// files — analyzing one document twice under two names reports one defect as two
// — while the name half asks for names, where two names really are two findings.
func sectionDiagnostics(at Path, source Source) ([]goyze.Diagnostic, findingCount, error) {
	if goyze.ByteCount(len(source)) > SizeLimit {
		return nil, 0, ErrTooLarge.With(nil, "path", string(at), "bytes", len(source))
	}
	if !utf8.ValidString(string(source)) {
		return nil, 0, ErrNotText.With(nil, "path", string(at))
	}
	_, ext := nameAndExtension(at)
	if !sectionExtensions[ext] {
		return nil, 0, nil
	}
	// Stripped ONCE, here, before the parse, so every reader below sees the same
	// text. A byte order mark ahead of `<!--` makes the line a paragraph rather
	// than an HTML comment — which is how a Windows editor silently exempts a
	// document from the claim its generator really did write.
	doc := parse(Source(strings.TrimPrefix(string(source), byteOrderMark)))
	if isGenerated(doc, ext) {
		// Only the SECTIONS are out of scope. A generated document cannot be
		// fixed by editing it, so a machine-written docs page carrying four
		// hundred `## Unreleased` headings is one problem in its generator, and
		// reporting four hundred findings buries it.
		return nil, 0, nil
	}
	headings, total := headingDiagnostics(at, doc)
	// Counted BEFORE the notice is appended. The truncation notice is this
	// analyzer's own bookkeeping, not something the document contains, and
	// counting it inflated the run total by one per truncated document.
	if total > findingLimit {
		headings = append(headings, truncation(at, total))
	}
	return headings, total, nil
}

// diagnostic builds one finding at a line. Every finding of this rule addresses
// a whole line — a file by its name, a section by its heading — so the column
// is always the first, which is where an editor should land.
func diagnostic(at Path, line lineNumber, message finding) goyze.Diagnostic {
	return goyze.Diagnostic{
		Tool:     Tool,
		Rule:     Rule,
		Path:     string(at),
		Line:     int(line),
		Col:      1,
		Severity: goyze.SeverityError,
		Message:  string(message),
	}
}
