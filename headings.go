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
// underscored variants, and the spaced ones: `Release Notes.md` is the spelling
// a person types, and the heading half already admitted the space while this
// half did not — so the doc's own example of an unambiguous FILE name was the
// one shape neither half caught. The name must be the WHOLE stem —
// `changelog-policy.md` is a document ABOUT the policy, which is worth keeping.
var changelogFileName = regexp.MustCompile(`^(change[-_ ]?log|changes|release[-_ ]?notes)$`)

// changelogTitle is a heading that opens a changelog section. The alternatives
// are spelled out rather than matched loosely, because "Change Process" is not
// a changelog and a rule that guessed would report it — and the plural
// "Changelogs" is deliberately absent, because the fleet's one instance is the
// documentation standard describing this very rule.
var changelogTitle = regexp.MustCompile(
	`^(?i)(change[-_ ]?log|recent changes|version history|release history|revision history|what'?s new|unreleased)$`,
)

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

// indentedCode is the indentation at which a line stops being markup and
// becomes the literal contents of an indented code block: four spaces, or one
// tab. A fence delimiter written there is TEXT, not a delimiter — and reading
// one as a delimiter let two lines of an ordinary tutorial silence every
// finding after them.
const indentedCode = 4

// isIndented reports any leading whitespace at all.
func isIndented(text line) bool {
	return strings.TrimLeft(string(text), " \t") != string(text)
}

// isIndentedCode reports a line indented far enough to be code rather than
// markup.
func isIndentedCode(text line) bool {
	if strings.HasPrefix(string(text), "\t") {
		return true
	}
	return len(string(text))-len(strings.TrimLeft(string(text), " ")) >= indentedCode
}

// headingDiagnostics reports the headings that open a changelog section, and
// how many there were.
//
// It stops COLLECTING at the limit rather than collecting everything and
// truncating afterwards. Truncating afterwards bounded the report and not the
// work: eight megabytes of legal prose, every line a banned heading, still
// built every diagnostic — 524 MB resident — before discarding all but a
// thousand of them, which is the cost the limit exists to avoid.
func headingDiagnostics(at Path, source Source, family markup) ([]goyze.Diagnostic, findingCount) {
	var diags []goyze.Diagnostic
	total := findingCount(0)
	state := scanner{markup: family}
	// The byte order mark is already gone: [Diagnostics] strips it once, for
	// every reader, because two readers stripping it separately is how one of
	// them came to forget.
	text := remaining(source)
	current, text, ok := nextLine(text)
	for number := lineNumber(1); ok; number++ {
		upcoming, tail, hasNext := nextLine(text)
		text = tail
		var found bool
		state, diags, found = state.read(at, family, current, upcoming, number, diags)
		if found {
			total++
		}
		current, ok = upcoming, hasNext
	}
	return diags, total
}

// read advances the scanner over one line and collects the finding it opens, if
// any. It stops APPENDING at the limit while still reporting that it found one,
// so the count stays true without the cost: collecting everything and
// truncating afterwards bounded the report and not the work.
func (s scanner) read(
	at Path,
	family markup,
	current, upcoming line,
	number lineNumber,
	diags []goyze.Diagnostic,
) (scanner, []goyze.Diagnostic, bool) {
	next, isProse := s.step(current)
	if !isProse {
		return next, diags, false
	}
	title, found := heading(family, current, upcoming)
	if !found {
		return next, diags, false
	}
	if findingCount(len(diags)) < findingLimit {
		diags = append(diags, diagnostic(at, number, headingFinding(title)))
	}
	return next, diags, true
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
//
// The marker-run form is tried FIRST. `# Changelog` followed by a run of equals
// signs is both — an ATX heading, and a line with an underline beneath it — and
// reading it as the second yields the title `# Changelog`, which matches no
// vocabulary and reports nothing.
func heading(family markup, text, following line) (line, bool) {
	if pattern, ok := family.heading(); ok {
		if found := pattern.FindStringSubmatch(string(text)); found != nil {
			return matched(line(found[1]))
		}
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
	return trimmed, changelogTitle.MatchString(string(unbracketed(trimmed)))
}

// unbracketed is a title with one surrounding pair of square brackets removed.
//
// `## [Unreleased]` is the Keep a Changelog spelling — the single most common
// way the banned section is actually written — and the vocabulary admitted
// `Unreleased` while the brackets it always arrives in made it no match. The
// REPORTED title keeps its brackets, because that is what the author will search
// their document for.
func unbracketed(title line) line {
	if len(title) < 2 || title[0] != '[' || title[len(title)-1] != ']' {
		return title
	}
	return title[1 : len(title)-1]
}

// headingFinding is the message for one banned section. The title is quoted
// with strconv, so a heading holding a quote, a tab or a backslash is rendered
// unambiguously rather than pasted into the sentence to be misread.
func headingFinding(title line) finding {
	return finding(fmt.Sprintf(headingMessage, string(title)))
}
