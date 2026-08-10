package docfiles

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNextLineReadsWindowsFilesExactlyAsUnixOnes pins the normalisation the
// streaming scan depends on. A carriage return rides at the end of every line a
// Windows editor writes, and leaving it in place makes a heading a reader
// plainly sees invisible to the rule. (The byte-order mark is stripped once, by
// the caller, before the first line is cut.)
func TestNextLineReadsWindowsFilesExactlyAsUnixOnes(t *testing.T) {
	t.Parallel()

	read := func(source string) []line {
		var lines []line
		text := remaining(source)
		for {
			one, rest, ok := nextLine(text)
			if !ok {
				return lines
			}
			lines, text = append(lines, one), rest
		}
	}

	assert.Equal(t, read("## Changelog\nbody\n"), read("## Changelog\r\nbody\r\n"),
		"a carriage return is not part of the line")
	assert.Equal(t, []line{"a", "", "b"}, read("a\n\nb"), "an empty line is a line")

	_, _, ok := nextLine("")
	assert.False(t, ok, "an exhausted document yields no line")
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
