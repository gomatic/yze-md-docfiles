package docfiles

// Deciding which lines are PROSE and which are an example the document is
// showing. The standards themselves display banned shapes inside fences,
// comments and listing blocks, so a rule that read those could not describe
// itself — and every hole in this model has been a rule that silenced itself
// for the rest of a file.

import (
	"regexp"
	"strings"
)

// scanner tracks the block a line sits in, so a heading written as an EXAMPLE
// is not read as a section. The standards themselves show banned shapes inside
// fences and comments, and a rule that reported those could not describe
// itself.
type scanner struct {
	open          fence
	markup        markup
	isInComment   bool
	isAfterTitle  bool
	isInDelimited bool
}

// delimiter is a reStructuredText or AsciiDoc block delimiter: a run of four or
// more punctuation characters standing alone. The SAME line is a section
// adornment when a title sits above it, which is why the scanner tracks whether
// the previous line was blank — an AsciiDoc listing block and an underlined
// heading are otherwise the same characters, and reading a listing block's
// contents as sections reported the examples a document was showing.
var delimiter = regexp.MustCompile(`^(-{4,}|={4,}|\.{4,}|\*{4,}|_{4,}|\+{4,}|~{4,})$`)

// fence is an open fenced code block: the character that opened it and how many
// of them there were.
type fence struct {
	length int
	marker byte
	isOpen bool
	isBare bool
}

// commentOpen and commentClose delimit an HTML comment, which markdown passes
// through untouched — a heading inside one is not a section of the document.
const (
	commentOpen  = "<!--"
	commentClose = "-->"
)

// minimumFence is how many markers open a fenced block, per CommonMark.
const minimumFence = 3

// step advances the scanner over one line, reporting whether that line is prose
// a heading may be read from.
func (s scanner) step(text line) (scanner, bool) {
	trimmed := strings.TrimSpace(string(text))
	was := s
	s.isAfterTitle = isTitleCandidate(line(trimmed))
	if state, isBlock := was.stepBlock(s, trimmed, isIndentedCode(text)); isBlock {
		return state, false
	}
	// An indented line is never a section title in the adornment formats: their
	// headings are flush-left, and an indented run is the literal block a
	// document is quoting rather than a section it is opening.
	return s, s.markup != adornmentMarkup || !isIndented(text)
}

// stepBlock advances the block state, reporting whether this line belongs to a
// block rather than to the prose.
func (s scanner) stepBlock(next scanner, trimmed string, isCode bool) (scanner, bool) {
	switch {
	case s.isInDelimited:
		next.isInDelimited = !delimiter.MatchString(trimmed)
		return next, true
	case s.open.isOpen:
		next.open = s.open.after(line(trimmed), isCode)
		return next, true
	case s.isInComment:
		next.isInComment = !strings.Contains(trimmed, commentClose)
		return next, true
	}
	return next.openBlock(trimmed, isCode, s.isAfterTitle)
}

// openBlock enters whatever block this line opens, if it opens one.
func (s scanner) openBlock(trimmed string, isCode, isAfterTitle bool) (scanner, bool) {
	switch {
	case !isCode && opensComment(line(trimmed)):
		s.isInComment = true
	case s.markup == adornmentMarkup && !isAfterTitle && delimiter.MatchString(trimmed):
		// The SAME run of dashes underlines a title above it and delimits a
		// listing block after anything else, so a document showing a banned
		// heading inside a block had its example reported as a section.
		s.isInDelimited = true
	default:
		opened, isFence := opening(line(trimmed))
		if !isFence || isCode || !s.markup.fences(opened.marker) {
			return s, false
		}
		s.open = opened
	}
	return s, true
}

// isTitleCandidate reports a line that could be the title of an underlined
// heading: something is written on it, and it is neither a block attribute nor
// a run of delimiter characters itself.
func isTitleCandidate(trimmed line) bool {
	if trimmed == "" || strings.HasPrefix(string(trimmed), "[") {
		return false
	}
	return !delimiter.MatchString(string(trimmed))
}

// opensComment reports a line that leaves an HTML comment open. The opener may
// sit anywhere on the line — `text <!--` comments out everything after it — and
// requiring it at the start left a heading on the next line reported as a
// section of a document that had commented it away.
func opensComment(trimmed line) bool {
	open := strings.LastIndex(string(trimmed), commentOpen)
	if open < 0 {
		return false
	}
	return !strings.Contains(string(trimmed)[open:], commentClose)
}

// after is the fence's state once one line inside it has been read. A block
// closes only on a run of ITS OWN marker, at least as long as the one that
// opened it and followed by nothing else — which is what makes a ```` block
// survive the ``` fences it wraps, and an info string like ```go not close
// anything.
func (f fence) after(trimmed line, isCode bool) fence {
	closing, ok := opening(trimmed)
	if ok && !isCode && closing.marker == f.marker && closing.length >= f.length && closing.isBare {
		return fence{}
	}
	return f
}

// opening reads a line as a fence delimiter, reporting whether it is one.
func opening(trimmed line) (fence, bool) {
	text := string(trimmed)
	if text == "" || (text[0] != '`' && text[0] != '~') {
		return fence{}, false
	}
	marker := text[0]
	length := len(text) - len(strings.TrimLeft(text, string(marker)))
	if length < minimumFence {
		return fence{}, false
	}
	return fence{
		length: length,
		marker: marker,
		isOpen: true,
		isBare: strings.TrimSpace(text[length:]) == "",
	}, true
}
