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
// `.rst` and `.adoc` stay here even though `yze/markup` bans both formats
// outright, because that rule can be exempted per repository and this one is a
// different rule: dropping them would make a `CHANGELOG.adoc` invisible in
// just the repositories likely to hold one.
var nameExtensions = map[extension]bool{
	extensionlessExt: true,
	markdownExt:      true,
	markdownLongExt:  true,
	plainTextExt:     true,
	restructuredExt:  true,
	asciidocExt:      true,
}

// sectionExtensions is the set this rule PARSES, and so the set whose headings
// it reads. Every admitted member is read as markdown, which is a decision
// rather than a default: `.txt` and the extensionless spelling are what a
// repository writes a bare `CHANGELOG` or `NOTES` in, and markdown is the only
// prose grammar this fleet has.
//
// The two markup spellings are refused rather than omitted. There is no
// CommonMark reading of a reStructuredText document that means anything, and
// `yze/markup` bans the file itself — so the only capability given up is a
// changelog SECTION inside one, which the fleet measurement says protects no
// file. Writing the refusal down is what lets the next reader tell it from an
// oversight.
var sectionExtensions = map[extension]bool{
	extensionlessExt: true,
	markdownExt:      true,
	markdownLongExt:  true,
	plainTextExt:     true,
	restructuredExt:  false,
	asciidocExt:      false,
}

// extension is a file's suffix, lower-cased.
type extension string

// The prose extensions this rule reads, named once so the three places that
// decide something per-format stay in step.
const (
	markdownExt      extension = ".md"
	markdownLongExt  extension = ".markdown"
	plainTextExt     extension = ".txt"
	restructuredExt  extension = ".rst"
	asciidocExt      extension = ".adoc"
	extensionlessExt extension = ""
)
