package main

// What this command claims: whose prose is somebody else's. Everything else
// about turning arguments into files — the symlinked root, the identity of a
// path reached two ways, the tree that cannot be read, the size bound, the
// ignore filter — is the shared discovery's, because three analyzers answering
// those questions separately answered them differently.
//
// WHICH FILES ARE DOCUMENTS is not decided here either, and used to be. The walk
// kept its own list of extensions and its own list of changelog stems beside the
// pattern in the library, which is two vocabularies for one question: a spelling
// admitted in the library was invisible to every real invocation until somebody
// remembered to add it here as well, and nothing would have failed in between.
// The predicate is the library's now.

import (
	goyze "github.com/gomatic/go-yze"

	docfiles "github.com/gomatic/yze-md-docfiles"
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

// isDocument reports a path this rule reads, which is the library's own
// question: prose by its extension, or a changelog by its name whether or not it
// has one.
func isDocument(path goyze.FilePath) bool {
	return docfiles.Claims(docfiles.Path(path))
}
