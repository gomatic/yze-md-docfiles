package docfiles

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHeadingFindingQuotesATitleUnambiguously pins how a title reaches the
// reader. A heading may hold a quote, a tab or a backslash, and pasting one
// into the sentence unescaped produces a message that cannot be read back to
// the line it came from — which is exactly what broke the fuzz target's
// invariant.
func TestHeadingFindingQuotesATitleUnambiguously(t *testing.T) {
	t.Parallel()

	message := string(headingFinding(`Change	Log`))

	assert.Contains(t, message, `"Change\tLog"`, "the tab is escaped, not embedded raw")
	require.NotContains(t, message, "Change\tLog", "and the raw form never reaches the message")
}

// TestUnbracketedRemovesOnlyASurroundingPair names the narrowness the doc
// comment claims. Keep a Changelog writes `## [Unreleased]`, so the brackets
// have to come off before the vocabulary is consulted — but only a matched
// surrounding pair, or a title merely MENTIONING a bracket would be rewritten
// into something its author never typed.
func TestUnbracketedRemovesOnlyASurroundingPair(t *testing.T) {
	t.Parallel()

	for name, title := range map[string]struct{ in, want line }{
		"surrounded":  {"[Unreleased]", "Unreleased"},
		"open only":   {"[Unreleased", "[Unreleased"},
		"close only":  {"Unreleased]", "Unreleased]"},
		"inner":       {"see [1] here", "see [1] here"},
		"bare":        {"Unreleased", "Unreleased"},
		"empty":       {"", ""},
		"one bracket": {"[", "["},
		"empty pair":  {"[]", ""},
	} {
		assert.Equal(t, title.want, unbracketed(title.in), "%s", name)
	}
}

// TestUnlinkedRemovesOnlyAWholeSurroundingLink names the narrowness the strip
// claims. Keep a Changelog writes `## [Unreleased](…/compare)`, so the target
// has to come off before the vocabulary is consulted — but only when the WHOLE
// title is one link, or a heading that merely mentions one would be rewritten
// into something its author never typed.
func TestUnlinkedRemovesOnlyAWholeSurroundingLink(t *testing.T) {
	t.Parallel()

	for name, title := range map[string]struct{ in, want line }{
		"whole link":     {"[Unreleased](https://x/y)", "[Unreleased]"},
		"relative":       {"[Changelog](./CHANGELOG.md)", "[Changelog]"},
		"empty target":   {"[Unreleased]()", "[Unreleased]"},
		"no target":      {"[Unreleased]", "[Unreleased]"},
		"bare":           {"Unreleased", "Unreleased"},
		"no closer":      {"[Unreleased](https://x/y", "[Unreleased](https://x/y"},
		"two links":      {"[a](x) and [b](y)", "[a](x) and [b](y)"},
		"link mid-title": {"See [the log](x) here", "See [the log](x) here"},
		"no opener":      {"Changelog](x)", "Changelog](x)"},
		"empty":          {"", ""},
	} {
		assert.Equal(t, title.want, unlinked(title.in), "%s", name)
	}
}
