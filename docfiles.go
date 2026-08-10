// Package docfiles reports hand-maintained changelogs — the file, and the
// section inside another document that is one by another name.
//
// A changelog duplicates what git already records, attributed and immutable,
// and the duplicate is the copy that rots: the file says one thing, `git log`
// says another, and the reader has no way to tell which is stale. Release notes
// generated from tags at release time carry the same information without the
// second source of truth.
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
package docfiles

import (
	"fmt"
	"path"
	"strings"
	"unicode/utf8"

	errs "github.com/gomatic/go-error"
	goyze "github.com/gomatic/go-yze"
)

// ErrTooLarge reports a document past the size this rule will read. Prose is
// not megabytes; a file that big is generated output, a data dump, or a mistake,
// and reading it costs its own size in memory for a rule that cannot apply.
const ErrTooLarge errs.Const = "document is too large to analyze as prose"

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
const fileMessage = "%s is a hand-maintained changelog; git history is the changelog — " +
	"tags and generated release notes carry the same information without a second source of truth to rot"

// headingMessage formats a banned-section finding.
const headingMessage = "the %q section is a changelog inside another document; " +
	"git history already records this, and the copy is the one that goes stale"

// prose is the set of extensions a changelog is written in. A changelog is
// prose; `changelog.go` is code that manages the concept rather than an
// instance of it, and reporting it would be reporting the cure as the disease.
// The empty extension is here for the canonical Unix spelling, a bare
// `CHANGELOG` with no extension at all.
var prose = map[extension]bool{
	"":          true,
	".md":       true,
	".markdown": true,
	".txt":      true,
	".rst":      true,
	".adoc":     true,
}

// extension is a file's suffix, lower-cased.
type extension string

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
	// Stripped ONCE, here, so every reader below sees the same text. The
	// heading scan stripped it and the generated-claim scan did not, so a
	// generated file written with a byte order mark — which is how a Windows
	// editor saves one — had its claim hidden behind an invisible character on
	// exactly the line the claim has to be on.
	family := markupOf(ext)
	text := Source(strings.TrimPrefix(string(source), byteOrderMark))
	if isGenerated(text, family) {
		// A generated document is out of scope ENTIRELY, file and sections
		// alike. Exempting only the sections reported a machine-written
		// CHANGELOG.md as a hand-maintained changelog — recommending generated
		// release notes to a file that already was one.
		return nil, 0, nil
	}
	diags := fileDiagnostics(at, base, ext)
	if !prose[ext] {
		return diags, findingCount(len(diags)), nil
	}
	headings, total := headingDiagnostics(at, text, family)
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
	if !prose[ext] {
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
