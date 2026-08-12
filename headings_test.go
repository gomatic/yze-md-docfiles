package docfiles_test

// The changelog SECTION: which titles the vocabulary names, and which lines the
// parse says are titles at all.
//
// What a heading IS belongs to CommonMark and is proven in goldmark's own
// suite; the tests here are this rule's own contract — the vocabulary, the title
// the message quotes, and the property that keeps the whole model honest: a
// document SHOWING a banned heading is not a document that has one, and the
// block it is shown in closes so the sections after it are still read.

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

// TestAChangelogTitleIsAnchoredAtBothEnds pins the vocabulary's NARROWNESS,
// which only half had a test. Dropping the start anchor left the suite green
// while turning `## Project Changelog` and `## The Unreleased` into findings —
// headings that merely end with a banned word and are nobody's changelog. The
// end anchor's equivalent mutation was already killed; this is the other side of
// the same contract.
func TestAChangelogTitleIsAnchoredAtBothEnds(t *testing.T) {
	t.Parallel()

	for _, title := range []string{
		"## Project Changelog", "## The Unreleased", "## Our Recent Changes", "## Old Version History",
	} {
		assert.Empty(t, analyze(t, "README.md", title+"\n\nprose\n"),
			"%q names something else that ends with the words", title)
	}

	for _, title := range []string{"## Changelog", "## Unreleased", "## Recent Changes", "## Version History"} {
		assert.Len(t, analyze(t, "README.md", title+"\n\nprose\n"), 1,
			"%q is the section itself", title)
	}
}

// TestBothWrittenFormsReachTheSameVocabulary pins that a marker-run heading and
// an underlined one are read as the same words. The two forms were once matched
// by two patterns of this package's own, and they disagreed — so a title was
// reported or silent according to which form its author had chosen. The parser
// answers for both now, and this is the assertion that they arrive together.
func TestBothWrittenFormsReachTheSameVocabulary(t *testing.T) {
	t.Parallel()

	for name, source := range map[string]string{
		"marker run":       "## Changelog\n",
		"setext equals":    "Changelog\n=========\n",
		"setext dashes":    "Recent Changes\n--------------\n",
		"keep a changelog": "## [Unreleased]\n",
	} {
		assert.Len(t, analyze(t, "README.md", source), 1, "%s is a heading", name)
	}
}

// TestAByteOrderMarkNeverHidesTheFirstLine pins this package's own
// normalisation, which is not the parser's. A mark rides invisibly ahead of the
// first character, and a parser that has not been told about it reads the first
// line as ordinary paragraph text — hiding both a heading a reader plainly sees
// and, worse, the generated claim a generator wrote on exactly that line.
func TestAByteOrderMarkNeverHidesTheFirstLine(t *testing.T) {
	t.Parallel()

	assert.Len(t, analyze(t, "README.md", "\ufeff## Changelog\n"), 1, "the heading is a heading")
	assert.Empty(t, analyze(t, "api.md", "\ufeff<!-- @generated -->\n\n## Changelog\n"),
		"and the claim behind one is still a claim")
}

