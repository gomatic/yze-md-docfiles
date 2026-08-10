package docfiles

// The block forms that are NOT markdown's. AsciiDoc delimits a block with a
// standalone run of punctuation and captions it on the line directly above;
// reStructuredText has no delimited block at all, and the same characters there
// are a transition or a title's overline. The two were read as one family for
// as long as they were believed to disagree only about underlines, and every
// heading below an RST transition was invisible.

import (
	"regexp"
	"strings"
)

// delimiter is a reStructuredText or AsciiDoc block delimiter: a run of four or
// more punctuation characters standing alone. `////` is AsciiDoc's block
// COMMENT, which is a block like any other here — a heading a document has
// deliberately commented out is not a section of it. The SAME line is a section
// adornment when a title sits above it, which is why the scanner tracks whether
// the previous line was blank — an AsciiDoc listing block and an underlined
// heading are otherwise the same characters, and reading a listing block's
// contents as sections reported the examples a document was showing.
var delimiter = regexp.MustCompile(`^(-{4,}|={4,}|\.{4,}|\*{4,}|_{4,}|\+{4,}|~{4,}|/{4,})$`)

// isTitleCandidate reports a line that could be the title of an underlined
// heading: something is written on it, it is not a run of delimiter characters
// itself, and it is not one of AsciiDoc's two block-metadata lines.
//
// Both metadata forms sit IMMEDIATELY above the delimiter they describe, with
// no blank line permitted, which is exactly the position a title occupies. The
// attribute line `[source,go]` was already excluded; the block TITLE `.Example
// listing` was not, so a captioned block — the ordinary way to label an example
// — was read as an underlined heading and its delimiter never opened the block.
// Everything the block was showing was then read as prose.
func isTitleCandidate(trimmed line) bool {
	if trimmed == "" || strings.HasPrefix(string(trimmed), "[") {
		return false
	}
	if isBlockTitle(trimmed) {
		return false
	}
	return !delimiter.MatchString(string(trimmed))
}

// isBlockTitle reports AsciiDoc's `.Title` line. A run of four or more dots is
// a delimiter rather than a title, and is left to [delimiter] to recognise.
func isBlockTitle(trimmed line) bool {
	text := string(trimmed)
	return len(text) > 1 && text[0] == '.' && text[1] != '.'
}
