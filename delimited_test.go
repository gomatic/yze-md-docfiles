package docfiles_test

// The block forms that are NOT markdown's: AsciiDoc's delimited blocks and
// captions, and reStructuredText's transitions and literal blocks. They were
// read as one family with markdown for as long as the three formats were
// believed to disagree only about underlines.

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestADelimitedBlockIsAnExampleNotASection pins the AsciiDoc and RST block
// forms. The same run of dashes is a section adornment under a title and a
// listing delimiter after a blank line, so a document SHOWING a banned heading
// inside a listing block had its example reported as a section — the failure
// the scanner exists to prevent, in the two formats it had no block model for.
func TestADelimitedBlockIsAnExampleNotASection(t *testing.T) {
	t.Parallel()

	adoc := "= My Doc\n\nHere is how NOT to write a doc:\n\n[source,asciidoc]\n----\n== Changelog\n\n* 1.0\n----\n\nDone.\n"
	assert.Empty(t, analyze(t, "guide.adoc", adoc), "the listing block holds an example")

	rst := "Guide\n=====\n\nExample of a banned section::\n\n.. code-block:: rst\n\n  Changelog\n  ---------\n\n  * 1.0\n"
	assert.Empty(t, analyze(t, "guide.rst", rst), "an indented literal block is quoted, not written")
}

// TestADelimitedBlockClosesSoLaterSectionsAreRead pins that the block model is
// a model and not a switch: a real section after the example is still reported.
func TestADelimitedBlockClosesSoLaterSectionsAreRead(t *testing.T) {
	t.Parallel()

	adoc := "= My Doc\n\n----\n== Changelog\n----\n\nChangelog\n=========\n\n* 1.0\n"

	assert.Len(t, analyze(t, "guide.adoc", adoc), 1, "the block closed, so the real section is read")
}

// TestAFenceNeedsThreeMarkers pins the lower bound. With fewer, an ordinary
// inline code span at the start of a line would open a block and silence the
// rest of the document.
func TestAFenceNeedsThreeMarkers(t *testing.T) {
	t.Parallel()

	assert.Len(t, analyze(t, "README.md", "`code`\n\n## Changelog\n"), 1, "one backtick opens nothing")
	assert.Len(t, analyze(t, "README.md", "``code``\n\n## Changelog\n"), 1, "two open nothing either")
	assert.Empty(t, analyze(t, "README.md", "```\n## Changelog\n"), "three do")
}

// TestAnIndentedDelimiterNeitherOpensNorClosesADelimitedBlock pins the
// indentation rule on BOTH halves of the delimited pair. The fence pair already
// had it on both; the delimited pair had it on neither, so an indented run
// inside a listing block closed it early and reported the example the document
// was showing, while an RST literal block opened a phantom one that silenced
// the rest of the file.
func TestAnIndentedDelimiterNeitherOpensNorClosesADelimitedBlock(t *testing.T) {
	t.Parallel()

	closedEarly := "= Doc\n\nExample:\n\n----\n        ----\n== Changelog\n----\n\nEnd.\n"
	assert.Empty(t, analyze(t, "c.adoc", closedEarly), "an indented run does not close the block")

	literal := "Guide\n=====\n\nA divider looks like this::\n\n    ----\n\nUnreleased\n~~~~~~~~~~\n\nstuff\n"
	assert.Len(t, analyze(t, "j.rst", literal), 1, "and an indented run inside a literal block opens nothing")
}

// TestAnHtmlCommentIsAMarkdownConstruct pins the markup split on the comment
// model, which had none. Four characters anywhere in an RST or AsciiDoc
// document — formats with no HTML comments at all — silenced every finding
// after them, needing no closing marker and leaving no trace.
func TestAnHtmlCommentIsAMarkdownConstruct(t *testing.T) {
	t.Parallel()

	source := "Guide\n=====\n\nWritten <!-- like this --> and unterminated <!--\n\nUnreleased\n~~~~~~~~~~\n\n* 1.0\n"

	assert.Len(t, analyze(t, "p.rst", source), 1, "reStructuredText has no HTML comments")
	assert.Len(t, analyze(t, "p.adoc", source), 1, "and neither does AsciiDoc")
}

// TestACommentThatClosesAndReopensStaysOpen pins the closing half of the
// comment model against the shape its opening half already handles: a line may
// close one comment and open another, so finding a closer is not the end of the
// question.
func TestACommentThatClosesAndReopensStaysOpen(t *testing.T) {
	t.Parallel()

	assert.Empty(t, analyze(t, "l.md", "Doc\n\n<!--\n## Changelog\n--> visible <!--\n## Changelog\n-->\n\nEnd\n"))
	assert.Len(t, analyze(t, "l.md", "Doc\n\n<!--\nhidden\n--> visible\n\n## Changelog\n"), 1,
		"and a comment that merely closes leaves the document readable")
}

