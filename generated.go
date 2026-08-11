package docfiles

// The generated-file exemption: the one way out of this rule, and so the one
// place an evasion is worth the most. Every repair here closed a shape that
// silenced a whole document — file finding included — by claiming an authorship
// no generator had made.

import (
	"regexp"
	"strings"
)

// generatedMarkers are the phrases a generator writes to say a file is its
// output and not an author's. A document that is generated cannot be fixed by
// editing it, so reporting one tells an author to change a file that will be
// overwritten — the ratified rule is scoped to hand-maintained prose.
//
// The marker must be a GENERATOR'S claim of authorship, not merely a request
// not to edit. A bare "DO NOT EDIT" was a one-line, audit-trail-free opt-out
// from the whole rule — anyone could silence a hand-written changelog by typing
// three words at the top of it — whereas "Code generated" and "@generated" are
// the conventional statements a tool writes about its own output. A file that
// merely invites manual additions is hand-maintained BY DESIGN, and exempting
// those would have silently excused every finding this rule has in the fleet.
// The margin is a COMMENT DELIMITER one of these formats actually has, and it is
// REQUIRED. It was once `\W*` — every non-word character, which is markdown's
// list markers, its blockquote marker and its backticks — and then an optional
// group including `#` and `;`, which are not comment delimiters in any of the
// five prose formats this rule reads. Each round left the same hole: a bare
// `@generated`, or `# @generated`, is VISIBLE BODY TEXT that a reader sees, and
// typing it at the top of a hand-written changelog silenced the whole document,
// file finding included. That is the one-line, audit-trail-free opt-out this
// pattern exists to close, reintroduced twice by the pattern itself.
//
// A generator writing a MULTI-LINE header states the claim inside the comment
// rather than beside a delimiter, so [bareClaim] answers for a line the scanner
// says is already within one.
var generatedClaims = map[extension]*regexp.Regexp{
	// Markdown's only comment is the HTML one. `//` and `/* */` are C, `..` is
	// reStructuredText: in a markdown document each renders as VISIBLE BODY
	// TEXT, so accepting them let one literal line — `// @generated`, which a
	// reader sees as a paragraph — exempt a hand-written changelog whole. That
	// is the one-line, audit-trail-free opt-out this pattern exists to close,
	// reached through a delimiter the format does not have.
	markdownExt:     claimIn(commentSyntax{opener: `<!--`, closer: `-->`}),
	markdownLongExt: claimIn(commentSyntax{opener: `<!--`, closer: `-->`}),
	// reStructuredText's comment is `..` followed by text. `.. name::` is a
	// DIRECTIVE, which renders as a visible admonition rather than vanishing —
	// `.. note:: @generated` at the top of a hand-written changelog is a note a
	// reader sees, and it exempted the document. The opener therefore requires
	// whitespace after the dots, which RST does too, and refuses a directive.
	restructuredExt: claimIn(commentSyntax{opener: `\.\.[ \t]+`}),
	asciidocExt:     claimIn(commentSyntax{opener: `//+`}),
	// Plain text and the extensionless spelling have NO comment syntax, and say
	// so here rather than by absence. Every spelling of the claim is a line a
	// reader simply sees, so accepting one there accepts the same line in a
	// hand-written file — the opt-out this pattern exists to close.
	plainTextExt:     nil,
	extensionlessExt: nil,
}

// claimIn is the pattern for a claim written between one family's comment
// delimiters. The closer is optional because a single-line comment has none.
func claimIn(delimiters commentSyntax) *regexp.Regexp {
	tail := `[ \t]*$`
	if delimiters.closer != "" {
		tail = `[ \t]*(?:` + delimiters.closer + `)?[ \t]*$`
	}
	return regexp.MustCompile(`^(?:` + delimiters.opener + `)[ \t]*(?:` + claimWords + `)` + tail)
}

// commentSyntax is how one prose format opens and closes a comment. A format
// whose comment runs to the end of the line has no closer.
type commentSyntax struct{ opener, closer string }

// directiveOpeners are the lines a format writes that LOOK like a comment and
// render as visible content.
//
// reStructuredText has one: `..` introduces both a comment and a directive, and
// only the directive has a name followed by two colons. `.. note:: @generated`
// renders as an admonition a reader sees, and it exempted a hand-written
// changelog whole — the same one-line, audit-trail-free opt-out this file has
// now closed for a delimiter markdown does not have, for a fence, and for
// indentation. Go's regexp has no lookahead, so the refusal is its own pattern
// rather than a clause inside the claim.
var directiveOpeners = map[extension]*regexp.Regexp{
	restructuredExt: regexp.MustCompile(`^\.\.[ \t]+[^ \t:]+::`),
	// The rest have no such ambiguity and say so here rather than by absence:
	// markdown's only comment is the HTML one and it vanishes; AsciiDoc's `//`
	// is a line comment with no directive spelling; plain text and the
	// extensionless form have no comments at all.
	markdownExt:      nil,
	markdownLongExt:  nil,
	asciidocExt:      nil,
	plainTextExt:     nil,
	extensionlessExt: nil,
}

