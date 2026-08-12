package docfiles_test

// What a changelog is CALLED, in every spelling admitted and every spelling
// refused. The refused ones are the point: each was measured across the fleet
// before the decision, and each is a document somebody legitimately writes.

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	docfiles "github.com/gomatic/yze-md-docfiles"
)

// TestADecoratedChangelogStemIsReportedWhenTheDecorationIsANumber pins the one
// loosening of the whole-stem anchor. Kubernetes writes `CHANGELOG-1.29.md`
// literally, and year-partitioned changelogs write `CHANGELOG_2024.md`; a
// decoration that is a version or a year PARTITIONS a changelog, so the file is
// still one.
func TestADecoratedChangelogStemIsReportedWhenTheDecorationIsANumber(t *testing.T) {
	t.Parallel()

	for _, name := range []string{
		"CHANGELOG-1.29.md", "CHANGELOG_2024.md", "CHANGELOG-v2.md", "changelog.1.md",
		"CHANGES-2024.md", "release-notes-1.0.md", "Change Log 2024.txt",
	} {
		assert.Len(t, analyze(t, "docs/"+name, ""), 1, "%s is a changelog partitioned by a number", name)
	}
}

// TestADecorationThatIsAWordNamesADifferentDocument pins the constraint that
// makes the loosening safe, and it is the whole reason the anchor exists. A
// prefix or substring match reports `changelog-policy.md`, which is a document
// ABOUT the policy and is worth keeping — the same objection that refused a bare
// `Changes` heading. The fleet's one decorated instance, `changelog_test.go`, is
// code and refused twice over.
func TestADecorationThatIsAWordNamesADifferentDocument(t *testing.T) {
	t.Parallel()

	for _, name := range []string{
		"changelog-policy.md", "changelog_test.md", "changelog-of-doom.md",
		"changes-guide.md", "release-notes-template.md", "changelogs.md",
		"my-changelog.md", "the-changelog-2024.md",
	} {
		assert.Empty(t, analyze(t, "docs/"+name, ""), "%s names something else", name)
	}
}

// TestTheDotfileSpellingIsStillTheFile pins the leading dot, which is not an
// extension however [path.Ext] reads it: `.changelog.md` is a changelog whose
// extension is `.md`, and a bare `.changelog` has none at all. Reading the dot
// as part of the extension gave the dotfile spelling of the ban a suffix no
// table had heard of, and it went out silently.
func TestTheDotfileSpellingIsStillTheFile(t *testing.T) {
	t.Parallel()

	for _, name := range []string{".changelog.md", ".changelog", ".changes.txt", ".release-notes.md"} {
		assert.Len(t, analyze(t, "docs/"+name, ""), 1, "%s is the file with a dot in front of it", name)
	}

	for _, name := range []string{".changelogrc", ".md", ".changelog-policy.md", "."} {
		assert.Empty(t, analyze(t, "docs/"+name, ""), "%s is not", name)
	}
}

// TestAChangelogKeptAsADirectoryIsReported pins the shape the file half could
// not see, because it judged only a path's final element. Kubernetes keeps
// `CHANGELOG/CHANGELOG-1.29.md` and a Hugo site keeps `changelog/index.md`; the
// file inside is innocently named and the directory around it is the changelog.
func TestAChangelogKeptAsADirectoryIsReported(t *testing.T) {
	t.Parallel()

	for _, at := range []string{
		"changelog/index.md", "CHANGELOG/CHANGELOG-1.29.md", "content/changelog/2024.md",
		"docs/Release Notes/january.md", "changes/_index.md",
	} {
		diags := analyze(t, at, "")

		require.Len(t, diags, 1, "%s sits in a changelog directory", at)
		assert.Equal(t, 1, diags[0].Line)
	}

	for _, at := range []string{
		"changelogs/index.md", "docs/index.md", "changelog-policy/index.md", "index.md",
	} {
		assert.Empty(t, analyze(t, at, ""), "%s does not", at)
	}
}

// TestADirectoryFindingNamesTheDirectory pins what the message says, because
// the file's own name is innocent: an author told that `index.md` must not be
// committed has no idea what to do about it.
func TestADirectoryFindingNamesTheDirectory(t *testing.T) {
	t.Parallel()

	diags := analyze(t, "content/changelog/index.md", "")

	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "index.md sits in the changelog directory")
	assert.Contains(t, diags[0].Message, "kept as a directory")
}

