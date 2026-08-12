package docfiles

// Which of the parsed extensions can carry a comment at all, from inside the
// package where the table can be named.

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCommentedExtensionsAnswersForEveryParsedExtension pins that the table is
// COMPLETE, and that the two spellings with no comment syntax say so rather than
// being absent. An extension missing from it is a decision nobody wrote down —
// and the decision is whether one literal line can exempt a hand-written
// changelog from the section half of the rule.
func TestCommentedExtensionsAnswersForEveryParsedExtension(t *testing.T) {
	t.Parallel()

	for ext := range sectionExtensions {
		_, isKnown := commentedExtensions[ext]
		assert.True(t, isKnown, "%q is an extension this rule decides about, so the table must answer for it", ext)
	}
	assert.Len(t, commentedExtensions, len(sectionExtensions), "and it answers for nothing else")

	for _, ext := range []extension{markdownExt, markdownLongExt} {
		assert.True(t, commentedExtensions[ext], "%q renders a comment as nothing", ext)
	}
	for _, ext := range []extension{plainTextExt, extensionlessExt} {
		assert.False(t, commentedExtensions[ext], "%q is shown to a reader verbatim", ext)
	}
	for _, ext := range []extension{restructuredExt, asciidocExt} {
		assert.False(t, sectionExtensions[ext], "%q is never parsed at all", ext)
		assert.False(t, commentedExtensions[ext], "so nothing written in it can claim anything", ext)
	}
}

// TestEveryHTMLBlockTypeIsDecided pins that the invisible-block table names all
// seven of CommonMark's HTML block types rather than only the one it admits. A
// type absent from it is a decision nobody made, and this table is the whole
// distinction between a claim a reader cannot see and one anybody can type.
func TestEveryHTMLBlockTypeIsDecided(t *testing.T) {
	t.Parallel()

	assert.Len(t, invisibleBlocks, 7, "CommonMark defines seven, and each is answered for")

	invisible := 0
	for _, isInvisible := range invisibleBlocks {
		if isInvisible {
			invisible++
		}
	}
	assert.Equal(t, 1, invisible, "exactly one of them — the comment — renders as nothing at all")
}
