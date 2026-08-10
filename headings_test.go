package docfiles_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEveryBannedSectionSpellingIsReported pins the heading vocabulary. Each
// name was measured against the fleet before being admitted, and each is a
// changelog under another title.
func TestEveryBannedSectionSpellingIsReported(t *testing.T) {
	t.Parallel()

	for _, heading := range []string{
		"## Changelog", "# CHANGELOG", "###### change log", "## Change-Log",
		"## Recent Changes", "## Version History", "## Release History",
		"## Revision History", "## What's New", "## Whats New", "## Unreleased",
	} {
		assert.Len(t, analyze(t, "README.md", heading+"\n"), 1, "%q opens a changelog", heading)
	}
}

// TestANeighbouringHeadingIsSilent pins the names deliberately REFUSED on the
// measurement. A bare `Changes` or `History` heading is ordinary prose in a
// design document; `Release Notes` as a heading has three fleet instances and
// all three are legitimate; and `Changelogs` plural is the documentation
// standard describing this very rule.
func TestANeighbouringHeadingIsSilent(t *testing.T) {
	t.Parallel()

	for _, heading := range []string{
		"## Changes", "## History", "## Release Notes", "## Changelogs",
		"## Change Process", "## Breaking Changes", "## Changelog Policy",
		"##Changelog", "#### Changelog of Doom",
	} {
		assert.Empty(t, analyze(t, "README.md", heading+"\n"), "%q is not a changelog section", heading)
	}
}

// TestAHeadingIsReadHoweverCommonMarkAllowsItToBeWritten pins the spellings a
// heading may legally take. Each was an evasion: a closing hash run, up to
// three leading spaces, a tab separator, a byte-order mark, and Windows line
// endings all produce a heading a reader sees and the rule did not.
func TestAHeadingIsReadHoweverCommonMarkAllowsItToBeWritten(t *testing.T) {
	t.Parallel()

	for name, source := range map[string]string{
		"closed ATX":      "## Changelog ##\n",
		"one space":       " ## Changelog\n",
		"three spaces":    "   ## Changelog\n",
		"tab separator":   "##\tChangelog\n",
		"byte-order mark": "\ufeff## Changelog\n",
		"carriage return": "## Changelog\r\n",
		"trailing space":  "## Changelog   \n",
		"setext":          "Changelog\n=========\n",
		"setext dashes":   "Recent Changes\n--------------\n",
		"asciidoc":        "== Changelog\n",
	} {
		assert.Len(t, analyze(t, "README.md", source), 1, "%s is a heading", name)
	}
}

// TestAFourSpaceIndentIsCodeNotAHeading pins the boundary of that allowance:
// four spaces opens an indented code block in CommonMark, so the line is an
// example rather than a section.
func TestAFourSpaceIndentIsCodeNotAHeading(t *testing.T) {
	t.Parallel()

	assert.Empty(t, analyze(t, "README.md", "    ## Changelog\n"))
}

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

// TestAnUnclosedFenceSilencesOnlyWhatFollowsIt pins the deliberate reading of a
// malformed document: an opened fence that never closes takes the rest of the
// file with it, so the sections BEFORE it must still be reported.
func TestAnUnclosedFenceSilencesOnlyWhatFollowsIt(t *testing.T) {
	t.Parallel()

	diags := analyze(t, "README.md", "## Changelog\n\n```\n## Recent Changes\n")

	require.Len(t, diags, 1)
	assert.Equal(t, 1, diags[0].Line)
}

// TestTheFindingNamesTheHeadingAndItsLine pins what a reader acts on: the exact
// title, trimmed, and the line it sits on.
func TestTheFindingNamesTheHeadingAndItsLine(t *testing.T) {
	t.Parallel()

	diags := analyze(t, "README.md", "# Title\n\nsome prose\n\n##   Recent Changes   \n")

	require.Len(t, diags, 1)
	assert.Equal(t, 5, diags[0].Line)
	assert.Contains(t, diags[0].Message, `"Recent Changes"`, "the quoted title is trimmed")
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

	assert.Empty(t, analyze(t, "README.md", "  ```\n## Changelog\n  ```\n"),
		"three spaces is still a fence, which is where CommonMark draws the line")
}

// TestAnIndentedDelimiterNeverClosesAnOpenFence pins the same boundary on the
// closing side: a delimiter indented into code territory does not end the block
// it appears in.
func TestAnIndentedDelimiterNeverClosesAnOpenFence(t *testing.T) {
	t.Parallel()

	assert.Empty(t, analyze(t, "README.md", "```\n    ```\n## Changelog\n"),
		"the block is still open, so the heading inside it is an example")
}

// TestReStructuredTextAdornmentsUnderlineSectionsRatherThanFence pins the
// markup split. `~~~` is a fenced code block in markdown and a section
// adornment in reStructuredText, and reading an RST adornment as a fence
// silenced every heading in the rest of the file.
func TestReStructuredTextAdornmentsUnderlineSectionsRatherThanFence(t *testing.T) {
	t.Parallel()

	rst := "Guide\n=====\n\nUsage\n~~~~~\n\nChangelog\n=========\n\n- entry\n"
	assert.Len(t, analyze(t, "guide.rst", rst), 1, "the tilde rule underlines a section, it opens nothing")
	assert.Len(t, analyze(t, "guide.adoc", rst), 1, "AsciiDoc reads the same way")
	assert.Empty(t, analyze(t, "guide.md", rst), "in markdown the same text IS a fence, and the heading is inside it")
}

// TestEveryAdornmentCharacterUnderlinesASection pins the vocabulary of the
// two-line form. reStructuredText lets any repeated punctuation underline a
// title, so a rule that knew only markdown's two characters missed most RST
// headings anyone actually writes.
func TestEveryAdornmentCharacterUnderlinesASection(t *testing.T) {
	t.Parallel()

	for _, rule := range []string{"=========", "---------", "~~~~~~~~~", "^^^^^^^^^", `"""""""""`, "+++++++++", "#########", "*********"} {
		assert.Len(t, analyze(t, "guide.rst", "Changelog\n"+rule+"\n\n- entry\n"), 1,
			"%s underlines a section", rule)
	}

	assert.Empty(t, analyze(t, "guide.md", "Changelog\n~~~~~~~~~\n"), "markdown has only two of them")
}

// TestAnIndentedUnderlineIsNotAHeading pins the indentation bound on the
// two-line form, which the marker-run form already had.
func TestAnIndentedUnderlineIsNotAHeading(t *testing.T) {
	t.Parallel()

	assert.Empty(t, analyze(t, "README.md", "Changelog\n    =========\n"))
	assert.Len(t, analyze(t, "README.md", "Changelog\n   =========\n"), 1, "three spaces is still an underline")
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

// TestATitleIsTrimmedOfEveryKindOfSpace pins that the two written forms agree
// about the same words. The marker-run pattern consumes only spaces and tabs,
// so a heading ending in a vertical tab kept it and stopped matching while the
// identical title in the two-line form was reported — one character silenced
// the rule, and the forms disagreed.
func TestATitleIsTrimmedOfEveryKindOfSpace(t *testing.T) {
	t.Parallel()

	assert.Len(t, analyze(t, "README.md", "## Changelog\v\n"), 1, "a vertical tab is not part of the title")
	assert.Len(t, analyze(t, "README.md", "Changelog\v\n=========\n"), 1, "and the other form agrees")
}
