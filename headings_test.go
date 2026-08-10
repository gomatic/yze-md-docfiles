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
