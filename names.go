package docfiles

// The half of this rule that reads no bytes: what a changelog is CALLED.
//
// Every spelling here was counted across the fleet before it was admitted, and
// the ones that were refused are recorded in the package doc with their counts.
// The whole-stem anchor is the design: `changelog-policy.md` is a document ABOUT
// the policy, and a prefix or substring match reports it.

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	goyze "github.com/gomatic/go-yze"
)

// baseName is a file's final path element.
type baseName string

// pathPart is one element of a path, or a piece of one — a base name, a
// directory name, an extension — as it is written on disk, before it is folded
// for comparison.
type pathPart string

// stem is a file's name with its extension removed, case-folded — what the
// vocabulary is matched against.
type stem string

// changelogFileName is the file this rule bans, in the spellings it is written
// in: CHANGELOG, ChangeLog, changelog, CHANGES, and the hyphenated, underscored
// and spaced variants — `Release Notes.md` is the spelling a person types.
//
// The name must be the WHOLE stem, optionally followed by a NUMBER. That
// exception is the Kubernetes layout, `CHANGELOG-1.29.md`, and the
// year-partitioned `CHANGELOG_2024.md`: a decoration that is a version or a year
// partitions a changelog, while a decoration that is a WORD names a different
// document. The separator is any of the four this vocabulary already spells its
// own words with, space included — `Release Notes 1.0.md` is what a person types
// when a person is doing the typing. `changelog-policy.md` is the standing example and stays unreported,
// and so does `changelog_test.go`, the fleet's one decorated instance — measured
// 2026-08-12, one file, code rather than prose.
var changelogFileName = regexp.MustCompile(`^(change[-_ ]?log|changes|release[-_ ]?notes)([-_. ]v?[0-9][0-9.]*)?$`)

// Claims reports a path this rule reads, and is exported because the command's
// discovery walks with it.
//
// ONE predicate, so the walk and the rule cannot disagree about which spellings
// exist. They did: the walk kept its own list of changelog stems beside this
// package's pattern, which meant a spelling admitted here was invisible to every
// real invocation until somebody remembered to add it there too.
func Claims(at Path) bool {
	base, ext := nameAndExtension(at)
	return readExtensions[ext] || isChangelogPath(at, base, ext)
}

// readExtensions is every extension the walk hands over on sight, derived from
// the two judging tables rather than typed out a third time. A third list is a
// third place an admission has to land, and the one that gets forgotten decides
// whether the file ever reaches the rule at all.
//
// The extensionless spelling is removed from it deliberately. It is the
// commonest form of the very thing this rule bans AND the form every Makefile,
// licence and binary takes, so a dotless file is claimed by its NAME below
// rather than on sight.
var readExtensions = onSight(nameExtensions, sectionExtensions)

// onSight is the union of the extensions two tables judge, less the
// extensionless spelling.
func onSight(tables ...map[extension]bool) map[extension]bool {
	read := map[extension]bool{}
	for _, table := range tables {
		for ext, judged := range table {
			read[ext] = read[ext] || judged
		}
	}
	delete(read, extensionlessExt)
	return read
}

// nameAndExtension is a path's final element and its case-folded extension.
//
// A leading dot belongs to the NAME rather than to an extension: `.changelog.md`
// is a changelog whose extension is `.md`, and a bare `.changelog` has none at
// all. [filepath.Ext] answers `.changelog` for the second, which would give the
// dotfile spelling of the ban an extension no table has ever heard of.
//
// It is [filepath], not [path], and on the platform this is written on the two
// behave alike — which is how the difference went unnoticed for as long as it
// did.
// The walk uses [filepath.WalkDir], so on Windows every path arrives
// backslash-separated, where `path.Base` returns the WHOLE path unchanged: the
// stem of `C:\repo\CHANGELOG.md` came out as `c:\repo\changelog`, matched
// nothing, and the whole name half of this rule went silent on a platform
// `.goreleaser.yaml` builds for. Found by an adversarial review, which is the
// only thing that could find it — no test on this machine can tell the two
// packages apart.
func nameAndExtension(at Path) (baseName, extension) {
	base := filepath.Base(string(at))
	return baseName(base), extension(folded(pathPart(filepath.Ext(strings.TrimPrefix(base, ".")))))
}

// stemOf is a name with its extension and any leading dot removed, case-folded.
func stemOf(base baseName, ext extension) stem {
	return stem(strings.TrimPrefix(strings.TrimSuffix(string(folded(pathPart(base))), string(ext)), "."))
}

// folded is text with every rune replaced by its case-folding representative:
// the smallest rune of its simple-folding orbit, lower-cased.
//
// Folded, not lower-cased, and the difference is uniformity rather than an
// exploit. `strings.ToLower` folds a strictly smaller set than a case-insensitive
// volume does: U+017F LATIN SMALL LETTER LONG S folds to `s`, so on the default
// macOS APFS volume `CHANGEſ.md` and `CHANGES.md` are ONE file — the same inode —
// while ToLower left the long s alone. Two of the words here contain an `s`.
//
// On a case-sensitive filesystem the two names really are different files and no
// tool would read `changeſ.md` as a changelog, so this is not a working bypass;
// [yze-md-markup] already folds this way, and two analyzers in one suite must not
// disagree about whether two names are the same name.
//
// The mapping is a literal because its signature is [strings.Map]'s rather than
// this package's, which is the one case a domain type cannot be declared for a
// parameter.
//
// [yze-md-markup]: https://github.com/gomatic/yze-md-markup
func folded(raw pathPart) pathPart {
	return pathPart(strings.Map(func(r rune) rune {
		smallest := r
		for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
			if next < smallest {
				smallest = next
			}
		}
		return unicode.ToLower(smallest)
	}, string(raw)))
}

// isChangelogPath reports a path whose NAME says it is a changelog.
//
// It judges the final element and NOTHING ABOVE IT. A changelog kept as a
// DIRECTORY — `changelog/index.md`, Kubernetes' `CHANGELOG/CHANGELOG-1.29.md` —
// was admitted here for one commit and is refused; the reason is recorded in the
// package doc, and it is a false positive an adversarial review demonstrated
// rather than an argument.
func isChangelogPath(_ Path, base baseName, ext extension) bool {
	if !nameExtensions[ext] {
		return false
	}
	return changelogFileName.MatchString(string(stemOf(base, ext)))
}

// nameFindings is everything this rule can say about a path knowing only its
// name — which is every reader with no bytes to offer: the walk reporting a
// path it could not enter, and the reader refusing a file it could not open.
// One function serves all three callers, so the two byteless readers and the
// ordinary read reach one conclusion about one name.
func nameFindings(at Path) []goyze.Diagnostic {
	base, ext := nameAndExtension(at)
	return fileDiagnostics(at, base, ext)
}

// fileDiagnostics reports the file when its name is a changelog.
func fileDiagnostics(at Path, base baseName, ext extension) []goyze.Diagnostic {
	if !isChangelogPath(at, base, ext) {
		return nil
	}
	return []goyze.Diagnostic{diagnostic(at, 1, nameFinding(base))}
}

// nameFinding is the message for one banned name.
func nameFinding(base baseName) finding {
	return finding(fmt.Sprintf(fileMessage, base))
}
