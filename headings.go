package docfiles

// The changelog SECTION: a heading, as the parser says it is one, whose title
// names a changelog. What is and is not a heading belongs to the parse; what a
// changelog is called belongs here.

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	goyze "github.com/gomatic/go-yze"
	"github.com/yuin/goldmark/ast"
)

// line is one physical line of a document, without its terminator.
type line string

// lineNumber is a 1-based position in a document, as a diagnostic reports it.
type lineNumber int

// byteOrderMark is the UTF-8 BOM some editors write. It precedes the first
// character while being invisible in the text, so a document opening with one
// has its first line read as ordinary paragraph text — which silently exempts
// whatever a generator wrote there.
const byteOrderMark = "\ufeff"

// changelogTitle is a heading that opens a changelog section. The alternatives
// are spelled out rather than matched loosely, because "Change Process" is not
// a changelog and a rule that guessed would report it — and the plural
// "Changelogs" is deliberately absent, because the fleet's one instance is the
// documentation standard describing this very rule.
var changelogTitle = regexp.MustCompile(
	`^(?i)(change[-_ ]?log|recent changes|version history|release history|revision history|what'?s new|unreleased)$`,
)

// headingDiagnostics reports the headings that open a changelog section, and how
// many there were.
//
// The count is the TRUE one and the slice is not: collecting stops at the
// per-document limit while counting continues, so a document holding ten
// thousand banned headings costs a thousand diagnostics rather than ten
// thousand, and still names how many it really held.
func headingDiagnostics(at Path, doc document) ([]goyze.Diagnostic, findingCount) {
	found := sections{}.walk(at, doc, doc.root)
	return found.diags, found.total
}

// sections is what a walk has found so far: the diagnostics it kept, and how
// many banned headings it saw.
type sections struct {
	diags []goyze.Diagnostic
	total findingCount
}

// walk descends the block tree, collecting the headings that open a changelog
// section, in source order.
//
// The accumulator is threaded through the recursion rather than each level
// returning its own slice, because appending a subtree's result at every level
// re-copies it once per level of nesting — quadratic for a document that nests
// linearly, which is a cost a checked-in file controls.
func (s sections) walk(at Path, doc document, node ast.Node) sections {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		s = s.opened(at, doc, child).walk(at, doc, child)
	}
	return s
}

// opened is s with this node's finding added, when the node is a heading that
// opens a changelog section.
func (s sections) opened(at Path, doc document, node ast.Node) sections {
	heading, isHeading := node.(*ast.Heading)
	if !isHeading {
		return s
	}
	title, opens := doc.changelogSection(heading)
	if !opens {
		return s
	}
	s.total++
	if findingCount(len(s.diags)) < findingLimit {
		s.diags = append(s.diags, diagnostic(at, doc.lineOf(heading.Lines().At(0)), headingFinding(title)))
	}
	return s
}

// changelogSection is the title a heading opens, and whether that title names a
// changelog.
//
// The title is read from the SOURCE rather than assembled from the parsed
// inline text, because the message must name what an author can search their own
// document for: `## [Unreleased]` is reported with its brackets, which is what
// they typed.
//
// A heading whose text does not stand on a SINGLE line is not read at all, and
// that covers two shapes with one condition. A heading with no text —
// `##` alone — has no title to name. A heading whose title is WRAPPED across
// lines is a heading no spelling in this vocabulary is written in: Keep a
// Changelog, release-please, git-cliff and every hand-typed `## Changelog` put
// it on one line, and a wrapped one is already a finding of the `yze/hardwrap`
// rule in this same suite. Reading it here would either quote a title spanning
// lines the finding does not point at, or quote its first line alone — and
// `Changelog` above `of Doom` is not a changelog section.
func (d document) changelogSection(heading *ast.Heading) (line, bool) {
	if heading.Lines().Len() != 1 {
		return "", false
	}
	return matched(line(d.textOf(heading.Lines().At(0))))
}

// matched reports a title when it names a changelog, trimmed.
//
// The trim is the rule's own, not the parser's: goldmark strips the spaces and
// tabs CommonMark says a heading's text ends at, and leaves everything else — so
// a heading ending in a vertical tab or a non-breaking space keeps it, and the
// vocabulary saw a title nobody typed.
//
// The REPORTED title is the trimmed source text, because that is what an author
// can search their own document for; only the form the vocabulary is matched
// against is normalised further.
func matched(title line) (line, bool) {
	trimmed := line(strings.TrimSpace(string(title)))
	return trimmed, changelogTitle.MatchString(string(unbracketed(unlinked(visible(trimmed)))))
}

// visible is a title with the characters a reader cannot see removed, trimmed
// again afterwards because removing them can leave ordinary space behind.
//
// Unicode's FORMAT characters render as nothing at all, so a heading carrying
// one is a heading whose words are exactly what a reader sees — and a single
// invisible character was a one-character opt-out from the rule: `## Changelog`
// with a zero width space after it, a soft hyphen inside it, or a byte order
// mark in front of it (which is stripped only at offset zero, where a
// document's own BOM lives) all rendered as `Changelog` and matched nothing.
// That is the same defect as the vertical tab this function was already
// repaired for, in characters nobody can see at all. Found by an adversarial
// review.
//
// Combining marks are deliberately NOT removed. A variation selector is
// invisible too, but it sits in the same Unicode category as the accents real
// titles are written with, and stripping those would rewrite a title into
// something its author never typed.
func visible(title line) line {
	return line(strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, string(title))))
}

// unlinked is a title with a surrounding markdown link's target removed:
// `[Unreleased](https://…/compare)` is the OTHER spelling Keep a Changelog
// writes, and the vocabulary saw a title that was mostly a URL.
func unlinked(title line) line {
	for _, form := range []linkForm{{opener: "](", closer: ")"}, {opener: "][", closer: "]"}} {
		if label, isLink := linkLabel(title, form); isLink {
			return label
		}
	}
	return title
}

// linkLabel is the label of a title that is ENTIRELY one link, written with the
// given target delimiters, reporting whether it is one.
//
// Both markdown link forms are read. `[Unreleased](…/compare)` and
// `[Unreleased][unreleased]` render identically, and Keep a Changelog's own
// template has used each — so admitting one and not the other left the same
// heading reported or silent depending on which spelling its author copied.
func linkLabel(text line, form linkForm) (line, bool) {
	if !strings.HasSuffix(string(text), form.closer) {
		return "", false
	}
	label, target, found := strings.Cut(string(text), form.opener)
	if !found || !strings.HasPrefix(label, "[") {
		return "", false
	}
	if strings.Contains(target[:len(target)-len(form.closer)], form.opener) {
		// Two links in one heading is a sentence, not a title.
		return "", false
	}
	return line(label + "]"), true
}

// linkForm is one of markdown's two ways of writing a link target: `](url)` for
// an inline one, `][label]` for a reference.
type linkForm struct{ opener, closer string }

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