// TestATitleIsTrimmedOfEveryKindOfSpace pins the trim this rule does on top of
// the parser's. CommonMark ends a heading's text at its spaces and tabs and
// leaves every other space character in place, so a title ending in a vertical
// tab reached the vocabulary with it attached and stopped matching — one
// character silencing the rule.
func TestATitleIsTrimmedOfEveryKindOfSpace(t *testing.T) {
	t.Parallel()

	assert.Len(t, analyze(t, "README.md", "## Changelog\v\n"), 1, "a vertical tab is not part of the title")
	assert.Len(t, analyze(t, "README.md", "Changelog\v\n=========\n"), 1, "and the other form agrees")
	assert.Len(t, analyze(t, "README.md", "## Changelog\u00a0\n"), 1, "nor is a non-breaking space")
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

// TestAnUnderlinedTitleIsReportedOnTheTitleLine pins the position of the OTHER
// written form, where the two lines that make the heading invite an off-by-one:
// the finding must name the line the words are on, not the rule beneath them,
// because that is where an author's editor has to land.
func TestAnUnderlinedTitleIsReportedOnTheTitleLine(t *testing.T) {
	t.Parallel()

	diags := analyze(t, "README.md", "intro\n\nChangelog\n=========\n")

	require.Len(t, diags, 1)
	assert.Equal(t, 3, diags[0].Line)
	assert.Contains(t, diags[0].Message, `"Changelog"`)
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

	for _, title := range []string{
		"## [Contributing](CONTRIBUTING.md)",
		"## See [the changelog](CHANGELOG.md) for details",
	} {
		assert.Empty(t, analyze(t, "README.md", title+"\n\nprose\n"), "%s is not one", title)
	}
}

// TestAWrappedTitleIsNotReadAsASection pins the one heading shape this rule
// declines. A title WRAPPED across lines is written in no spelling this
// vocabulary names, and it is already a `yze/hardwrap` finding in this same
// suite — while reading it here would have to either quote a title spanning
// lines the finding does not point at, or quote its first line alone, which
// invents a section out of `Changelog` standing above `of Doom`.
func TestAWrappedTitleIsNotReadAsASection(t *testing.T) {
	t.Parallel()

	assert.Empty(t, analyze(t, "README.md", "Changelog\nof Doom\n=======\n"),
		"the first line alone is not the title")
	assert.Empty(t, analyze(t, "README.md", "Recent\nChanges\n=======\n"),
		"nor are the lines together a spelling this vocabulary names")
	assert.Empty(t, analyze(t, "README.md", "##\n\nprose\n"), "and a heading with no text has no title at all")
	assert.Len(t, analyze(t, "README.md", "Recent Changes\n=======\n"), 1,
		"while the same words on one line are the section")
}

// TestAnExampleInsideABlockIsNotASection pins the exemption without which the
// rule could not describe itself: the standards show banned shapes inside
// fences, comments and code blocks, and reporting those makes the documentation
// of a rule a violation of it.
func TestAnExampleInsideABlockIsNotASection(t *testing.T) {
	t.Parallel()

	for name, source := range map[string]string{
		"backtick fence": "```\n## Changelog\n```\n",
		"tilde fence":    "~~~\n## Changelog\n~~~\n",
		"info string":    "```markdown\n## Changelog\n```\n",
		"nested fence":   "````markdown\n```text\n## Changelog\n```\n````\n",
		"html comment":   "<!--\n## Changelog\n-->\n",
		"indented fence": "  ```\n  ## Changelog\n  ```\n",
		"indented code":  "    ## Changelog\n",
		"tab code":       "\t## Changelog\n",
		"html block":     "<div>\n# Changelog\n</div>\n",
		"list fence":     "- ```\n  # Changelog\n  ```\n",
	} {
		assert.Empty(t, analyze(t, "README.md", source), "%s holds an example", name)
	}
}

// TestABlockClosesSoLaterSectionsAreStillRead pins the property that makes the
// exemption a model rather than a switch. A block that never closed would
// silence the whole rest of a document — one ``` line disabling every finding
// after it — and a real fleet document with nested fences did exactly that.
func TestABlockClosesSoLaterSectionsAreStillRead(t *testing.T) {
	t.Parallel()

	for name, source := range map[string]string{
		"after a plain fence":  "```\ncode\n```\n\n## Changelog\n",
		"after a tilde fence":  "~~~\ncode\n~~~\n\n## Changelog\n",
		"after a nested fence": "````\n```\ncode\n```\n````\n\n## Changelog\n",
		"after a comment":      "<!-- note -->\n\n## Changelog\n",
		"after a long comment": "<!--\nnote\n-->\n\n## Changelog\n",
		"after an html block":  "<div>\nx\n</div>\n\n## Changelog\n",
		"after indented code":  "    code\n\n## Changelog\n",
	} {
		require.Len(t, analyze(t, "README.md", source), 1, "%s: the block closed, so the section is read", name)
	}
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

// TestAnInlineCommentDelimiterDoesNotSilenceTheDocument pins a behaviour the
// hand-rolled scanner had BACKWARDS, and it is the reason the parse replaced it.
// That scanner modelled a browser's comment nesting rather than the format's, so
// a bare `<!--` written mid-sentence — three characters, no closing marker
// needed, no trace — commented away every finding below it for the rest of the
// file. CommonMark opens an HTML comment BLOCK only on a line that begins one;
// anywhere else the delimiter is inline text and the headings under it are still
// headings.
func TestAnInlineCommentDelimiterDoesNotSilenceTheDocument(t *testing.T) {
	t.Parallel()

	for name, source := range map[string]string{
		"opened mid-line":    "Intro text <!--\n## Changelog\n",
		"closed then opened": "<!-- a --> tail <!--\n## Changelog\n",
		"never closed":       "Intro <!-- hidden\n\n## Changelog\n",
		"shown in a span":    "Shown: `<!--` opener\n\n## Changelog\n",
	} {
		assert.Len(t, analyze(t, "README.md", source), 1,
			"%s: the delimiter is text in a paragraph, not a block a document hides behind", name)
	}

	assert.Empty(t, analyze(t, "README.md", "<!--\n## Changelog\n-->\n\nEnd.\n"),
		"while a comment BLOCK really does hide what it holds")
}

// TestALinkReferenceDefinitionDoesNotHideTheHeadingBelowIt pins a gap the block
// parser had for one commit. A link reference definition is lifted out of its
// paragraph by a PARAGRAPH TRANSFORMER, and the parser was assembled without
// them — so `[semver]: https://semver.org/` sitting above a setext heading left
// the definition inside the paragraph, the heading's text spanned two lines, and
// the finding was discarded. One ordinary line of link definitions above a
// `Changelog` heading silenced it. Found by an adversarial review.
func TestALinkReferenceDefinitionDoesNotHideTheHeadingBelowIt(t *testing.T) {
	t.Parallel()

	for name, source := range map[string]string{
		"one definition": "[x]: /y\nChangelog\n=========\n",
		"two definitions": "[keep-a-changelog]: https://keepachangelog.com/\n" +
			"[semver]: https://semver.org/\nWhat's New\n----------\n",
	} {
		assert.Len(t, analyze(t, "README.md", source), 1, "%s: the heading below it is a heading", name)
	}
}