// TestAnAsciiDocBlockCommentIsAnExample pins `////`, which is AsciiDoc's block
// comment: a heading a document has deliberately commented out is not a section
// of it.
func TestAnAsciiDocBlockCommentIsAnExample(t *testing.T) {
	t.Parallel()

	assert.Empty(t, analyze(t, "t.adoc", "= Guide\n\n////\nNote to self:\n== Changelog\nremove later\n////\n\nDone.\n"))
	assert.Len(t, analyze(t, "t.adoc", "= Guide\n\n////\nnote\n////\n\n== Changelog\n"), 1,
		"and the block closes, so a real section after it is read")
}

// TestAnAsciidocBlockTitleDoesNotStopItsBlockFromOpening pins the AsciiDoc
// caption. A block title sits immediately above its delimiter, with no blank
// line permitted, which is exactly where an underlined heading's title sits —
// so the delimiter was read as an underline and the block never opened, and
// every example the block was showing was read as prose.
func TestAnAsciidocBlockTitleDoesNotStopItsBlockFromOpening(t *testing.T) {
	t.Parallel()

	for name, delimiter := range map[string]string{
		"listing": "----", "example": "====", "literal": "....", "sidebar": "****",
	} {
		source := ".A caption\n" + delimiter + "\n== Changelog\n" + delimiter + "\n"
		assert.Empty(t, analyze(t, "guide.adoc", source),
			"%s: a captioned block still shows its contents rather than opening a section", name)
	}
	assert.Len(t, analyze(t, "guide.adoc", "A title\n----\n== Changelog\n"), 1,
		"a real title above a delimiter still underlines it, so what follows is prose")
}

// TestAReStructuredTextTransitionDoesNotOpenABlock pins the format's own
// grammar against AsciiDoc's. reStructuredText has no delimited block: a
// standalone run of punctuation is a transition, or the overline of a title.
// Reading one as a block that never closed made every heading below it
// invisible, which is a rule silencing itself for the rest of a file.
func TestAReStructuredTextTransitionDoesNotOpenABlock(t *testing.T) {
	t.Parallel()

	assert.Len(t, analyze(t, "notes.rst", "Intro\n\n----\n\nChangelog\n=========\n"), 1,
		"a transition is not a block opener")
	assert.Len(t, analyze(t, "notes.rst", "=========\nChangelog\n=========\n"), 1,
		"an overlined title is the commonest RST heading and is still a heading")
	assert.Len(
		t,
		analyze(t, "notes.rst", "Intro\n\n.. code-block:: text\n\n   ## Changelog\n\nChangelog\n---------\n"),
		1,
		"an indented literal block is still an example, and the section after it is still read",
	)
}

// TestACommentOpenerInsideACodeSpanIsShownNotUsed pins the difference between
// markdown SHOWING `<!--` and using it. A backticked opener silenced every
// finding below it — a one-line, unlogged opt-out — and the document most
// likely to write one is the documentation of this rule.
func TestACommentOpenerInsideACodeSpanIsShownNotUsed(t *testing.T) {
	t.Parallel()

	assert.Len(t, analyze(t, "README.md", "Intro mentions the `<!--` opener.\n\n## Changelog\n"), 1,
		"a backticked opener does not open a comment")
	assert.Len(t, analyze(t, "README.md", "Double ``<!--`` opener.\n\n## Changelog\n"), 1,
		"a two-backtick span closes on two backticks")
	assert.Empty(t, analyze(t, "README.md", "Intro <!-- hidden\n\n## Changelog\n"),
		"a bare opener still comments out everything after it")
	assert.Empty(t, analyze(t, "README.md", "Unclosed ` span with <!-- opener\n\n## Changelog\n"),
		"an unclosed backtick run is literal text, so the opener after it is real")
}

// TestOnlyTheLastCommentOpenerOnALineDecides pins which opener is asked about.
// A line may open and close a comment and then open another; reading the FIRST
// opener finds a closer after it and calls the line closed, so a heading the
// author commented away was reported as a section.
func TestOnlyTheLastCommentOpenerOnALineDecides(t *testing.T) {
	t.Parallel()

	assert.Empty(t, analyze(t, "README.md", "Intro <!-- note --> tail <!--\n## Changelog\n"),
		"the trailing opener leaves the comment open")
}

// TestATitleCandidateIsNotItselfADelimiter pins the guard that keeps a
// delimiter from being read as the title of the delimiter below it. Two
// delimiters in a row open a block, they do not underline each other.
func TestATitleCandidateIsNotItselfADelimiter(t *testing.T) {
	t.Parallel()

	assert.Empty(t, analyze(t, "guide.adoc", "Title\n=====\n----\n== Changelog\n----\n"),
		"the second delimiter opens a block rather than underlining the first")
}

// TestEveryDelimiterOpensABlock pins AsciiDoc's whole delimiter vocabulary, for
// the same reason and in the same direction. A delimiter this rule does not know
// fails to suppress the example inside it, so a document SHOWING a changelog
// heading is reported as having one.
func TestEveryDelimiterOpensABlock(t *testing.T) {
	t.Parallel()

	for _, marker := range []string{"-", "=", ".", "*", "_", "+", "~", "/"} {
		fence := repeated(marker, 4)
		shown := fmt.Sprintf(".An example\n%s\n== Changelog\n%s\n", fence, fence)

		assert.Empty(t, analyze(t, "guide.adoc", shown),
			"a %q block holds an example, not a section", marker)
	}
}
