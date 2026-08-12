// Package docfiles reports changelog FILES — and the section inside another
// document that is one by another name.
//
// A changelog file does not belong in a repository, generated or not. It
// duplicates what git already records, attributed and immutable, and the
// duplicate is the copy that rots: the file says one thing, `git log` says
// another, and the reader has no way to tell which is stale. Release notes are
// cut from that history and belong to the tag and the release, where they are
// written once and never edited again.
//
// GENERATED-NESS IS NOT AN EXEMPTION FROM THE FILE. It is a property of who
// typed the document; the ban is on the document EXISTING. The file finding was
// once suppressed by a generated claim, on the reasoning that telling a machine
// to stop hand-maintaining its output is nonsense — which confused a badly
// worded message with a reason, and made the exemption a door standing open in
// exactly the shape the ban is most likely broken in: release-please, git-cliff
// and goreleaser all write a `CHANGELOG.md` opening with `Code generated … DO
// NOT EDIT.` or `@generated`, and the rule reported nothing at all for every one
// of them.
//
// What the claim still exempts is the HEADING half, inside a document that is
// not itself a changelog. A machine-written docs page carrying four hundred
// `## Unreleased` headings is ONE problem, in its generator, and four hundred
// findings bury it — and no author can act on them, because editing that file
// is overwritten on the next run.
//
// The rule is deliberately NARROW, and every word of its vocabulary was
// measured against the fleet before being admitted. It reports a file whose
// name IS a changelog, in the prose extensions a changelog is written in, and a
// heading that opens a changelog section. It does not report a document merely
// called `history.md` — the fleet has two, both genuine Hugo content pages
// about a project's history — nor a source file named changelog.go, which is
// code that manages the concept rather than an instance of it.
//
// Three names were considered and REFUSED on the measurement. A bare `Changes`
// or `History` heading is ordinary prose in a design document. `Release Notes`
// as a heading has three fleet instances and all three are legitimate — Go's
// own doc ABOUT writing release notes, and two docs-site index sections — so it
// is banned as a file name, where it is unambiguous, and left alone as a
// heading. A rule that fires on any of those is a rule its own repository
// cannot document itself under.
//
// # THE TWO HALVES READ DIFFERENT SETS OF FILES
//
// The name half judges a path and reads nothing, so it applies to every
// extension a changelog is spelled with — [nameExtensions] — including `.rst`
// and `.adoc`, which the `yze/markup` rule bans outright. Dropping them because
// another rule forbids the format would open a hole reachable through a
// SANCTIONED exemption: exemptions in this fleet are per-rule, so a repository
// that earns a legitimate `yze/markup` exemption would become a place where a
// `CHANGELOG.adoc` is invisible to this one.
//
// The section half needs a parse, so it applies only to what this rule parses —
// [sectionExtensions], which is markdown and the two spellings read as markdown.
// The capability given up is a changelog SECTION inside a reStructuredText or
// AsciiDoc document; measured over the fleet, that protects no file, in formats
// that must not exist here at all.
package docfiles

import (
	"fmt"
	"path"
	"strings"
	"unicode/utf8"

	errs "github.com/gomatic/go-error"
	goyze "github.com/gomatic/go-yze"
)

// ErrTooLarge reports a file past the size this rule will read. It IS the shared
// sentinel, not a second one beside it: the bound is enforced in two places —
// here, and at the one place the command reads a file — and two sentinels for
// one condition meant `errors.Is` answered false for whichever layer the caller
// had not thought of.
const ErrTooLarge = goyze.ErrTooLarge

// SizeLimit is the largest document read, in bytes. It is exported so the
// command can refuse a file from its directory entry, BEFORE opening it —
// asking afterwards cost the file's own size twice over for a rule that then
// declined to apply. It is generous by three
// orders of magnitude over any real document, so it bounds the pathological
// case without ever turning away prose.
const SizeLimit goyze.ByteCount = 8 << 20

// ErrNotText reports a document whose bytes are not text. A binary file cannot
// be read as prose, and guessing at its lines would invent findings from
// whatever byte happened to look like a `#`.
const ErrNotText errs.Const = "document is not valid UTF-8 text"

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

// nameExtensions is the set of extensions in which a file's NAME is judged. A
// changelog is prose; `changelog.go` is code that manages the concept rather
// than an instance of it, and reporting it would be reporting the cure as the
// disease. The empty extension is here for the canonical Unix spelling, a bare
// `CHANGELOG` with no extension at all.
//
// `.rst` and `.adoc` stay here even though `yze/markup` bans both formats
// outright, because that rule can be exempted per repository and this one is a
// different rule: dropping them would make a `CHANGELOG.adoc` invisible in
// just the repositories likely to hold one.
var nameExtensions = map[extension]bool{
	extensionlessExt: true,
	markdownExt:      true,
	markdownLongExt:  true,
	plainTextExt:     true,
	restructuredExt:  true,
	asciidocExt:      true,
}

