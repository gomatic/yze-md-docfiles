package docfiles

// Which extensions each half of this rule judges, and which grammar the ones it
// parses are read in.
//
// One map used to do both jobs, and it could, for as long as every extension the
// rule judged was also one it read line by line. A real parse ended that: a
// reStructuredText document has no CommonMark reading that means anything, while
// its NAME is as knowable as any other. So there are two tables, and every
// extension appears in both — including the ones each refuses, because an
// extension absent from a table is a decision nobody wrote down.
//
// The membership is DERIVED rather than collected, from two rules:
//
//  1. Every spelling of markdown. A `CHANGELOG.mkd` is a changelog written in
//     markdown, spelled the way its author's editor spells it.
//  2. Every prose markup `yze/markup` bans outright, for the reason `.adoc` is
//     here: exemptions in this fleet are PER-RULE, so a repository that earns a
//     sanctioned markup exemption would otherwise become the one place a
//     `CHANGELOG.textile` is invisible to the changelog ban.
//
// Everything else is refused, and refused BY NAME below rather than by absence —
// an adversarial review found four markdown spellings sitting in neither list,
// where a reader could not tell an omission from a decision.

// nameExtensions is the set of extensions in which a file's NAME is judged. A
// changelog is prose; `changelog.go` is code that manages the concept rather
// than an instance of it, and reporting it would be reporting the cure as the
// disease. The empty extension is here for the canonical Unix spelling, a bare
// `CHANGELOG` with no extension at all.
var nameExtensions = map[extension]bool{
	// Markdown, in every spelling that renders as markdown.
	extensionlessExt: true,
	markdownExt:      true,
	markdownLongExt:  true,
	markdownJSXExt:   true,
	markdownDownExt:  true,
	markdownWnExt:    true,
	markdownMkdExt:   true,
	markdownMkdnExt:  true,
	markdownMkdownXt: true,
	// Plain text, which is not a markup and is where a `CHANGELOG.txt` lives.
	plainTextExt: true,
	// The prose markups `yze/markup` bans. Judged here anyway, per rule 2 above.
	restructuredExt: true,
	asciidocExt:     true,
	asciidocLongExt: true,
	textileExt:      true,
	mediawikiExt:    true,
	wikiExt:         true,
	creoleExt:       true,
	podExt:          true,
	rdocExt:         true,
	orgModeExt:      true,
	// REFUSED. Each was considered and each is a decision, not an oversight.
	// `.html` is rendered output rather than authored prose; `.tex` is a
	// typesetting system for documents markdown does not replace; `.text` is
	// plain text under another name, and admitting it would put a name rule on a
	// suffix nothing agrees is prose. Zero fleet instances of any of them
	// carrying a changelog name, measured 2026-08-12.
	htmlExt:     false,
	htmlLongExt: false,
	latexExt:    false,
	plainDocExt: false,
}

// sectionExtensions is the set this rule PARSES, and so the set whose headings
// it reads. Every admitted member is read as markdown, which is a decision
// rather than a default: `.txt` and the extensionless spelling are what a
// repository writes a bare `CHANGELOG` or `NOTES` in, and markdown is the only
// prose grammar this fleet has.
//
// `.mdx` is admitted because MDX is markdown with JSX added: its markdown
// constructs are CommonMark, so a heading is read as it renders, and the JSX it
// adds parses as an HTML block or a paragraph — neither of which invents a
// heading. What that costs is a heading INSIDE a JSX element, which CommonMark
// swallows as raw HTML; under-reading a construct nobody in this fleet writes is
// the safe direction, and it is written down rather than discovered later.
//
// The markups are refused rather than omitted. There is no CommonMark reading of
// a reStructuredText document that means anything, and `yze/markup` bans the
// file itself — so the only capability given up is a changelog SECTION inside
// one, which the fleet measurement says protects no file. Writing the refusal
// down is what lets the next reader tell it from an oversight.
var sectionExtensions = map[extension]bool{
	extensionlessExt: true,
	markdownExt:      true,
	markdownLongExt:  true,
	markdownJSXExt:   true,
	markdownDownExt:  true,
	markdownWnExt:    true,
	markdownMkdExt:   true,
	markdownMkdnExt:  true,
	markdownMkdownXt: true,
	plainTextExt:     true,
	restructuredExt:  false,
	asciidocExt:      false,
	asciidocLongExt:  false,
	textileExt:       false,
	mediawikiExt:     false,
	wikiExt:          false,
	creoleExt:        false,
	podExt:           false,
	rdocExt:          false,
	orgModeExt:       false,
	htmlExt:          false,
	htmlLongExt:      false,
	latexExt:         false,
	plainDocExt:      false,
}

// extension is a file's suffix, case-folded.
type extension string

// The extensions this rule decides about, named once so every place that
// decides something per-format stays in step.
const (
	markdownExt      extension = ".md"
	markdownLongExt  extension = ".markdown"
	markdownJSXExt   extension = ".mdx"
	markdownDownExt  extension = ".mdown"
	markdownWnExt    extension = ".mdwn"
	markdownMkdExt   extension = ".mkd"
	markdownMkdnExt  extension = ".mkdn"
	markdownMkdownXt extension = ".mkdown"
	plainTextExt     extension = ".txt"
	restructuredExt  extension = ".rst"
	asciidocExt      extension = ".adoc"
	asciidocLongExt  extension = ".asciidoc"
	textileExt       extension = ".textile"
	mediawikiExt     extension = ".mediawiki"
	wikiExt          extension = ".wiki"
	creoleExt        extension = ".creole"
	podExt           extension = ".pod"
	rdocExt          extension = ".rdoc"
	orgModeExt       extension = ".org"
	htmlExt          extension = ".html"
	htmlLongExt      extension = ".htm"
	latexExt         extension = ".tex"
	plainDocExt      extension = ".text"
	extensionlessExt extension = ""
)