// rendersVisibly reports a line whose own format shows it to a reader, however
// much it resembles a comment.
func rendersVisibly(text blockLine, ext extension) bool {
	directive := directiveOpeners[ext]
	return directive != nil && directive.MatchString(string(text))
}

// bareClaim is the same words with NO delimiter, which is only a claim when the
// line is already INSIDE a comment.
var bareClaim = regexp.MustCompile(`^(?:` + claimWords + `)$`)

// generatedClaim is the pattern for one document's family. A family with no
// comment syntax at all — plain text — has no way to make a claim that a reader
// would not simply see, so it has none.
func generatedClaim(ext extension) (*regexp.Regexp, bool) {
	claim, isKnown := generatedClaims[ext]
	return claim, isKnown && claim != nil
}

// claimWords are the conventional statements a generator makes about its own
// output.
//
// The trailing period is OPTIONAL and the claim may carry the generator's own
// words after it: gomarkdoc — the standard Go markdown generator — writes
// `<!-- Code generated by gomarkdoc. DO NOT EDIT -->` with no period, and
// mirror-table writes `<!-- This file is @generated by mirror-table, do not edit
// it manually -->`. Both were reported as hand-maintained, which tells an author
// to stop editing by hand a file that is overwritten on the next run. Verified
// against the copies in this machine's module cache, not invented.
const claimWords = `.*Code generated .*DO NOT EDIT\.?.*|.*@generated.*`

// claimSite is where in the document a candidate claim sits, which decides
// whether it needs a comment delimiter of its own.
type claimSite int

const (
	// inProse is a line the document renders, where a claim must carry a
	// comment delimiter or it is text a reader sees.
	inProse claimSite = iota
	// insideComment is a line already within one, where the delimiters are on
	// the lines above and below it.
	insideComment
)

// generatedHeader is how many leading lines are searched for a claim. A
// generator writes its claim at the top, alone on a line; a document ABOUT the
// convention QUOTES it inside a sentence, and searching for the words anywhere
// in those lines exempted two real fleet documents that merely described the
// rule — a standards page and a project record, both hand-written, both wholly
// out of scope by accident.
const generatedHeader = 5

// claimSiteOf says whether a line may carry a claim at all, and where it sits. A
// line inside a fenced block is SHOWING a generated header rather than making
// one, and exempted a whole document that merely demonstrated the convention.
func claimSiteOf(state lineState) (claimSite, bool) {
	if state.isInComment {
		return insideComment, true
	}
	return inProse, state.isProse
}

// lineState is what the block scanner says about one line of the header: whether
// the document renders it, and whether it sits inside a comment.
type lineState struct{ isProse, isInComment bool }

// isGenerated reports a document that declares itself a generator's output.
//
// The header is read through the SAME block scanner the headings are, so a
// claim the document is merely SHOWING does not exempt it. A fenced block in
// the first few lines — a document demonstrating what a generated header looks
// like — otherwise silenced every finding in the file, which is the same
// one-line opt-out the claim pattern itself was tightened to close, reached
// through the one door still open.
func isGenerated(source Source, family markup, ext extension) bool {
	state := scanner{markup: family}
	text := remaining(source)
	for read := 0; read < generatedHeader; read++ {
		current, tail, ok := nextLine(text)
		if !ok {
			return false
		}
		text = tail
		was := state
		var isProse bool
		state, isProse = state.step(current)
		where, eligible := claimSiteOf(lineState{isProse: isProse, isInComment: was.isInComment})
		if eligible && declaresGenerated(current, where, ext) {
			return true
		}
	}
	return false
}

// declaresGenerated reports one line that IS a generator's claim rather than
// one that merely contains its words. The claim stands alone on its line, in
// whatever comment syntax the format uses — the pattern's own non-word margins
// consume the delimiters, so no separate stripping is needed — while a sentence
// quoting it does not match at all.
func declaresGenerated(text line, where claimSite, ext extension) bool {
	trimmed := strings.TrimSpace(string(text))
	if rendersVisibly(blockLine(trimmed), ext) {
		return false
	}
	claim, hasComments := generatedClaim(ext)
	if !hasComments {
		// `.txt` has no comment syntax, so every spelling of the claim is text a
		// reader sees. A plain-text file cannot declare itself generated without
		// saying so out loud, and a rule that accepted it would accept the same
		// line in a hand-written one.
		return false
	}
	if where == insideComment {
		// Already inside a comment, so no delimiter is expected on this line —
		// the conventional multi-line generated header puts the claim on a line
		// of its own between `<!--` and `-->`, and requiring a delimiter there
		// reported a genuinely generated document as hand-maintained.
		return bareClaim.MatchString(trimmed)
	}
	return claim.MatchString(trimmed)
}
