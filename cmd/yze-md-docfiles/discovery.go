package main

// What this command claims: what counts as a document, and whose prose is
// somebody else's. Everything else about turning arguments into files — the
// symlinked root, the identity of a path reached two ways, the tree that cannot
// be read, the size bound, the ignore filter — is the shared discovery's,
// because three analyzers answering those questions separately answered them
// differently.

import (
	"path/filepath"
	"strings"

	goyze "github.com/gomatic/go-yze"
)

// discovery is this command's file discovery: the shared walk, told what a
// document is and whose trees to skip.
func discovery() goyze.Discovery {
	return goyze.Discovery{Files: files, Claims: isDocument, Prunes: pruned}
}

// pruned reports the trees that hold somebody else's prose. A dependency's
// changelog is its own business, and reporting it tells this repository to
// delete a file it does not own — a Python virtualenv's vendored licence files
// turned up in the first real sweep, one of them not even valid UTF-8.
//
// This list names only trees that are somebody else's in EVERY repository. What
// a particular repository ignores is git's answer, not a list's, and `.git` and
// nested checkouts are the shared walk's business. `testdata` is here for a
// reason specific to this family: it is where an analyzer is proven in both
// directions, so a fixture that MUST contain a violation would otherwise be
// reported as one.
func pruned(name goyze.DirName) bool {
	switch name {
	case "vendor", "node_modules", "themes", "testdata", ".venv", "venv", ".tox":
		return true
	}
	return false
}

// documentExtensions are the suffixes prose is written in.
var documentExtensions = map[string]bool{
	".md": true, ".markdown": true, ".txt": true, ".rst": true, ".adoc": true,
}

// changelogStems are the file names that are a changelog whatever they are
// called next. They are claimed WITHOUT an extension because the canonical Unix
// spelling has none: a bare `CHANGELOG` is the most common form of the very
// thing this rule bans, and requiring an extension made it invisible to every
// real invocation while the test suite advertised it as covered.
var changelogStems = map[string]bool{
	"changelog": true, "change-log": true, "change_log": true, "change log": true,
	"changes": true, "releasenotes": true, "release-notes": true, "release_notes": true,
	"release notes": true,
}

// isDocument reports a path this rule reads: prose by its extension, or a
// changelog by its name whether or not it has one.
func isDocument(path goyze.FilePath) bool {
	base := strings.ToLower(filepath.Base(string(path)))
	if documentExtensions[filepath.Ext(base)] {
		return true
	}
	// Every changelog stem is dotless, so a name reaching here with an
	// extension cannot match one — the guard that used to sit in front of this
	// lookup could never be false, which is an unreachable condition dressed as
	// a check.
	return changelogStems[base]
}
