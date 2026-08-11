package docfiles_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAnExampleInsideABlockIsNotASection pins the exemption without which the
// rule could not describe itself: the standards show banned shapes inside
// fences and comments, and reporting those makes the documentation of a rule a
// violation of it.
func TestAnExampleInsideABlockIsNotASection(t *testing.T) {
	t.Parallel()

	for name, source := range map[string]string{
		"backtick fence": "```\n## Changelog\n```\n",
		"tilde fence":    "~~~\n## Changelog\n~~~\n",
		"info string":    "```markdown\n## Changelog\n```\n",
		"nested fence":   "````markdown\n```text\n## Changelog\n```\n````\n",
		"html comment":   "<!--\n## Changelog\n-->\n",
		"indented fence": "  ```\n  ## Changelog\n  ```\n",
	} {
		assert.Empty(t, analyze(t, "README.md", source), "%s holds an example", name)
	}
}

// TestAFenceClosesSoLaterSectionsAreStillRead pins the property that makes the
// exemption safe rather than a switch. A fence that never closed would silence
// the whole rest of the document — one ``` line disabling every finding after
// it — and a real fleet document with nested fences did exactly that.
func TestAFenceClosesSoLaterSectionsAreStillRead(t *testing.T) {
	t.Parallel()

	for name, source := range map[string]string{
		"after a plain fence":  "```\ncode\n```\n\n## Changelog\n",
		"after a tilde fence":  "~~~\ncode\n~~~\n\n## Changelog\n",
		"after a nested fence": "````\n```\ncode\n```\n````\n\n## Changelog\n",
		"after a comment":      "<!-- note -->\n\n## Changelog\n",
		"after a long comment": "<!--\nnote\n-->\n\n## Changelog\n",
	} {
		diags := analyze(t, "README.md", source)
		require.Len(t, diags, 1, "%s: the block closed, so the section is read", name)
	}
}

// TestAShorterRunNeverClosesALongerFence pins the CommonMark rule that makes
// nesting work: a ```` block is not closed by the ``` fences it wraps, and an
// info string closes nothing at all.
func TestAShorterRunNeverClosesALongerFence(t *testing.T) {
	t.Parallel()

	assert.Empty(t, analyze(t, "README.md", "````\n```\n## Changelog\n```\n````\n"))
	assert.Empty(t, analyze(t, "README.md", "```\n```go\n## Changelog\n```\n"))
	assert.Empty(t, analyze(t, "README.md", "```\n~~~\n## Changelog\n~~~\n```\n"),
		"a fence closes only on its own marker")
}

// TestALongerRunClosesAShorterFence pins the direction CommonMark allows and
// nothing tested: a closing run may be LONGER than the one that opened the
// block. Only the shorter-run direction was covered, so a rule that required
// exact equality would have left the rest of such a document exempt.
func TestALongerRunClosesAShorterFence(t *testing.T) {
	t.Parallel()

	diags := analyze(t, "README.md", "```\ncode\n``````\n\n## Changelog\n")

	assert.Len(t, diags, 1, "the longer run closed the block, so the later section is read")
}

// TestAnUnclosedFenceSilencesOnlyWhatFollowsIt pins the deliberate reading of a
// malformed document: an opened fence that never closes takes the rest of the
// file with it, so the sections BEFORE it must still be reported.
func TestAnUnclosedFenceSilencesOnlyWhatFollowsIt(t *testing.T) {
	t.Parallel()

	diags := analyze(t, "README.md", "## Changelog\n\n```\n## Recent Changes\n")

	require.Len(t, diags, 1)
	assert.Equal(t, 1, diags[0].Line)
}