// TestTheDocusaurusAndAsciidocSpellingsAreJudged pins the two extensions that
// were missing from the vocabulary. `.mdx` is Docusaurus' default and so the
// likeliest way a changelog arrives in a docs site; `.asciidoc` is AsciiDoc's
// other spelling, kept for the same reason `.adoc` is — a repository holding a
// sanctioned `yze/markup` exemption must not become the one place a
// `CHANGELOG.asciidoc` is invisible.
func TestTheDocusaurusAndAsciidocSpellingsAreJudged(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"CHANGELOG.mdx", "CHANGELOG.asciidoc", "changelog-1.29.mdx"} {
		assert.Len(t, analyze(t, "docs/"+name, ""), 1, "%s is a changelog file", name)
	}

	assert.Len(t, analyze(t, "docs/guide.mdx", "## Changelog\n"), 1,
		"an MDX document's markdown is CommonMark, so its sections are read")
	assert.Empty(t, analyze(t, "docs/guide.asciidoc", "== Changelog\n"),
		"while AsciiDoc is not parsed at all, exactly as .adoc is not")
}

// TestANameIsFoldedRatherThanLowerCased pins the folding this suite's sibling
// already does. `strings.ToLower` folds a strictly smaller set than a
// case-insensitive volume: U+017F folds to `s`, so `CHANGEſ.md` and `CHANGES.md`
// are ONE file on the default macOS volume while ToLower left them distinct. It
// is uniformity rather than a working bypass — two analyzers in one suite must
// not disagree about whether two names are the same name.
func TestANameIsFoldedRatherThanLowerCased(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"CHANGEſ.md", "changeſ.md", "CHANGEſ.MD"} {
		assert.Len(t, analyze(t, "docs/"+name, ""), 1, "%s folds onto CHANGES", name)
	}

	assert.Empty(t, analyze(t, "docs/changeſ-policy.md", ""), "and the anchor still holds after folding")
}

// TestTheRefusedSpellingsAreStillRefused is a regression test rather than an
// inspection. Each of these was a candidate, each was measured, and each was
// refused for a reason written into the package doc — so each has to stay
// unreported, or the refusal is a paragraph nothing enforces.
func TestTheRefusedSpellingsAreStillRefused(t *testing.T) {
	t.Parallel()

	for name, why := range map[string]string{
		"CHANGELOG.old":     "a backup extension, refused with `changelog.md.bak` rather than half of it",
		"changelog.md.bak":  "the commoner backup spelling, and already pinned silent",
		"CHANGELOG.orig":    "the same class",
		"NEWS.md":           "the GNU and R convention, and an ordinary page title on a website",
		"news.md":           "the same word in the shape this fleet actually writes",
		"history.md":        "two genuine fleet instances, both Hugo pages about a project's history",
		"changelog.go":      "code that manages the concept rather than an instance of it",
		"changelog_test.go": "the fleet's one decorated instance, and it is code",
		"CHANGELOG.png":     "not prose in any spelling",
	} {
		assert.Empty(t, analyze(t, "docs/"+name, ""), "%s: %s", name, why)
	}
}

// TestTheWalkClaimsEverySpellingTheRuleJudges pins the acceptance criterion an
// earlier round would have failed. The walk and the rule kept separate
// vocabularies, so a spelling admitted in one was invisible to every real
// invocation until somebody remembered the other — and nothing failed in
// between. There is one predicate now, and this is the assertion that every
// admitted spelling reaches it.
func TestTheWalkClaimsEverySpellingTheRuleJudges(t *testing.T) {
	t.Parallel()

	for _, at := range []string{
		"docs/CHANGELOG.md", "docs/CHANGELOG", "docs/CHANGELOG-1.29.md", "docs/.changelog.md",
		"docs/CHANGELOG.mdx", "docs/CHANGELOG.asciidoc", "docs/CHANGELOG.rst", "docs/CHANGEſ.md",
		"changelog/index.md", "docs/notes.md", "docs/notes.txt", "docs/guide.mdx",
	} {
		assert.True(t, docfiles.Claims(docfiles.Path(at)), "%s is a path this rule reads", at)
	}

	for _, at := range []string{
		"docs/changelog.go", "docs/image.png", "Makefile", "docs/notes.html", "docs/.changelogrc",
	} {
		assert.False(t, docfiles.Claims(docfiles.Path(at)), "%s is not", at)
	}
}
