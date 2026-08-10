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
// The margins are the COMMENT DELIMITERS these formats actually use, named one
// by one. They were once `\W*`, chosen to consume a delimiter of any shape —
// and `\W` is every non-word character, which is also markdown's list markers,
// its blockquote marker and its backticks. `- @generated` in a list of the
// conventional markers, `> @generated` in a quotation, or the claim inside a
// code span exempted the whole document, file finding included: the same
// one-line, audit-trail-free opt-out this pattern was tightened to close, and a
// document explaining the convention still silenced itself.
var generatedClaim = regexp.MustCompile(
	`^(?:<!--|//+|#+|/\*|;+|\.\.)?[ \t]*(?:Code generated .*DO NOT EDIT\.|@generated)[ \t]*(?:-->|\*/)?$`,
)

// generatedHeader is how many leading lines are searched for a claim. A
// generator writes its claim at the top, alone on a line; a document ABOUT the
// convention QUOTES it inside a sentence, and searching for the words anywhere
// in those lines exempted two real fleet documents that merely described the
// rule — a standards page and a project record, both hand-written, both wholly
// out of scope by accident.
const generatedHeader = 5

// isGenerated reports a document that declares itself a generator's output.
//
// The header is read through the SAME block scanner the headings are, so a
// claim the document is merely SHOWING does not exempt it. A fenced block in
// the first few lines — a document demonstrating what a generated header looks
// like — otherwise silenced every finding in the file, which is the same
// one-line opt-out the claim pattern itself was tightened to close, reached
// through the one door still open.
func isGenerated(source Source, family markup) bool {
	state := scanner{markup: family}
	text := remaining(source)
	for read := 0; read < generatedHeader; read++ {
		current, tail, ok := nextLine(text)
		if !ok {
			return false
		}
		text = tail
		var isProse bool
		if state, isProse = state.step(current); isProse && declaresGenerated(current) {
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
func declaresGenerated(text line) bool {
	return generatedClaim.MatchString(strings.TrimSpace(string(text)))
}
