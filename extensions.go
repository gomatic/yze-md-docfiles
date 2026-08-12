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

// nameExtensions is the set of extensions in which a file's NAME is judged. A
// changelog is prose; `changelog.go` is code that manages the concept rather
// than an instance of it, and reporting it would be reporting the cure as the
// disease. The empty extension is here for the canonical Unix spelling, a bare
// `CHANGELOG` with no extension at all.
//
// `.rst`, `.adoc` and `.asciidoc` stay here even though `yze/markup` bans all
// three formats outright, because that rule can be exempted per repository and
// this one is a different rule: dropping them would make a `CHANGELOG.adoc`
// invisible in just the repositories likely to hold one.
//
// `.mdx` is Docusaurus' default extension, and so the likeliest way a changelog
// arrives in a docs site. `.asciidoc` is AsciiDoc's other spelling. Both were
// absent, and both were measured before being admitted: 2026-08-12, over 647
// repository trees and 35,663 walked files, the fleet holds no file with either
// suffix at all — so this admits a spelling rather than reporting one.
var nameExtensions = map[extension]bool{
	extensionlessExt: true,
	markdownExt:      true,
	markdownLongExt:  true,
	markdownJSXExt:   true,
	plainTextExt:     true,
	restructuredExt:  true,
	asciidocExt:      true,
	asciidocLongExt:  true,
}

// sectionExtensions is the set this rule PARSES, and so the set whose headings
// it reads. Every admitted member is read as markdown, which is a decision
// rather than a default: `.txt` and the extensionless spelling are what a
// repository writes a bare `CHANGELOG` or `NOTES` in, and markdown is the only
// prose grammar this fleet has.
//
// `.mdx` is admitted because MDX is markdown with JSX added: its markdown
// constructs are CommonMark, so a heading is read as it renders, and the
// JSX it adds parses as an HTML block or a paragraph — neither of which invents
// a heading. What that costs is a heading INSIDE a JSX element, which CommonMark
// swallows as raw HTML; under-reading a construct nobody in this fleet writes is
// the safe direction, and it is written down rather than discovered later.
//
// The markup spellings are refused rather than omitted. There is no CommonMark
// reading of a reStructuredText document that means anything, and `yze/markup`
// bans the file itself — so the only capability given up is a changelog SECTION
// inside one, which the fleet measurement says protects no file. Writing the
// refusal down is what lets the next reader tell it from an oversight.
var sectionExtensions = map[extension]bool{
	extensionlessExt: true,
	markdownExt:      true,
	markdownLongExt:  true,
	markdownJSXExt:   true,
	plainTextExt:     true,
	restructuredExt:  false,
	asciidocExt:      false,
	asciidocLongExt:  false,
}

// extension is a file's suffix, case-folded.
type extension string

// The prose extensions this rule judges, named once so every place that decides
// something per-format stays in step.
const (
	markdownExt      extension = ".md"
	markdownLongExt  extension = ".markdown"
	markdownJSXExt   extension = ".mdx"
	plainTextExt     extension = ".txt"
	restructuredExt  extension = ".rst"
	asciidocExt      extension = ".adoc"
	asciidocLongExt  extension = ".asciidoc"
	extensionlessExt extension = ""
)
