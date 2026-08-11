package docfiles

// Markdown's two remaining block models: the raw HTML block, whose lines the
// renderer passes through untouched, and the list marker a fence may open
// beside. Each was a false positive — markup a document was SHOWING, reported as
// markup it was making.

import "regexp"

// stepRawHTML advances the markdown HTML-block state, reporting whether this
// line belongs to one.
//
// A block opens on a line beginning with a tag and closes at the first blank
// line, which is what CommonMark specifies for the HTML blocks a document
// actually writes. Only markdown has them: the other families pass HTML through
// by other means, and inventing a block model for them would suppress headings
// they really do open.
func (s scanner) stepRawHTML(next scanner, trimmed blockLine) (scanner, bool) {
	if s.markup.usesAdornments() {
		return next, false
	}
	if s.isInRawHTML {
		next.isInRawHTML = trimmed != ""
		return next, true
	}
	if !htmlBlockOpener.MatchString(string(trimmed)) {
		return next, false
	}
	next.isInRawHTML = true
	return next, true
}

// htmlBlockOpener is a line that starts a raw HTML block: an opening or closing
// tag at the head of the line. A comment is deliberately absent — the scanner
// tracks those itself, and the generated-claim reader depends on it.
var htmlBlockOpener = regexp.MustCompile(`^</?[a-zA-Z][a-zA-Z0-9-]*(?:[ \t>/]|$)`)

// withoutListMarker is the line's content with a leading list bullet or number
// removed, because a fence may open on the same line as the marker that
// introduces it.
//
// CommonMark puts the item's content on that line — goldmark renders "- ```" as
// a list item holding a code block — and requiring the fence to be the first
// thing on the line left the block closed, so every line inside it was
// read as prose and a `# Changelog` a document was SHOWING became a finding
// nobody could act on.
func withoutListMarker(text blockLine) blockLine {
	if marker := listMarker.FindString(string(text)); marker != "" {
		return text[len(marker):]
	}
	return text
}

// blockLine is one line of a document as the block scanner reads it, trimmed of
// the surrounding whitespace that decides indentation elsewhere.
type blockLine string

// listMarker is a bullet or an ordered-list number introducing an item, with the
// space that separates it from the item's content.
var listMarker = regexp.MustCompile(`^(?:[-+*]|[0-9]{1,9}[.)])[ \t]+`)
