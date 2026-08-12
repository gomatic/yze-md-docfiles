package docfiles

// The parse this rule reads a document with, and the map from a parser position
// back to the line an author's editor shows.
//
// Nothing here decides what a heading is, what a comment is, or which lines a
// document merely SHOWS rather than writes. That is the whole point of parsing
// instead of scanning: every construct that looks like a section and is not one
// — a fence, an indented code block, an HTML comment, a list item's own content,
// a raw HTML block — is a different node to the parser, while a scanner has to
// re-decide each of them separately and be wrong somewhere. This package's own
// history is a catalogue of exactly that, and every entry in it is a shape that
// silenced the rest of a file.

import (
	"sort"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	mdtext "github.com/yuin/goldmark/text"
)

// markdown is the parser every document is read with.
//
// It is built ONCE. A parser is an immutable binding, assembled at construction
// and unchanged thereafter, so rebuilding it per document would rebuild the same
// block-parser table for every file in a fleet-wide walk.
var markdown = blockParser()

// blockParser assembles CommonMark's BLOCK grammar: its block parsers and the
// paragraph transformers that finish the job. No inline parsers, and no
// extensions.
//
// Neither omission loses a heading. This rule takes a heading's text from the
// SOURCE, so an inline tree would be built for nobody to consult; and the
// extensions a sibling analyzer enables — GFM, footnotes, definition lists —
// reshape what a PARAGRAPH is rather than what a heading is. Measured
// 2026-08-12 over 38 written forms: heading detection is identical with and
// without them, in every one. What taking the SOURCE text does cost is stated in
// the package doc, under the limitations, and it is not nothing.
//
// The inline parsers are absent for a second reason, and it is a bound rather
// than an economy. Goldmark's code-span parser is superlinear in the number of
// backtick runs on one line: a megabyte of `x` spans takes 43.9 seconds through
// the full parser and 5 milliseconds through this one, and eight megabytes — a
// size [SizeLimit] deliberately admits — takes 40 milliseconds here. That is the
// same 39-seconds-per-megabyte disaster this package once repaired by hand in
// its own code-span scanner, reintroduced verbatim by the library, and the
// regression tests that guarded the hand-rolled version now guard this choice.
//
// IT IS NOT THE WHOLE BOUND, and an earlier version of this comment said it was.
// Goldmark's BLOCK grammar is quadratic in container nesting depth, which no
// choice here reaches: 400 KB of `>` takes 50 seconds and a megabyte takes 5m42s
// at 721 MB resident, so a checked-in file well under [SizeLimit] can still cost
// hours. That is an open defect against the parse itself and it is written down
// in the package doc rather than left in a claim nothing stands behind.
func blockParser() parser.Parser {
	return parser.NewParser(
		parser.WithBlockParsers(parser.DefaultBlockParsers()...),
		// The paragraph transformers are part of the BLOCK grammar, however they
		// are packaged: the one in the default set lifts link reference
		// definitions out of the paragraph they were read into. Omitting them
		// left `[x]: /y` sitting inside the paragraph above a setext underline,
		// so the heading's text spanned two lines and was discarded — one
		// ordinary line of link definitions above a `Changelog` heading silenced
		// it. Found by an adversarial review.
		parser.WithParagraphTransformers(parser.DefaultParagraphTransformers()...),
	)
}

// document is one parsed document: the bytes that were parsed, the block tree
// over them, and where its lines begin.
type document struct {
	source []byte
	root   ast.Node
	lines  lineIndex
}

// parse reads one document's block structure.
func parse(text Source) document {
	source := []byte(text)
	return document{
		source: source,
		root:   markdown.Parse(mdtext.NewReader(source)),
		lines:  newLineIndex(source),
	}
}

// textOf is the source text a parser segment covers.
func (d document) textOf(at mdtext.Segment) []byte {
	return d.source[at.Start:at.Stop]
}

// lineOf is the 1-based line a parser position sits on.
func (d document) lineOf(at mdtext.Segment) lineNumber {
	return d.lines.of(byteOffset(at.Start))
}

// byteOffset is a position in a document's bytes.
type byteOffset int

// lineIndex is where each line of a document starts, so a byte offset from the
// parser becomes the line number an author's editor shows.
type lineIndex []byteOffset

// newLineIndex indexes a document's lines.
func newLineIndex(text []byte) lineIndex {
	starts := lineIndex{0}
	for at := range len(text) {
		if text[at] == '\n' {
			starts = append(starts, byteOffset(at+1))
		}
	}
	return starts
}

// of is the 1-based line holding a byte offset.
//
// The result is never zero and needs no guard saying so: the index always begins
// at offset 0 and every position the parser reports is non-negative, so the
// search can never stop before the first entry. A branch for it would be one no
// input can take — a check that looks like one and tests nothing.
func (l lineIndex) of(at byteOffset) lineNumber {
	return lineNumber(sort.Search(len(l), func(i int) bool { return l[i] > at }))
}
