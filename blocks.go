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
	opened, isFence := opening(line(trimmed))
	switch {
	case isFence && !isCode && s.markup.fences(opened.marker):
		// Tested FIRST. A fenced block may carry an info string, and `<!--` is a
		// legal one — read as a comment opener instead, it commented away every
		// line after it and nothing ever closed it.
		s.open = opened
	case s.markup.hasComments() && !isCode && leavesCommentOpen(line(trimmed)):
		s.isInComment = true
	case s.markup.usesDelimitedBlocks() && !isCode && !isAfterTitle && delimiter.MatchString(trimmed):
		// The SAME run of dashes underlines a title above it and delimits a
		// listing block after anything else, so a document showing a banned
		// heading inside a block had its example reported as a section.
		s.isInDelimited = true
	default:
		return s, false
	}
	return s, true
}

// closesAndStaysClosed reports whether a comment is still open after this line,
// given that it began inside one.
//
// The closer is looked for in the RAW line: inside an HTML comment there are no
// code spans, so a backticked `-->` really does end it. Only the remainder,
// which is prose again, is asked the prose question.
func closesAndStaysClosed(trimmed line) bool {
	raw := string(trimmed)
	closed := strings.Index(raw, commentClose)
	if closed < 0 {
		return true
	}
	return leavesCommentOpen(line(raw[closed+len(commentClose):]))
}

// leavesCommentOpen reports a line that ends inside an HTML comment, having
// begun outside one.
//
// It is ONE scan rather than two questions, because the two disagreed. Asking
// "is there an opener outside a code span?" and then "is there a closer?" on
// DIFFERENT texts made `<!-- see ` + "`-->`" + “ open a comment that never
// closed: the closer was inside a code span, so the stripped text did not have
// it — while the identical span on a CONTINUATION line did close the comment,
// because that half reads the raw line. Three characters, an internal
// contradiction, and a one-line opt-out from every finding below it.
//
// Scanning once gets both right: outside a comment a code span hides an opener,
// and once an opener is found everything after it is raw until a closer.
func leavesCommentOpen(text line) bool {
	raw := string(text)
	spans := codeSpans(text)
	for at := 0; at < len(raw); {
		if width, isSpan := spans[at]; isSpan {
			at += int(width)
			continue
		}
		if !strings.HasPrefix(raw[at:], commentOpen) {
			at++
			continue
		}
		closed := strings.Index(raw[at:], commentClose)
		if closed < 0 {
			return true
		}
		at += closed + len(commentClose)
	}
	return false
}

// codeSpans is where each inline code span begins and how wide it is, computed
// in ONE pass over the line.
//
// It used to be answered per position, by scanning forward for a matching
// backtick run — which rescans the whole line for every run that never closes.
// A line of runs of strictly increasing length closes none of them, so the cost
// was superlinear in its length: eight megabytes of it, a size the limit
// deliberately admits, took 27 seconds and reported nothing, and four such files
// wedged the gate for two minutes.
func codeSpans(text line) map[int]width {
	runs := backtickRuns(text)
	next := searching(runsByLength(runs))
	spans := map[int]width{}
	for i := runIndex(0); int(i) < len(runs); i++ {
		closer, isClosed := next.after(runs, i)
		if !isClosed {
			// An unclosed run is literal text, and so is everything after it
			// until some later run closes.
			continue
		}
		spans[runs[i].at] = width(runs[closer].at + int(runs[closer].length) - runs[i].at)
		i = closer
	}
	return spans
}

// backtickRun is one run of backticks: where it starts and how long it is.
type backtickRunAt struct {
	at     int
	length runLength
}

// backtickRuns is every backtick run in the line, left to right.
func backtickRuns(text line) []backtickRunAt {
	var runs []backtickRunAt
	for at := 0; at < len(text); {
		if text[at] != '`' {
			at++
			continue
		}
		run := backtickRun(text[at:])
		runs = append(runs, backtickRunAt{at: at, length: run})
		at += int(run)
	}
	return runs
}

// runsByLength indexes the runs by length, so finding the next one of a given
// length is a step rather than a scan.
func runsByLength(runs []backtickRunAt) map[runLength][]int {
	byLength := map[runLength][]int{}
	for i, run := range runs {
		byLength[run.length] = append(byLength[run.length], i)
	}
	return byLength
}

// search finds each run's closer, remembering how far it has already looked at
// every length.
//
// The cursor is what makes the whole scan linear, and its absence is why the
// one-pass rewrite was still quadratic on the shape it was written for. Indexing
// by length turned "scan the line again" into "scan this length's runs again",
// which is the same cost when a line is nothing BUT runs of one length: ordinary
// prose with code spans has openers at candidate positions 0, 2, 4, …, and each
// restarted from the front. A megabyte of `x` spans took 39 seconds — 112 times
// slower than the pathological shape the regression test uses, at half the size
// — so an eight-megabyte file, which the size limit deliberately admits, was
// forty minutes of CPU in one call.
//
// A cursor never needs to go back: codeSpans asks about openers in increasing
// order, and a closer already passed can never be the answer for a later one.
type search struct {
	byLength map[runLength][]int
	cursor   map[runLength]int
}

// searching starts a search over runs already indexed by length.
func searching(byLength map[runLength][]int) search {
	return search{byLength: byLength, cursor: map[runLength]int{}}
}

// after is the first run past opener whose length matches — the closer
// CommonMark asks for, which is a backtick string of the same length and no
// other.
func (s search) after(runs []backtickRunAt, opener runIndex) (runIndex, bool) {
	length := runs[opener].length
	candidates := s.byLength[length]
	at := s.cursor[length]
	for at < len(candidates) && runIndex(candidates[at]) <= opener {
		at++
	}
	s.cursor[length] = at
	if at == len(candidates) {
		return 0, false
	}
	return runIndex(candidates[at]), true
}

// runIndex is a position in the list of backtick runs on one line.
type runIndex int

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

// width is how many bytes of a line a construct occupies.
type width int

// runLength is how many backticks stand together.
type runLength int

// backtickRun is the length of the backtick run beginning the text.
func backtickRun(text line) runLength {
	return runLength(len(text) - len(strings.TrimLeft(string(text), "`")))
}
