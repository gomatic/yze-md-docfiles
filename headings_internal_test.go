package docfiles

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDocumentLinesReadsWindowsAndBomFilesExactlyAsUnixOnes pins the
// normalisation the scan depends on. A byte-order mark precedes the first
// character while being invisible in the text, and a carriage return rides at
// the end of every line a Windows editor writes; leaving either in place makes
// a heading a reader plainly sees invisible to the rule.
func TestDocumentLinesReadsWindowsAndBomFilesExactlyAsUnixOnes(t *testing.T) {
	t.Parallel()

	unix := documentLines("## Changelog\nbody\n")

	assert.Equal(t, unix, documentLines("\ufeff## Changelog\nbody\n"), "a BOM changes nothing")
	assert.Equal(t, unix, documentLines("## Changelog\r\nbody\r\n"), "CRLF changes nothing")
}

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
