package docfiles

// Deciding which lines are PROSE and which are an example the document is
// showing. The standards themselves display banned shapes inside fences,
// comments and listing blocks, so a rule that read those could not describe
// itself — and every hole in this model has been a rule that silenced itself
// for the rest of a file.

import (
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
	// document is quoting rather than a section it is opening. In
	// reStructuredText this is the ONLY block model — a literal block, a
	// directive's body and a quoted example are all just indentation.
	return s, !s.markup.usesAdornments() || !isIndented(text)
}

// stepBlock advances the block state, reporting whether this line belongs to a
// block rather than to the prose.
func (s scanner) stepBlock(next scanner, trimmed string, isCode bool) (scanner, bool) {
	switch {
	case s.isInDelimited:
		// An INDENTED delimiter does not close the block: AsciiDoc and RST
		// require a delimiter at column zero, so an indented run is the
		// literal content the block is showing. The fence pair already knew
		// this; the delimited pair did not, and closing early reported the
		// example a document was quoting.
		next.isInDelimited = isCode || !delimiter.MatchString(trimmed)
		return next, true
	case s.open.isOpen:
		next.open = s.open.after(line(trimmed), isCode)
		return next, true
	case s.isInComment:
		next.isInComment = closesAndStaysClosed(line(trimmed))
		return next, true
	}
	return next.openBlock(trimmed, isCode, s.isAfterTitle)
}

// openBlock enters whatever block this line opens, if it opens one.
func (s scanner) openBlock(trimmed string, isCode, isAfterTitle bool) (scanner, bool) {
	switch {
	case s.markup == markdownMarkup && !isCode && opensComment(line(trimmed)):
		s.isInComment = true
	case s.markup.usesDelimitedBlocks() && !isCode && !isAfterTitle && delimiter.MatchString(trimmed):
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

// closesAndStaysClosed reports whether a comment is still open after this line.
// A line may close one and open another — `--> visible <!--` — so finding a
// closer is not the end of the question; the opener half already knew that.
//
// The closer is looked for in the RAW line, deliberately unlike [opensComment].
// Inside an HTML comment there are no code spans: a comment is raw text, and a
// backtick in it is a backtick. Stripping spans here would let a `-->` written
// inside one fail to close a comment that markdown really does end there.
func closesAndStaysClosed(trimmed line) bool {
	if !strings.Contains(string(trimmed), commentClose) {
		return true
	}
	_, after, _ := strings.Cut(string(trimmed), commentClose)
	// The remainder is prose again, so it is asked the prose question.
	return opensComment(line(after))
}

// opensComment reports a line that leaves an HTML comment open. The opener may
// sit anywhere on the line — `text <!--` comments out everything after it — and
// requiring it at the start left a heading on the next line reported as a
// section of a document that had commented it away.
//
// Inline code spans are removed first. A backticked `<!--` is markdown SHOWING
// the opener, not using it, and reading it as a real one gave any document that
// explains markdown comments — this rule's own documentation among them — a
// one-line, unlogged opt-out from every finding below it.
func opensComment(trimmed line) bool {
	text := string(withoutCodeSpans(trimmed))
	open := strings.LastIndex(text, commentOpen)
	if open < 0 {
		return false
	}
	return !strings.Contains(text[open:], commentClose)
}

// runLength is how many backticks stand together.
type runLength int

// offset is a position within a line, or [notFound].
type offset int

// notFound is the [offset] of something that is not there.
const notFound offset = -1

// withoutCodeSpans is the line with its inline code spans removed. Per
// CommonMark a span runs from a backtick string to the next backtick string of
// the same length; an unclosed run is literal text and is left as written.
func withoutCodeSpans(text line) line {
	var out strings.Builder
	rest := string(text)
	for {
		start := strings.IndexByte(rest, '`')
		if start < 0 {
			_, _ = out.WriteString(rest)
			return line(out.String())
		}
		open := backtickRun(line(rest[start:]))
		closed := closingRun(line(rest[start+int(open):]), open)
		if closed == notFound {
			_, _ = out.WriteString(rest)
			return line(out.String())
		}
		_, _ = out.WriteString(rest[:start])
		rest = rest[start+int(open)+int(closed)+int(open):]
	}
}

// backtickRun is the length of the backtick run beginning the text.
func backtickRun(text line) runLength {
	return runLength(len(text) - len(strings.TrimLeft(string(text), "`")))
}

// closingRun is the offset of the next backtick run of the same length, or
// [notFound] when the span is left open.
func closingRun(text line, length runLength) offset {
	for at := 0; at < len(text); {
		next := strings.IndexByte(string(text)[at:], '`')
		if next < 0 {
			return notFound
		}
		at += next
		run := backtickRun(text[at:])
		if run == length {
			return offset(at)
		}
		at += int(run)
	}
	return notFound
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
