package docfiles_test

import (
	"strconv"
	"strings"
	"testing"

	goyze "github.com/gomatic/go-yze"

	docfiles "github.com/gomatic/yze-md-docfiles"
)

// FuzzDiagnostics drives arbitrary text and paths through the rule. The
// contract under fuzz, asserted on every input rather than merely exercised:
//
//   - Diagnostics never panics, on any path or content.
//   - Every diagnostic carries this rule's identity and a navigable 1-based
//     position, and its line never points past the document.
//   - A heading finding names a line that really is in the source, which is the
//     property that keeps the rule from inventing sections.
//
// That last assertion has to know HOW the message quotes a title. It compared
// raw text against a %q-escaped message and failed on the first heading holding
// a tab — a real defect in the assertion, found only because the target was run
// with -fuzz, which the gate does not do. Its input is a seed now.
//
// The seed corpus is the edge matrix: the banned spellings, the neighbours that
// must stay silent, fenced examples, unterminated fences, headings with odd
// spacing and escapes, and multi-byte runes.
func FuzzDiagnostics(f *testing.F) {
	for _, seed := range []string{
		"", "\n", "# Title\n", "## Changelog\n", "## changelog\n", "##Changelog\n",
		"## Change Process\n", "## Breaking Changes\n", "```\n## Changelog\n```\n",
		"```\n## Changelog\n", "## What's New\n", "## Whats New\n", "## Version History\n",
		"### Changelog   \n", "## Changelog é\n", "text\n## Recent Changes\nmore\n",
		"#\tWhAts New\n", "## \"Changelog\"\n", "##  Changelog ##\n", "~~~\n## Changelog\n~~~\n",
		"<!--\n## Changelog\n-->\n", "Changelog\n=========\n", "\ufeff## Changelog\n",
		"## Changelog\r\n", "````\n```\n## Changelog\n```\n````\n",
	} {
		f.Add("README.md", seed)
	}
	f.Add("CHANGELOG.md", "")
	f.Add("changelog.go", "## Changelog\n")
	// The families disagree about what a heading is, so each is seeded in its
	// own spelling and in the others' — a pattern shared between them was wrong
	// in both directions and no markdown-only corpus could reach it.
	f.Add("guide.adoc", "== Changelog\n")
	f.Add("guide.adoc", ".A caption\n----\n== Changelog\n----\n")
	f.Add("notes.rst", "=========\nChangelog\n=========\n")
	f.Add("notes.rst", "Intro\n\n----\n\nChangelog\n=========\n")
	f.Add("README.md", "## [Unreleased]\n")
	f.Add("README.md", "Shown: `<!--` opener\n\n## Changelog\n")
	f.Add("README.md", "- @generated\n\n## Changelog\n")

	f.Fuzz(func(t *testing.T, path, source string) {
		diags, err := docfiles.Diagnostics(docfiles.Path(path), docfiles.Source(source))
		if err != nil {
			if len(diags) != 0 {
				t.Fatalf("a tool failure yields no findings, got %d", len(diags))
			}
			return
		}

		lines := strings.Split(strings.TrimPrefix(source, "\ufeff"), "\n")
		for _, d := range diags {
			if d.Rule != docfiles.Rule || d.Tool != "yze" {
				t.Fatalf("diagnostic must carry this rule's identity, got %q/%q", d.Tool, d.Rule)
			}
			if d.Line < 1 || d.Col != 1 {
				t.Fatalf("diagnostic must be navigable, got line %d col %d", d.Line, d.Col)
			}
			if d.Severity != goyze.SeverityError {
				t.Fatalf("every finding here is an error, got %q", d.Severity)
			}
			// A heading finding must point at a line that exists and really is
			// the heading it quotes; only the file finding may sit at line 1 of
			// an empty document.
			if !strings.Contains(d.Message, "section is a changelog") {
				continue
			}
			if d.Line > len(lines) {
				t.Fatalf("heading finding at line %d, past a %d-line document", d.Line, len(lines))
			}
			// ONE assertion, and an equality. This was a disjunction whose
			// second half compared the message's title against the line the
			// title had just been taken FROM — true whenever the first half was
			// false, so any wrong title satisfied it and the property held for
			// every input including the broken ones. The title is unquoted and
			// must EQUAL the line it points at, stripped of the marker run that
			// is not part of it.
			title, unquoteErr := strconv.Unquote(quoted(d.Message))
			if unquoteErr != nil {
				t.Fatalf("heading finding must quote its title unambiguously, got %q", d.Message)
			}
			written := strings.TrimSuffix(lines[d.Line-1], "\r")
			if bareTitle(written) != title {
				t.Fatalf("heading finding names %q, but line %d is %q", title, d.Line, written)
			}
		}
	})
}

// quoted is the %q-rendered title a heading message carries.
func quoted(message string) string {
	_, rest, _ := strings.Cut(message, `the `)
	title, _, _ := strings.Cut(rest, ` section is a changelog`)
	return title
}

// bareTitle is a heading line with its marker run removed — what the message
// must name. Both written forms reduce to it: `## Changelog ##` and a
// `Changelog` sitting above its underline are the same title.
func bareTitle(written string) string {
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(written), "#="))
}