// sectionExtensions is the set this rule PARSES, and so the set whose headings
// it reads. Every admitted member is read as markdown, which is a decision
// rather than a default: `.txt` and the extensionless spelling are what a
// repository writes a bare `CHANGELOG` or `NOTES` in, and markdown is the only
// prose grammar this fleet has.
//
// The two markup spellings are refused rather than omitted. There is no
// CommonMark reading of a reStructuredText document that means anything, and
// `yze/markup` bans the file itself — so the only capability given up is a
// changelog SECTION inside one, which the fleet measurement says protects no
// file. Writing the refusal down is what lets the next reader tell it from an
// oversight.
var sectionExtensions = map[extension]bool{
	extensionlessExt: true,
	markdownExt:      true,
	markdownLongExt:  true,
	plainTextExt:     true,
	restructuredExt:  false,
	asciidocExt:      false,
}

// extension is a file's suffix, lower-cased.
type extension string

// The prose extensions this rule reads, named once so the three places that
// decide something per-format stay in step.
const (
	markdownExt      extension = ".md"
	markdownLongExt  extension = ".markdown"
	plainTextExt     extension = ".txt"
	restructuredExt  extension = ".rst"
	asciidocExt      extension = ".adoc"
	extensionlessExt extension = ""
)

// baseName is a file's final path element.
type baseName string

// findingCount is how many findings a document produced.
type findingCount int

// findingLimit bounds how many findings ONE document contributes.
//
// Streaming the lines removed the amplification from the line slice and left it
// in the diagnostic slice: eight megabytes of legal prose, every line a banned
// heading, produced 230 MB of report and a gigabyte resident — and four such
// files in one walk reached 4.7 GB. No author needs the ten-thousandth
// instance to act, and a document with this many is one problem, not ten
// thousand.
const findingLimit findingCount = 1000

// Diagnostics reports the changelog findings for one document: the file itself
// when its name is a changelog, and every heading that opens one.
//
// A document that is not text yields [ErrNotText], so the caller surfaces a
// tool failure rather than a clean pass over a file nobody read.
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
	if goyze.ByteCount(len(source)) > SizeLimit {
		return nil, 0, ErrTooLarge.With(nil, "path", string(at), "bytes", len(source))
	}
	if !utf8.ValidString(string(source)) {
		return nil, 0, ErrNotText.With(nil, "path", string(at))
	}
	base, ext := nameAndExtension(at)
	// Stripped ONCE, here, before the parse, so every reader below sees the same
	// text. A byte order mark ahead of `<!--` makes the line a paragraph rather
	// than an HTML comment — which is how a Windows editor silently exempts a
	// document from the claim its generator really did write.
	text := Source(strings.TrimPrefix(string(source), byteOrderMark))
	// The FILE finding is raised BEFORE the generated claim is even read, and
	// nothing exempts it. A changelog is banned because it EXISTS in the
	// repository, not because of who typed it — and reading the claim first made
	// the rule silent for exactly the files it most needs to catch, since
	// release-please, git-cliff and goreleaser all open their CHANGELOG.md with
	// one.
	diags := fileDiagnostics(at, base, ext)
	if !sectionExtensions[ext] {
		return diags, findingCount(len(diags)), nil
	}
	doc := parse(text)
	if isGenerated(doc, ext) {
		// Only the SECTIONS are out of scope. A generated document cannot be
		// fixed by editing it, so a machine-written docs page carrying four
		// hundred `## Unreleased` headings is one problem in its generator, and
		// reporting four hundred findings buries it.
		return diags, findingCount(len(diags)), nil
	}
	headings, total := headingDiagnostics(at, doc)
	// Counted BEFORE the notice is appended. The truncation notice is this
	// analyzer's own bookkeeping, not something the document contains, and
	// counting it inflated the run total by one per truncated document.
	held := findingCount(len(diags)) + total
	diags = append(diags, headings...)
	if total > findingLimit {
		diags = append(diags, truncation(at, total))
	}
	return diags, held, nil
}

// nameAndExtension is a path's final element and its lower-cased extension.
func nameAndExtension(at Path) (baseName, extension) {
	base := path.Base(string(at))
	return baseName(base), extension(strings.ToLower(path.Ext(base)))
}

// fileDiagnostics reports the file when its name is a changelog.
func fileDiagnostics(at Path, base baseName, ext extension) []goyze.Diagnostic {
	if !nameExtensions[ext] {
		return nil
	}
	stem := strings.TrimSuffix(strings.ToLower(string(base)), string(ext))
	if !changelogFileName.MatchString(stem) {
		return nil
	}
	return []goyze.Diagnostic{diagnostic(at, 1, finding(fmt.Sprintf(fileMessage, base)))}
}

// truncationMessage formats the finding that stands for the ones not reported.
const truncationMessage = "%d changelog findings in this document, of which %d are reported; a document with this " +
	"many is one problem rather than that many, and reporting them all costs more memory than reading it did"

// truncation is the finding that replaces everything past the limit, so the
// count is never silently lost.
func truncation(at Path, found findingCount) goyze.Diagnostic {
	return diagnostic(at, 1, finding(fmt.Sprintf(truncationMessage, found, findingLimit)))
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
