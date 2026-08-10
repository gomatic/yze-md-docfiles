package docfiles

import (
	"fmt"
	"regexp"
	"strings"

	goyze "github.com/gomatic/go-yze"
)

// line is one physical line of a document, without its terminator.
type line string

// lineNumber is a 1-based position in a document, as a diagnostic reports it.
type lineNumber int

// remaining is the not-yet-read tail of a document being streamed.
type remaining string

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

// setextUnderline is the second line of markdown's two-line heading, where the
// title sits on the line ABOVE. Markdown recognises two adornment characters
// here; reStructuredText recognises many more, which is what [adornmentUnderline]
// is for.
var setextUnderline = regexp.MustCompile(`^ {0,3}(?:=+|-+)[ \t]*$`)

// adornmentUnderline is reStructuredText's and AsciiDoc's equivalent. Their
// section adornment is ANY repeated punctuation, not just markdown's two, so a
// `Changelog` underlined with tildes or carets was invisible to a pattern that
// only knew `=` and `-` — which is most of the RST headings anyone writes.
var adornmentUnderline = regexp.MustCompile(`^ {0,3}(?:=+|-+|~+|\^+|"+|\++|#+|\*+|'+|` + "`+" + `|:+|\.+|_+)[ \t]*$`)

// markup is the family a document is written in. It decides two things that
// disagree between them: whether `~~~` opens a fenced code block (markdown) or
// underlines a section title (reStructuredText), and which characters may
// underline a title at all.
type markup int

const (
	// markdownMarkup is markdown and everything read as it.
	markdownMarkup markup = iota
	// adornmentMarkup is reStructuredText and AsciiDoc, whose sections are
	// underlined with repeated punctuation and which have no `~~~` fence.
	adornmentMarkup
)

// adornmentExtensions are the prose extensions whose sections are underlined
// with adornments rather than fenced with tildes.
var adornmentExtensions = map[extension]bool{".rst": true, ".adoc": true}

// markupOf is the family a document's extension puts it in.
func markupOf(ext extension) markup {
	if adornmentExtensions[ext] {
		return adornmentMarkup
	}
	return markdownMarkup
}

// underlines reports a line that underlines the title above it.
func (m markup) underlines(text line) bool {
	if m == adornmentMarkup {
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

// indentedCode is the indentation at which a line stops being markup and
// becomes the literal contents of an indented code block: four spaces, or one
// tab. A fence delimiter written there is TEXT, not a delimiter — and reading
// one as a delimiter let two lines of an ordinary tutorial silence every
// finding after them.
const indentedCode = 4

// isIndentedCode reports a line indented far enough to be code rather than
// markup.
func isIndentedCode(text line) bool {
	if strings.HasPrefix(string(text), "\t") {
		return true
	}
	return len(string(text))-len(strings.TrimLeft(string(text), " ")) >= indentedCode
}

// headingDiagnostics reports every heading that opens a changelog section.
func headingDiagnostics(at Path, source Source, family markup) []goyze.Diagnostic {
	var diags []goyze.Diagnostic
	state := scanner{markup: family}
	text := remaining(strings.TrimPrefix(string(source), byteOrderMark))
	current, text, ok := nextLine(text)
	for number := lineNumber(1); ok; number++ {
		upcoming, tail, hasNext := nextLine(text)
		text = tail
		var isProse bool
		state, isProse = state.step(current)
		if isProse {
			if title, found := heading(family, current, upcoming); found {
				diags = append(diags, diagnostic(at, number, headingFinding(title)))
			}
		}
		current, ok = upcoming, hasNext
	}
	return diags
}

// nextLine cuts the next line off text, normalising the carriage return a
// Windows editor leaves behind and reporting whether there was one.
//
// The document is STREAMED rather than split into a slice of lines. Splitting
// allocated a string header per line and then a second slice of the same size,
// which cost roughly thirty-six times the file: a 38 MB document of newlines
// drove 1.4 GB resident. A document is read once, in order, with one line of
// lookahead — nothing here needs them all at once.
func nextLine(text remaining) (line, remaining, bool) {
	if text == "" {
		return "", "", false
	}
	one, rest, _ := strings.Cut(string(text), "\n")
	return line(strings.TrimSuffix(one, "\r")), remaining(rest), true
}

// heading is the changelog title a line opens, if it opens one. Both written
// forms are read: the marker-run heading, and the two-line form whose title
// sits above its underline.
func heading(family markup, text, following line) (line, bool) {
	if found := atxHeading.FindStringSubmatch(string(text)); found != nil {
		return matched(line(found[1]))
	}
	if family.underlines(following) {
		return matched(text)
	}
	return "", false
}

// matched reports a title when it names a changelog, trimmed.
//
// The trim is NOT redundant with the patterns that produce the title, which was
// the previous claim here and was wrong: the marker-run pattern consumes only
// spaces and tabs, so a heading ending in a vertical tab or a non-breaking
// space kept it and stopped matching, while the same title in the two-line form
// was reported. One character silenced the rule, and the two written forms
// disagreed about the same words.
func matched(title line) (line, bool) {
	trimmed := line(strings.TrimSpace(string(title)))
	return trimmed, changelogTitle.MatchString(string(trimmed))
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
	markup      markup
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
	isCode := isIndentedCode(text)
	switch {
	case s.open.isOpen:
		s.open = s.open.after(line(trimmed), isCode)
		return s, false
	case s.isInComment:
		s.isInComment = !strings.Contains(trimmed, commentClose)
		return s, false
	case strings.HasPrefix(trimmed, commentOpen) && !strings.Contains(trimmed, commentClose):
		s.isInComment = true
		return s, false
	}
	if opened, ok := opening(line(trimmed)); ok && !isCode && s.markup.fences(opened.marker) {
		return scanner{open: opened, markup: s.markup}, false
	}
	return s, true
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
