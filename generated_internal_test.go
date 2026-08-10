package docfiles

// Which comment syntax each prose format has, from inside the package where the
// table can be named.

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeneratedClaimsNamesEveryProseExtension pins that the table is COMPLETE,
// and that the two formats with no comment syntax say so rather than being
// absent. An extension missing from it is a decision nobody wrote down — and the
// decision is whether one literal line can exempt a hand-written changelog from
// the whole rule.
func TestGeneratedClaimsNamesEveryProseExtension(t *testing.T) {
	t.Parallel()

	for ext := range prose {
		_, isKnown := generatedClaims[ext]
		assert.True(t, isKnown, "%q is prose this rule reads, so the table must answer for it", ext)
	}
	assert.Len(t, generatedClaims, len(prose), "and it answers for nothing else")

	for _, ext := range []extension{plainTextExt, extensionlessExt} {
		claim, hasComments := generatedClaim(ext)
		assert.False(t, hasComments, "%q has no comment syntax", ext)
		assert.Nil(t, claim)
	}
	for _, ext := range []extension{markdownExt, markdownLongExt, restructuredExt, asciidocExt} {
		claim, hasComments := generatedClaim(ext)
		assert.True(t, hasComments, "%q does", ext)
		require.NotNil(t, claim)
	}
}
