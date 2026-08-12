package docfiles

import (
	"fmt"

	errs "github.com/gomatic/go-error"
	goyze "github.com/gomatic/go-yze"
)

// What this rule refuses to read, and how much of what it finds a report may
// carry.
//
// Three bounds, at three scales, each added after the one below it turned out
// not to cover the case above it: a document too large or too malformed to read
// at all, a document holding more findings than anyone can act on, and a RUN
// holding more than that.
//
// None of them bounds what a file's NAME earns. A name is knowable from a
// directory entry, so it survives every refusal here — see the package doc.

// ErrTooLarge reports a file past the size this rule will read. It IS the shared
// sentinel, not a second one beside it: the bound is enforced in two places —
// here, and at the one place the command reads a file — and two sentinels for
// one condition meant `errors.Is` answered false for whichever layer the caller
// had not thought of.
const ErrTooLarge = goyze.ErrTooLarge

// SizeLimit is the largest document read, in bytes.
//
// It is exported so the command can bound its own read at the same number this
// package refuses at, which is what keeps a named argument and a walked file
// from disagreeing. The bound is on the READ, not on a directory entry:
// [goyze.BoundedRead] opens the file and reads one byte past the limit, then
// refuses — which is deliberate, because a size asked for beforehand describes a
// PATH rather than the bytes behind it, and a stat that did not follow a symlink
// once reported a link's own few bytes while two gigabytes were read through it.
//
// It is generous by three orders of magnitude over any real document, so it
// bounds the pathological case without ever turning away prose. What it does NOT
// bound is a finding about the file's NAME, which needs no bytes at all.
const SizeLimit goyze.ByteCount = 8 << 20

// ErrNotText reports a document whose bytes are not text. A binary file cannot
// be read as prose, and guessing at its lines would invent findings from
// whatever byte happened to look like a `#`.
const ErrNotText errs.Const = "document is not valid UTF-8 text"

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

// truncationMessage formats the finding that stands for the ones not reported.
const truncationMessage = "%d changelog findings in this document, of which %d are reported; a document with this " +
	"many is one problem rather than that many, and reporting them all costs more memory than reading it did"

// truncation is the finding that replaces everything past the limit, so the
// count is never silently lost.
func truncation(at Path, found findingCount) goyze.Diagnostic {
	return diagnostic(at, 1, finding(fmt.Sprintf(truncationMessage, found, findingLimit)))
}

// reportLimit bounds how many findings ONE RUN carries.
//
// The per-document limit bounds a document; it does not bound a tree. Two
// thousand documents each at their own limit produced 490 MB of report and
// 2.3 GB resident from 31 MB of input — the same failure the per-document
// limit exists to prevent, reached by a route it does not cover. A run with
// this many findings is one problem too, and the true count is still named.
const reportLimit = 10_000

// runTruncationMessage formats the finding that stands for the rest of a run.
const runTruncationMessage = "%d changelog findings across this run, of which %d are reported; findings from %s " +
	"onward are omitted, because a tree with this many is one problem rather than that many"
