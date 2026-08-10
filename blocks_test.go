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
