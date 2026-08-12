// Package docfiles reports changelog FILES — and the section inside another
// document that is one by another name.
//
// A changelog file does not belong in a repository, generated or not. It
// duplicates what git already records, attributed and immutable, and the
// duplicate is the copy that rots: the file says one thing, `git log` says
// another, and the reader has no way to tell which is stale. Release notes are
// cut from that history and belong to the tag and the release, where they are
// written once and never edited again.
//
// GENERATED-NESS IS NOT AN EXEMPTION FROM THE FILE. It is a property of who
// typed the document; the ban is on the document EXISTING. The file finding was
// once suppressed by a generated claim, on the reasoning that telling a machine
// to stop hand-maintaining its output is nonsense — which confused a badly
// worded message with a reason, and made the exemption a door standing open in
// exactly the shape the ban is most likely broken in: release-please, git-cliff
// and goreleaser all write a `CHANGELOG.md` opening with `Code generated … DO
// NOT EDIT.` or `@generated`, and the rule reported nothing at all for every one
// of them.
//
// What the claim still exempts is the HEADING half, inside a document that is
// not itself a changelog. A machine-written docs page carrying four hundred
// `## Unreleased` headings is ONE problem, in its generator, and four hundred
// findings bury it — and no author can act on them, because editing that file
// is overwritten on the next run.
//
// The rule is deliberately NARROW, and every word of its vocabulary was
// measured against the fleet before being admitted. It reports a file whose
// name IS a changelog, in the prose extensions a changelog is written in, and a
// heading that opens a changelog section. It does not report a document merely
// called `history.md` — the fleet has two, both genuine Hugo content pages
// about a project's history — nor a source file named changelog.go, which is
// code that manages the concept rather than an instance of it.
//
// # WHAT WAS REFUSED, AND WHAT IT COST
//
// Every spelling was counted across the fleet before the decision, and the
// refusals matter as much as the admissions. A rule that fires on any of these
// is a rule its own repository cannot document itself under.
//
// Refused on a LEGITIMATE INSTANCE, which is the strongest reason there is:
//
//   - A bare `Changes` or `History` HEADING — ordinary prose in a design
//     document.
//   - `Release Notes` as a HEADING — three fleet instances, all three
//     legitimate: Go's own documentation ABOUT writing release notes, and two
//     docs-site index sections. It is banned as a file NAME, where it is
//     unambiguous, and left alone as a heading.
//   - `history.md` — two fleet instances, both genuine Hugo content pages about
//     a project's history.
//   - `changelog.go` — code that manages the concept rather than an instance of
//     it. `changelog_test.go` beside it is the fleet's only DECORATED instance,
//     and it is code twice over.
//   - A decoration that is a WORD: `changelog-policy.md` is a document about the
//     policy. Loosening the whole-stem anchor to a prefix or a substring reports
//     it, and that is why the anchor holds.
//
// Refused at ZERO instances, deliberately, and recorded so a later measurement
// can revisit rather than rediscover (all counts 2026-08-12, over 647 repository
// trees and 35,663 walked files, the work account excluded):
//
//   - `NEWS.md` — 0 files. It is the GNU and R ecosystems' changelog, and it is
//     also an ordinary page title on a website, in a fleet built out of Hugo
//     sites. That is precisely the shape `history.md` was refused for, on two
//     real instances, and admitting the one while refusing the other would make
//     the rule's narrowness a matter of which word came up first.
//
//   - `CHANGELOG.old` — 0 files. Admitting it means admitting a backup
//     vocabulary, and that vocabulary has to cover `changelog.md.bak` too — the
//     commoner spelling, already pinned unreported. Half a vocabulary is worse
//     than none: the rule's answer would depend on which backup convention an
//     author happened to use.
//
//   - The changelog kept as a DIRECTORY — `changelog/index.md`, Kubernetes'
//     `CHANGELOG/CHANGELOG-1.29.md`. 0 such directories fleet-wide. It was
//     ADMITTED for one commit and an adversarial review broke it: judging the
//     parent element reports every file in a repository whose own root directory
//     is named `changelog` — its README, its LICENSE — and the answer depends on
//     how the root was spelled on the command line, since `.` has no parent to
//     read. A gate whose answer depends on the shape of its argument cannot be
//     baselined, so the rule judges a path's final element and nothing above it.
//     Readmitting it needs the walk's ROOT, which no entry point here is given.
//
// Admitted at ZERO instances, because each is a real spelling of the banned
// thing with no legitimate document standing in its way: the numeric decoration
// (`CHANGELOG-1.29.md`, Kubernetes' literal layout; `CHANGELOG_2024.md`), the
// dotfile spelling (`.changelog.md`), and the missing extensions — `.mdx`,
// `.asciidoc`, every remaining spelling of markdown (`.mkd`, `.mdown`, `.mdwn`,
// `.mkdn`, `.mkdown`) and every prose markup `yze/markup` bans, of which the
// fleet holds 0 files of ANY name. Each admits a spelling rather than reporting
// one, which is the whole point of measuring first.
//
// # WHAT THIS RULE IS KNOWN NOT TO CATCH
//
// Stated because an adversarial review proved each of them, and a limitation
// nobody wrote down is indistinguishable from one nobody found.
//
//   - A heading whose title carries INLINE MARKUP. The vocabulary is matched
//     against a heading's SOURCE text, so `## **Changelog**`, “ ## `Changelog` “
//     and `## <a name="changelog"></a>Changelog` — the standard README anchor
//     idiom — all render `Changelog` and are all unreported. Reading the
//     rendered text needs the inline tree that [blockParser] deliberately does
//     not build, and the same omission is what keeps a megabyte of code spans
//     from taking 43.9 seconds; the two decisions are one decision, and this
//     one is the price of the other. (Invisible characters are a different
//     matter and ARE handled — see [visible].)
//
//   - A NESTED-CONTAINER document, on cost rather than on reading. Goldmark's
//     block grammar is quadratic in nesting depth and nothing here bounds it:
//     400 KB of `>` takes 50 seconds, and a megabyte takes 5m42s at 721 MB
//     resident, well inside the eight megabytes [SizeLimit] admits. A checked-in
//     file can therefore cost hours, and a gate that can be hung is a gate that
//     gets disabled. The bound belongs to the shared parse rather than to this
//     analyzer, since every sibling reading markdown has it.
//
//   - A PLAIN-TEXT table read as a section. `.txt` is parsed as markdown, so an
//     ASCII rule under a word — the universal plain-text table convention — is a
//     setext heading to CommonMark and nothing at all to a reader. `Unreleased`
//     over a run of dashes in a `.txt` is reported and should not be.
//
// # THE TWO HALVES READ DIFFERENT SETS OF FILES
//
// The name half judges a path and reads nothing, so it applies to every
// extension a changelog is spelled with — [nameExtensions] — including `.rst`
// and `.adoc`, which the `yze/markup` rule bans outright. Dropping them because
// another rule forbids the format would open a hole reachable through a
// SANCTIONED exemption: exemptions in this fleet are per-rule, so a repository
// that earns a legitimate `yze/markup` exemption would become a place where a
// `CHANGELOG.adoc` is invisible to this one.
//
// The section half needs a parse, so it applies only to what this rule parses —
// [sectionExtensions], which is markdown and the two spellings read as markdown.
// The capability given up is a changelog SECTION inside a reStructuredText or
// AsciiDoc document; measured over the fleet, that protects no file, in formats
// that must not exist here at all.
//
// They also read different LISTS, and [Report] takes both. A walk hands back one
// entry per NAME and one per FILE, and the two differ exactly where an alias
// exists. The name half asks for names, because there the name IS the finding:
// `ln -s docs/versions.md CHANGELOG.md` is a banned name in the repository, it
// survives a clone as mode 120000, and resolving it to its target deletes the
// evidence. The section half asks for files, one spelling per inode, because
// reading one document twice under two names reports one defect as two. Driving
// both halves off one list had to be wrong for one of them, and it was wrong for
// the half nothing else can catch.
//
// # A NAME OUTLIVES EVERY REASON THE BYTES COULD NOT BE READ
//
// A file that cannot be read is reported as a tool failure — [ErrTooLarge] for
// one past [SizeLimit], [ErrNotText] for one whose bytes are not text, and the
// reader's own error for one nobody could open. None of those suppresses what
// the rule already knows.
//
// A tool failure once meant NO findings, on the reasoning that an analyzer which
// read nothing has nothing to say. That reasoning holds for a section and fails
// for a name: an oversized `CHANGELOG.md` was told the analyzer could not read
// it, when what needed saying is that it must not exist. The size and text
// guards describe a file's CONTENTS, and they were deciding a rule that depends
// only on its NAME.
//
// So the contract is that BOTH are said. An unreadable document yields the
// findings its name earns — never any others, because nothing else was
// determined — together with the error, and every reader of this package
// composes them the same way: the name findings first, then the one that says
// the file could not be read. [Report] counts both, so a run's total is still
// the true one. Nothing is passed over in silence, which is the outcome a gate
// must never produce, and nothing is invented for a file nobody read either — an
// innocent oversized document still yields exactly one finding, the unreadable
// one.
package docfiles
