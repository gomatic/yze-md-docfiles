package docfiles

import (
	"fmt"
	"regexp"
	"strings"

	goyze "github.com/gomatic/go-yze"
)

// line is one physical line of a document, without its terminator.
type line string

// lineNumber is a 1-based position in a document.
type lineNumber int

// byteOrderMark is the UTF-8 BOM some editors write. It precedes the first
// character while being invisible in the text, so a heading check that does not
// strip it silently exempts the first line of the file.
const byteOrderMark = "\ufeff"

// changelogFileName is the file this rule bans, in the spellings it is written
// in: CHANGELOG, ChangeLog, changelog, CHANGES, and the hyphenated and
// underscored variants. The name must be the WHOLE stem — `changelog-policy.md`
// is a document ABOUT the policy, which is a thing worth keeping.
var changelogFileName = regexp.MustCompile(`^(change[-_]?log|changes|release[-_]?notes)$`)

// changelogTitle is a heading that opens a changelog section. The alternatives
// are spelled out rather than matched loosely, because "Change Process" is not
// a changelog and a rule that guessed would report it — and the plural
// "Changelogs" is deliberately absent, because the fleet's one instance is the
// documentation standard describing this very rule.
var changelogTitle = regexp.MustCompile(
	`^(?i)(change[-_ ]?log|recent changes|version history|release history|revision history|what'?s new|unreleased)$`,
)

// atxHeading is a heading written with a leading marker run: markdown's
// `## Title`, and AsciiDoc's `== Title`. CommonMark allows up to three leading
// spaces and an optional closing run of hashes, so `   ## Changelog ##` is a
// heading and anchoring hard at the first column let both spellings through.
var atxHeading = regexp.MustCompile(`^ {0,3}(?:#{1,6}|={1,6})[ \t]+(.*?)[ \t]*#*[ \t]*$`)

// underline is the second line of a two-line heading: markdown's setext form
// and reStructuredText's, where the title sits on the line ABOVE.
var underline = regexp.MustCompile(`^ {0,3}(?:=+|-+)[ \t]*$`)

// headingDiagnostics reports every heading that opens a changelog section.
func headingDiagnostics(at Path, source Source) []goyze.Diagnostic {
	lines := documentLines(source)
	var diags []goyze.Diagnostic
	state := scanner{}
	for i, text := range lines {
		var isProse bool
		state, isProse = state.step(text)
		if !isProse {
			continue
		}
		if title, ok := heading(text, next(lines, i)); ok {
			diags = append(diags, diagnostic(at, lineNumber(i+1), headingFinding(title)))
		}
	}
	return diags
}

// documentLines is a document's lines, with the byte-order mark and carriage returns
// removed so a file written on Windows or by an editor that stamps a BOM reads
// exactly as one written on Unix.
func documentLines(source Source) []line {
	text := strings.TrimPrefix(string(source), byteOrderMark)
	raw := strings.Split(text, "\n")
	lines := make([]line, 0, len(raw))
	for _, one := range raw {
		lines = append(lines, line(strings.TrimSuffix(one, "\r")))
	}
	return lines
}

// next is the line after i, or empty at the end of the document.
func next(lines []line, i int) line {
	if i+1 < len(lines) {
		return lines[i+1]
	}
	return ""
}

// heading is the changelog title a line opens, if it opens one. Both written
// forms are read: the marker-run heading, and the two-line form whose title
// sits above its underline.
func heading(text, following line) (line, bool) {
	if found := atxHeading.FindStringSubmatch(string(text)); found != nil {
		return matched(line(found[1]))
	}
	if underline.MatchString(string(following)) {
		return matched(line(strings.TrimSpace(string(text))))
	}
	return "", false
}

// matched reports a title when it names a changelog. The title arrives already
// trimmed — the marker-run pattern consumes the space on both sides of it, and
// the two-line form trims its own — so trimming again here would be code no
// input could distinguish, which is code no test can pin.
func matched(title line) (line, bool) {
	return title, changelogTitle.MatchString(string(title))
}

// headingFinding is the message for one banned section. The title is quoted
// with strconv, so a heading holding a quote, a tab or a backslash is rendered
// unambiguously rather than pasted into the sentence to be misread.
func headingFinding(title line) finding {
	return finding(fmt.Sprintf(headingMessage, string(title)))
}

// scanner tracks the block a line sits in, so a heading written as an EXAMPLE
// is not read as a section. The standards themselves show banned shapes inside
// fences and comments, and a rule that reported those could not describe
// itself.
type scanner struct {
	open        fence
	isInComment bool
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
	switch {
	case s.open.isOpen:
		s.open = s.open.after(trimmed)
		return s, false
	case s.isInComment:
		s.isInComment = !strings.Contains(trimmed, commentClose)
		return s, false
	case strings.HasPrefix(trimmed, commentOpen) && !strings.Contains(trimmed, commentClose):
		s.isInComment = true
		return s, false
	}
	if opened, ok := opening(trimmed); ok {
		return scanner{open: opened}, false
	}
	return s, true
}

// after is the fence's state once one line inside it has been read. A block
// closes only on a run of ITS OWN marker, at least as long as the one that
// opened it and followed by nothing else — which is what makes a ```` block
// survive the ``` fences it wraps, and an info string like ```go not close
// anything.
func (f fence) after(trimmed string) fence {
	closing, ok := opening(trimmed)
	if ok && closing.marker == f.marker && closing.length >= f.length && closing.isBare {
		return fence{}
	}
	return f
}

// opening reads a line as a fence delimiter, reporting whether it is one.
func opening(trimmed string) (fence, bool) {
	if trimmed == "" || (trimmed[0] != '`' && trimmed[0] != '~') {
		return fence{}, false
	}
	marker := trimmed[0]
	length := len(trimmed) - len(strings.TrimLeft(trimmed, string(marker)))
	if length < minimumFence {
		return fence{}, false
	}
	return fence{
		marker: marker,
		length: length,
		isOpen: true,
		isBare: strings.TrimSpace(trimmed[length:]) == "",
	}, true
}