// TestAnIndentedFenceIsCodeNotADelimiter pins the repair for a two-line
// evasion. Four spaces (or a tab) makes a line the literal CONTENT of an
// indented code block, so a ``` written there is text a tutorial is showing the
// reader — and reading it as a delimiter opened a fence that never closed,
// silencing every heading in the rest of the document.
func TestAnIndentedFenceIsCodeNotADelimiter(t *testing.T) {
	t.Parallel()

	for name, source := range map[string]string{
		"four spaces": "A fence opens with:\n\n    ```\n\nand closes with three more.\n\n## Changelog\n",
		"tab":         "A fence opens with:\n\n\t```go\n\nand closes with three more.\n\n## Changelog\n",
		"six spaces":  "Shown:\n\n      ~~~\n\n## Changelog\n",
	} {
		diags := analyze(t, "README.md", source)
		assert.Len(t, diags, 1, "%s: an indented delimiter is code, so the later section is still read", name)
	}

	// THREE spaces, which is the boundary named: a fence may be indented up to
	// three and is code at four. This assertion used to be written with two,
	// so the constant it exists to pin could be moved to three and nothing
	// failed — it verified an allowance nobody disputed instead of the edge.
	assert.Empty(t, analyze(t, "README.md", "   ```\n## Changelog\n   ```\n"),
		"three spaces is still a fence, which is where CommonMark draws the line")
	assert.Len(t, analyze(t, "README.md", "    ```\n## Changelog\n    ```\n"), 1,
		"four spaces is an indented code block, so the fence is text and the heading is a section")
}

// TestAnIndentedDelimiterNeverClosesAnOpenFence pins the same boundary on the
// closing side: a delimiter indented into code territory does not end the block
// it appears in.
func TestAnIndentedDelimiterNeverClosesAnOpenFence(t *testing.T) {
	t.Parallel()

	assert.Empty(t, analyze(t, "README.md", "```\n    ```\n## Changelog\n"),
		"the block is still open, so the heading inside it is an example")
}

// TestAnIndentedCommentOpenerIsCodeNotAComment pins the hole that survived the
// round-2 repair verbatim. The indentation rule was added for fences and never
// applied to comments, so two lines of an ordinary tutorial — a four-space
// indented `<!--` — silenced every finding after them, and no closing marker
// was ever needed because the state never cleared.
func TestAnIndentedCommentOpenerIsCodeNotAComment(t *testing.T) {
	t.Parallel()

	source := "How to comment things out:\n\n    <!--\n    still code\n\n# Changelog\n\n- 1.0\n"

	assert.Len(t, analyze(t, "README.md", source), 1, "the indented opener is code, so the section is read")
}

// TestACommentOpenedMidLineStillComments pins the other direction: `text <!--`
// comments out everything after it, and requiring the opener at the start of
// the line reported a heading the document had commented away.
func TestACommentOpenedMidLineStillComments(t *testing.T) {
	t.Parallel()

	assert.Empty(t, analyze(t, "README.md", "Intro text <!--\n## Changelog\n-->\nEnd.\n"))
	assert.Len(t, analyze(t, "README.md", "Intro <!-- aside --> text\n\n## Changelog\n"), 1,
		"a comment that closes on its own line leaves the document readable")
}

// TestACommentClosedInsideACodeSpanOnItsOpeningLine pins the contradiction that
// two separate questions produced. Asking "is there an opener outside a code
// span?" and "is there a closer?" on DIFFERENT texts made a comment opened and
// closed on one line stay open forever — while the identical span on a
// CONTINUATION line closed it, because that half reads the raw line. Three
// characters, and every finding below them vanished.
func TestACommentClosedInsideACodeSpanOnItsOpeningLine(t *testing.T) {
	t.Parallel()

	assert.Len(t, analyze(t, "README.md", "# Guide\n\n<!-- see `-->`\n\n## Changelog\n"), 1,
		"the comment ends at the first raw closer, backticks or not")
	assert.Len(t, analyze(t, "README.md", "<!-- a -->\n\n## Changelog\n"), 1,
		"and an ordinary one-line comment closes too")
	assert.Empty(t, analyze(t, "README.md", "<!-- a --> tail <!--\n## Changelog\n"),
		"while a line that closes one and opens another stays open")
	assert.Len(t, analyze(t, "README.md", "Shown: `<!--` opener\n\n## Changelog\n"), 1,
		"a backticked OPENER is still only shown")
}

