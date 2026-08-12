package docfiles_test

// What a changelog is CALLED, in every spelling admitted and every spelling
// refused. The refused ones are the point: each was measured across the fleet
// before the decision, and each is a document somebody legitimately writes.

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
// extension however [filepath.Ext] reads it: `.changelog.md` is a changelog whose
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

// TestAChangelogKeptAsADirectoryIsNotReported pins a candidate that was
// ADMITTED for one commit and is refused, and this is the regression test for
// the refusal rather than an inspection.
//
// A directory named `changelog` holding `index.md` really is a changelog, and
// judging the parent element reported it. It also reported every file in a
// repository whose OWN root directory is called `changelog` — its README, its
// LICENSE, its docs — and the answer depended on how the root was spelled on the
// command line: `docfiles /src/changelog` gave three findings and `cd
// /src/changelog && docfiles .` gave none, because `.` has no parent element to
// read. A gate whose answer depends on the shape of its argument cannot be
// baselined. Found by an adversarial review; zero such directories exist in the
// fleet, so the refusal costs nothing measured.
func TestAChangelogKeptAsADirectoryIsNotReported(t *testing.T) {
	t.Parallel()

	for _, at := range []string{
		"changelog/index.md", "CHANGELOG/notes.md", "content/changelog/2024.md",
		"docs/Release Notes/january.md", "changes/_index.md",
	} {
		assert.Empty(t, analyze(t, at, ""), "%s is judged by its own name, not its parent's", at)
	}

	assert.Len(t, analyze(t, "CHANGELOG/CHANGELOG-1.29.md", ""), 1,
		"while the file inside that really is named one is still reported, once")
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

// TestEverySpellingOfMarkdownIsJudged pins the extensions an adversarial review
// found sitting in neither the admitted list nor the refused one. GitHub renders
// `.mkd`, `.mdown`, `.mdwn`, `.mkdn` and `.mkdown` as markdown, so each is a
// changelog spelled the way its author's editor spells it — and the reasoning
// that admitted `.mdx` at zero fleet instances applies to each of them verbatim.
func TestEverySpellingOfMarkdownIsJudged(t *testing.T) {
	t.Parallel()

	for _, ext := range []string{".mkd", ".mkdn", ".mkdown", ".mdown", ".mdwn"} {
		assert.Len(t, analyze(t, "docs/CHANGELOG"+ext, ""), 1, "CHANGELOG%s is a changelog file", ext)
		assert.Len(t, analyze(t, "docs/guide"+ext, "## Changelog\n"), 1, "and %s is read as markdown", ext)
	}
}

// TestABannedMarkupSpellingIsStillJudgedByName pins the second derivation rule.
// `yze/markup` bans these formats outright, and exemptions in this fleet are
// PER-RULE — so a repository holding a sanctioned markup exemption must not
// become the one place a `CHANGELOG.textile` is invisible to the changelog ban.
// `.rst` and `.adoc` were here for that reason and the rest of the banned set was
// not, which is the same gap wearing seven more suffixes.
func TestABannedMarkupSpellingIsStillJudgedByName(t *testing.T) {
	t.Parallel()

	for _, ext := range []string{".textile", ".mediawiki", ".wiki", ".creole", ".pod", ".rdoc", ".org"} {
		assert.Len(t, analyze(t, "docs/CHANGELOG"+ext, "## Changelog\n"), 1,
			"CHANGELOG%s is banned by name, and its contents are not parsed", ext)
		assert.Empty(t, analyze(t, "docs/guide"+ext, "## Changelog\n"),
			"while %s is not read as markdown, exactly as .adoc is not", ext)
	}
}

// TestTheRefusedExtensionsAreStillRefused pins the other half of that decision.
// `.html` is rendered output rather than authored prose, `.tex` is a typesetting
// system markdown does not replace, and `.text` is plain text under a name
// nothing agrees is prose. Each is recorded as a refusal, so each has to stay
// unreported or the record is a paragraph nothing enforces.
func TestTheRefusedExtensionsAreStillRefused(t *testing.T) {
	t.Parallel()

	for _, ext := range []string{".html", ".htm", ".tex", ".text"} {
		assert.Empty(t, analyze(t, "docs/CHANGELOG"+ext, "## Changelog\n"),
			"CHANGELOG%s is refused by name and unparsed", ext)
	}
}

// TestAnInvisibleCharacterNeverSilencesAHeading pins the repair for a
// one-character opt-out. Unicode's format characters render as nothing, so a
// heading carrying one shows a reader exactly `Changelog` — while the vocabulary,
// matched against the raw source text, saw a word it had never heard of. A zero
// width space, a soft hyphen, a byte order mark mid-line and a left-to-right
// mark each silenced the rule completely. Found by an adversarial review; it is
// the same defect as the vertical tab this rule was already repaired for, in
// characters nobody can see at all.
func TestAnInvisibleCharacterNeverSilencesAHeading(t *testing.T) {
	t.Parallel()

	for name, heading := range map[string]string{
		"zero width space": "## Changelog\u200b",
		"byte order mark":  "## \ufeffChangelog",
		"soft hyphen":      "## Change\u00adlog",
		"left-to-right":    "## Changelog\u200e",
		"word joiner":      "## Change\u2060log",
		"zero width join":  "## Change\u200dlog",
	} {
		assert.Len(t, analyze(t, "README.md", heading+"\n"), 1,
			"%s renders as nothing, so the reader sees a changelog", name)
	}

	assert.Empty(t, analyze(t, "README.md", "## Chàngelog\n"),
		"while a character a reader CAN see is a different word, and combining marks stay")
}
