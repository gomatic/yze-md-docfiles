package docfiles_test

// Which markup family a path's spelling puts a document in. The table is
// exhaustive by construction; each entry's VALUE is a separate decision.

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestADotlessNameIsReadAsMarkdown pins the markup family the extensionless
// spelling gets, which is a decision and not a default. `CHANGELOG` with no
// extension is a markdown document by convention — it is what a repository
// writes — and reading it as reStructuredText makes its comment syntax a
// different one, so a heading hidden inside an HTML comment becomes visible and
// a real claim stops being read.
func TestADotlessNameIsReadAsMarkdown(t *testing.T) {
	t.Parallel()
	commented := "# Docs\n\n<!--\n## Changelog\n-->\n\n## Recent Changes\n"

	assert.Len(t, analyze(t, "docs/NOTES", commented), 1,
		"the commented heading is hidden, and only the live one is reported")
	assert.Len(t, analyze(t, "docs/NOTES.md", commented), 1,
		"which is the same answer the explicit spelling gets")
}
