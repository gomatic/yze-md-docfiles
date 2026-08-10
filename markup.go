package docfiles

// The markup family a document belongs to, and everything the three formats
// disagree about. They were two families for as long as reStructuredText and
// AsciiDoc were believed to differ only in how they underline a title; they
// differ in whether a marker run opens one, which characters may underline one,
// and whether a standalone run of punctuation delimits a block at all.

import "regexp"

// markdownHeading is markdown's `## Title`. CommonMark allows up to three
// leading spaces and an optional closing run of hashes, so `   ## Changelog ##`
// is a heading and anchoring hard at the first column let both spellings
// through.
var markdownHeading = regexp.MustCompile(`^ {0,3}#{1,6}[ \t]+(.*?)[ \t]*#*[ \t]*$`)

// asciidocHeading is AsciiDoc's `== Title`, and the `## Title` form Asciidoctor
// accepts for Markdown compatibility. Section titles begin at column zero.
//
// It is a SEPARATE pattern from markdown's rather than one alternation serving
// both, because the `=` half is not symmetric: `= Changelog` is not a heading in
// markdown, it is ordinary prose, and a `=` run means something there only on
// the line BELOW a title. One pattern for both families read that prose as a
// section.
var asciidocHeading = regexp.MustCompile(`^(?:#{1,6}|={1,6})[ \t]+(.*?)[ \t]*[#=]*[ \t]*$`)

// markup is the family a document is written in. The three disagree about
// things that decide whether a line is prose: whether `~~~` opens a fenced code
// block or underlines a section title, which characters may underline a title,
// whether a marker run opens one, and whether a standalone run of punctuation
// delimits a block at all.
//
// reStructuredText and AsciiDoc were ONE family here, on the grounds that both
// underline their titles. They do — and they disagree about everything else.
// AsciiDoc's delimited block (`----` … `----`) does not exist in RST, where the
// same characters are a transition, or the overline of a title; reading them as
// a block that never closed made every heading below invisible. One shared
// value cannot answer a question the two formats answer differently.
type markup int

const (
	// markdownMarkup is markdown and everything read as it.
	markdownMarkup markup = iota
	// restructuredMarkup is reStructuredText: titles underlined (and
	// optionally overlined) with repeated punctuation, literal blocks by
	// indentation, and no delimited block.
	restructuredMarkup
	// asciidocMarkup is AsciiDoc: titles underlined or written `== Title`,
	// and blocks delimited by a standalone run of four or more punctuation
	// characters.
	asciidocMarkup
)

// markupExtensions are the prose extensions that are not read as markdown.
var markupExtensions = map[extension]markup{".rst": restructuredMarkup, ".adoc": asciidocMarkup}

// markupOf is the family a document's extension puts it in.
func markupOf(ext extension) markup {
	if family, ok := markupExtensions[ext]; ok {
		return family
	}
	return markdownMarkup
}

// usesAdornments reports a family whose section titles are underlined with a
// repeated punctuation character rather than markdown's two.
func (m markup) usesAdornments() bool {
	return m == restructuredMarkup || m == asciidocMarkup
}

// usesDelimitedBlocks reports a family that delimits a block with a standalone
// run of punctuation. Only AsciiDoc does: in RST the same line is a transition
// or a title's overline, and reading it as an unterminated block silenced the
// rest of the document.
func (m markup) usesDelimitedBlocks() bool {
	return m == asciidocMarkup
}

// heading is the marker-run heading pattern this family spells its titles with,
// reporting whether it has one at all.
//
// Every family is named and the result is returned once, so there is no branch
// here that no document can reach: a switch returning from each arm needs a
// trailing return the compiler demands and no input can execute, which is a
// line that can only be covered by pretending.
func (m markup) heading() (*regexp.Regexp, bool) {
	var pattern *regexp.Regexp
	switch m {
	case markdownMarkup:
		pattern = markdownHeading
	case asciidocMarkup:
		pattern = asciidocHeading
	case restructuredMarkup:
		// No marker-run form at all. Its `#` is an adornment character, so
		// reading a `#` line as a heading invents sections in a format that
		// has none.
	}
	return pattern, pattern != nil
}

// underlines reports a line that underlines the title above it.
func (m markup) underlines(text line) bool {
	if m.usesAdornments() {
		return adornmentUnderline.MatchString(string(text))
	}
	return setextUnderline.MatchString(string(text))
}

// fences reports whether this family spells a fenced code block with the given
// marker. Only markdown does, and `~~~` in reStructuredText is a section
// adornment — reading it as a fence silenced every heading in the rest of the
// file, which is how a `Changelog` section under a tilde rule went unreported.
func (m markup) fences(marker byte) bool {
	return m == markdownMarkup && (marker == '`' || marker == '~')
}
