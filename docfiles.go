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
// # WHAT WAS REFUSED, AND WHAT IT COST
//
// Every spelling was counted across the fleet before the decision, and the
// refusals matter as much as the admissions. A rule that fires on any of these
// is a rule its own repository cannot document itself under.
//
// Refused on a LEGITIMATE INSTANCE, which is the strongest reason there is:
//
//   - A bare `Changes` or `History` HEADING — ordinary prose in a design
//     document.
//   - `Release Notes` as a HEADING — three fleet instances, all three
//     legitimate: Go's own documentation ABOUT writing release notes, and two
//     docs-site index sections. It is banned as a file NAME, where it is
//     unambiguous, and left alone as a heading.
//   - `history.md` — two fleet instances, both genuine Hugo content pages about
//     a project's history.
//   - `changelog.go` — code that manages the concept rather than an instance of
//     it. `changelog_test.go` beside it is the fleet's only DECORATED instance,
//     and it is code twice over.
//   - A decoration that is a WORD: `changelog-policy.md` is a document about the
//     policy. Loosening the whole-stem anchor to a prefix or a substring reports
//     it, and that is why the anchor holds.
//
// Refused at ZERO instances, deliberately, and recorded so a later measurement
// can revisit rather than rediscover (all counts 2026-08-12, over 647 repository
// trees and 35,663 walked files, the work account excluded):
//
//   - `NEWS.md` — 0 files. It is the GNU and R ecosystems' changelog, and it is
//     also an ordinary page title on a website, in a fleet built out of Hugo
//     sites. That is precisely the shape `history.md` was refused for, on two
//     real instances, and admitting the one while refusing the other would make
//     the rule's narrowness a matter of which word came up first.
//   - `CHANGELOG.old` — 0 files. Admitting it means admitting a backup
//     vocabulary, and that vocabulary has to cover `changelog.md.bak` too — the
//     commoner spelling, already pinned unreported. Half a vocabulary is worse
//     than none: the rule's answer would depend on which backup convention an
//     author happened to use.
//
// Admitted at ZERO instances, because each is a real spelling of the banned
// thing with no legitimate document standing in its way: the numeric decoration
// (`CHANGELOG-1.29.md`, Kubernetes' literal layout; `CHANGELOG_2024.md`), the
// dotfile spelling (`.changelog.md`), the changelog kept as a DIRECTORY
// (`changelog/index.md` — 0 such directories fleet-wide), and the two missing
// extensions `.mdx` and `.asciidoc` — of which the fleet holds 0 files of ANY
// name. Each admits a spelling rather than reporting one, which is the whole
// point of measuring first.
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
//
// # A NAME OUTLIVES EVERY REASON THE BYTES COULD NOT BE READ
//
// A file that cannot be read is reported as a tool failure — [ErrTooLarge] for
// one past [SizeLimit], [ErrNotText] for one whose bytes are not text, and the
// reader's own error for one nobody could open. None of those suppresses what
// the rule already knows.
//
// A tool failure once meant NO findings, on the reasoning that an analyzer which
// read nothing has nothing to say. That reasoning holds for a section and fails
// for a name: an oversized `CHANGELOG.md` was told the analyzer could not read
// it, when what needed saying is that it must not exist. The size and text
// guards describe a file's CONTENTS, and they were deciding a rule that depends
// only on its NAME.
//
// So the contract is that BOTH are said. An unreadable document yields the
// findings its name earns — never any others, because nothing else was
// determined — together with the error, and every reader of this package
// composes them the same way: the name findings first, then the one that says
// the file could not be read. [Report] counts both, so a run's total is still
// the true one. Nothing is passed over in silence, which is the outcome a gate
// must never produce, and nothing is invented for a file nobody read either — an
// innocent oversized document still yields exactly one finding, the unreadable
// one.
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
	base, ext := nameAndExtension(at)
	// The NAME is decided FIRST, ahead of everything that reads a byte, and
	// nothing below can take it back. It is raised before the generated claim,
	// because a changelog is banned for EXISTING rather than for who typed it —
	// and reading the claim first made the rule silent for exactly the files it
	// most needs to catch, since release-please, git-cliff and goreleaser all
	// open their CHANGELOG.md with one. It is raised before the size and text
	// guards for the same reason one step further out: those describe the
	// CONTENTS, and a name is knowable with zero bytes read, so letting them
	// answer first told an author the analyzer had trouble reading a file whose
	// whole problem is that it is there.
	diags := fileDiagnostics(at, base, ext)
	if goyze.ByteCount(len(source)) > SizeLimit {
		return diags, findingCount(len(diags)), ErrTooLarge.With(nil, "path", string(at), "bytes", len(source))
	}
	if !utf8.ValidString(string(source)) {
		return diags, findingCount(len(diags)), ErrNotText.With(nil, "path", string(at))
	}
	// Stripped ONCE, here, before the parse, so every reader below sees the same
	// text. A byte order mark ahead of `<!--` makes the line a paragraph rather
	// than an HTML comment — which is how a Windows editor silently exempts a
	// document from the claim its generator really did write.
	text := Source(strings.TrimPrefix(string(source), byteOrderMark))
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

// directoryMessage formats the finding for a document inside a changelog
// DIRECTORY. It names the directory as well as the file, because the file's own
// name is innocent — `index.md` says nothing at all — and an author reading a
// finding about `index.md` would have no idea what to do with it.
const directoryMessage = "%s sits in the %s directory, which is a changelog kept as a directory and must not be " +
	"committed to a repository; git history is the changelog, and release notes belong to the tag and the release"