// TestAFenceInfoStringIsNotACommentOpener pins the order the two blocks are
// tested in. A fenced block may carry an info string and `<!--` is a legal one —
// read as a comment opener instead, it commented away every line after it and
// nothing ever closed it.
func TestAFenceInfoStringIsNotACommentOpener(t *testing.T) {
	t.Parallel()

	assert.Len(t, analyze(t, "README.md", "# Guide\n\n```<!--\ncode\n```\n\n## Changelog\n"), 1,
		"the fence opens and closes, so the section after it is read")
	assert.Empty(t, analyze(t, "README.md", "```<!--\n## Changelog\n```\n"),
		"and the heading inside it is still an example")
}

// TestAsciidocFencesTheWayAsciidoctorDoes pins the other half of the Markdown
// compatibility this family already accepts for headings. Taking `## Title` as a
// heading in AsciiDoc while refusing its fenced block reported an AsciiDoc
// document's own EXAMPLE as a section of it.
func TestAsciidocFencesTheWayAsciidoctorDoes(t *testing.T) {
	t.Parallel()

	assert.Empty(t, analyze(t, "guide.adoc", "= Guide\n\n```\n== Changelog\n```\n"),
		"a backtick fence holds an example")
	assert.Len(t, analyze(t, "guide.adoc", "= Guide\n\n```\ncode\n```\n\n== Changelog\n"), 1,
		"and it closes, so a real section after it is read")
	assert.Len(t, analyze(t, "notes.rst", "Guide\n=====\n\n```\nChangelog\n=========\n"), 1,
		"reStructuredText has no fence at all, so its adornment still underlines")
}

// TestAClosingRunMustBeExactlyAsLongAsItsOpener pins the CommonMark rule that
// makes code-span stripping safe. A longer run does not close a shorter span,
// and reading one as a closer would swallow text that is not code.
func TestAClosingRunMustBeExactlyAsLongAsItsOpener(t *testing.T) {
	t.Parallel()

	assert.Len(t, analyze(t, "README.md", "Shown: ``<!--`` opener\n\n## Changelog\n"), 1,
		"a two-backtick run closes on two, so the opener inside it is shown")
	assert.Empty(t, analyze(t, "README.md", "Shown: ``<!--` opener\n\n## Changelog\n"),
		"a single backtick does not close a double run, so those backticks are literal "+
			"text and the opener between them is real")
}

// TestAnIndentedRunOfDelimiterCharactersOpensNoBlock pins the column-zero rule
// on the OPENING half. AsciiDoc and RST require a delimiter at column zero; an
// indented run is literal content, and reading one as a delimiter opens a
// phantom block that swallows the real section beneath it — a silent pass
// produced by four spaces. The closing half of the same pair was pinned; this
// half had nothing behind it.
func TestAnIndentedRunOfDelimiterCharactersOpensNoBlock(t *testing.T) {
	t.Parallel()

	assert.Len(t, analyze(t, "doc.adoc", "    ----\n== Changelog\n\nrecent things\n"), 1,
		"the indented run is content, so the section below it is a section")
	assert.Empty(t, analyze(t, "doc.adoc", "----\n== Changelog\n----\n"),
		"while a delimiter at column zero really does hold an example")
}

// TestATabIndentedLineIsLiteralContent pins the tab half of the indent test. RST
// makes a tab-indented line a literal block, so the text in it is quoted rather
// than written — and reading it as a title invents a section out of an example
// somebody was showing.
func TestATabIndentedLineIsLiteralContent(t *testing.T) {
	t.Parallel()

	assert.Empty(t, analyze(t, "doc.rst", "para\n\n\tChangelog\n=========\n"),
		"a tab-indented line is quoted content, not a title")
	assert.Len(t, analyze(t, "doc.rst", "para\n\nChangelog\n=========\n"), 1,
		"while the same text at column zero is one")
}
