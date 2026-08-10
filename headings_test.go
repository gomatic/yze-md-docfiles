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

// TestTheFindingNamesTheHeadingAndItsLine pins what a reader acts on: the exact
// title, trimmed, and the line it sits on.
func TestTheFindingNamesTheHeadingAndItsLine(t *testing.T) {
	t.Parallel()

	diags := analyze(t, "README.md", "# Title\n\nsome prose\n\n##   Recent Changes   \n")

	require.Len(t, diags, 1)
	assert.Equal(t, 5, diags[0].Line)
	assert.Contains(t, diags[0].Message, `"Recent Changes"`, "the quoted title is trimmed")
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

// TestAnUnderlinedTitleIsStillAHeadingInAdornmentMarkup pins the distinction
// the blank line carries: the same characters underline a title when one sits
// directly above them.
func TestAnUnderlinedTitleIsStillAHeadingInAdornmentMarkup(t *testing.T) {
	t.Parallel()

	assert.Len(t, analyze(t, "guide.rst", "Changelog\n---------\n\n* 1.0\n"), 1)
	assert.Len(t, analyze(t, "guide.adoc", "== Changelog\n\n* 1.0\n"), 1, "and the marker-run form still works")
}
