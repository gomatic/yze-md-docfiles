package docfiles_test

import (
	"fmt"
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
		"closed ATX":       "## Changelog ##\n",
		"one space":        " ## Changelog\n",
		"three spaces":     "   ## Changelog\n",
		"tab separator":    "##\tChangelog\n",
		"byte-order mark":  "\ufeff## Changelog\n",
		"carriage return":  "## Changelog\r\n",
		"trailing space":   "## Changelog   \n",
		"setext":           "Changelog\n=========\n",
		"setext dashes":    "Recent Changes\n--------------\n",
		"keep a changelog": "## [Unreleased]\n",
	} {
		assert.Len(t, analyze(t, "README.md", source), 1, "%s is a heading", name)
	}
}

// TestAMarkerRunHeadingIsReadInTheFamilyThatSpellsItThatWay pins the marker-run
// form PER FAMILY, because the three formats do not agree on it and one pattern
// serving all three was wrong in both directions. AsciiDoc titles are `==`, and
// Asciidoctor also accepts markdown's `##`; markdown has only `##`, and a line
// beginning `= ` there is prose; reStructuredText has no marker-run form at all
// — its `#` is an adornment character, and reading a `#` line as a heading
// invented sections in a format that has none.
func TestAMarkerRunHeadingIsReadInTheFamilyThatSpellsItThatWay(t *testing.T) {
	t.Parallel()

	for name, spelling := range map[string]struct {
		file, source string
		want         int
	}{
		"asciidoc equals":      {"guide.adoc", "== Changelog\n", 1},
		"asciidoc hash":        {"guide.adoc", "## Changelog\n", 1},
		"markdown hash":        {"README.md", "## Changelog\n", 1},
		"markdown equals":      {"README.md", "= Changelog\n", 0},
		"restructured hash":    {"notes.rst", "## Changelog\n", 0},
		"restructured equals":  {"notes.rst", "= Changelog\n", 0},
		"restructured adorned": {"notes.rst", "Changelog\n=========\n", 1},
	} {
		assert.Len(t, analyze(t, spelling.file, spelling.source), spelling.want, "%s", name)
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

// TestAMarkerRunHeadingIsReadBeforeItsUnderline pins the order the two written
// forms are tried in. `# Changelog` followed by a run of equals signs is both a
// marker-run heading and a line with an underline beneath it; read as the
// second, the title is `# Changelog`, which matches no vocabulary and reports
// nothing at all.
func TestAMarkerRunHeadingIsReadBeforeItsUnderline(t *testing.T) {
	t.Parallel()

	assert.Len(t, analyze(t, "README.md", "# Changelog\n===========\n"), 1,
		"the marker-run form wins, so the title is Changelog rather than # Changelog")
}

// TestTheReportedTitleKeepsItsBrackets pins what an author is shown. The Keep a
// Changelog spelling arrives bracketed, and the vocabulary is matched without
// them — but the message must name the text the author can search their
// document for, not the stripped form nobody wrote.
func TestTheReportedTitleKeepsItsBrackets(t *testing.T) {
	t.Parallel()

	diags := analyze(t, "README.md", "## [Unreleased]\n")

	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, `"[Unreleased]"`)
}

// TestTheLinkedKeepAChangelogSpellingIsRead pins the OTHER form Keep a Changelog
// writes. `## [Unreleased](https://…/compare)` is what its own template emits,
// and the vocabulary saw a title that was mostly a URL.
func TestTheLinkedKeepAChangelogSpellingIsRead(t *testing.T) {
	t.Parallel()

	for _, title := range []string{
		"## [Unreleased]",
		"## [Unreleased](https://github.com/o/r/compare/v1.0.0...HEAD)",
		"## [Changelog](./CHANGELOG.md)",
	} {
		assert.Len(t, analyze(t, "README.md", title+"\n"), 1, "%s opens a changelog", title)
	}

	for _, title := range []string{
		"## [Contributing](CONTRIBUTING.md)",
		"## See [the changelog](CHANGELOG.md) for details",
	} {
		assert.Empty(t, analyze(t, "README.md", title+"\n"), "%s is not one", title)
	}
}

// TestEveryAdornmentCharacterUnderlinesASection pins the whole reStructuredText
// vocabulary rather than a sample of it. Five of the thirteen had nothing behind
// them: each could be dropped from the pattern with the suite still green, and
// each drop makes a real `Changelog` heading invisible — which is not a missing
// finding but a silent pass on the one thing this rule looks for.
func TestEveryAdornmentCharacterUnderlinesASection(t *testing.T) {
	t.Parallel()

	for _, adornment := range []string{"=", "-", "~", "^", `"`, "+", "#", "*", "'", "`", ":", ".", "_"} {
		source := fmt.Sprintf("Guide\n=====\n\nChangelog\n%s\n\nrecent things\n", repeated(adornment, 9))

		assert.Len(t, analyze(t, "guide.rst", source), 1,
			"a section underlined with %q is a section", adornment)
	}
}

// TestBothMarkdownLinkFormsAreReadInATitle pins the reference spelling beside
// the inline one. `[Unreleased](…/compare)` and `[Unreleased][unreleased]`
// render identically and Keep a Changelog's own template has used each, so
// reading one and not the other left the same heading reported or silent
// according to which spelling its author copied.
func TestBothMarkdownLinkFormsAreReadInATitle(t *testing.T) {
	t.Parallel()

	for name, title := range map[string]string{
		"inline":    "## [Unreleased](https://example.test/compare/v1...HEAD)",
		"reference": "## [Unreleased][unreleased]",
		"plain":     "## Unreleased",
	} {
		assert.Len(t, analyze(t, "notes.md", title+"\n\nrecent things\n"), 1,
			"%s: the link is the target, and the title is what is left", name)
	}
}
