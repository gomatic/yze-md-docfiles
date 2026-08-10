package main

// Discovery decides which files this command hands to the analyzer: what counts
// as a document, which trees are somebody else's, and what is not prose at all.

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// walkRoot is the directory a walk started from.
type walkRoot string

// entryPath is the path of one entry the walk visited.
type entryPath string

// entryName is a single path element — one directory's own name.
type entryName string

// documentsUnder walks dir collecting every document beneath it.
func documentsUnder(dir searchDir) ([]string, error) {
	root := resolvedRoot(dir)
	var files []string
	err := walkDir(root, func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir():
			return prunedDir(walkRoot(root), entryPath(path), entryName(d.Name()))
		case readableSource(entryPath(path), d) && isDocument(entryPath(path)):
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// resolvedRoot is the directory a walk starts from, with a symlinked root
// followed. The walk lstats its own root, so a symlink to a directory reported
// itself as a non-directory and the walk yielded NOTHING — a deliberate request
// answered with a silent clean pass, which is the one result a gate must never
// invent. A root that cannot be resolved is cleaned and walked as given.
func resolvedRoot(dir searchDir) string {
	cleaned := filepath.Clean(string(dir))
	info, err := lstatPath(cleaned)
	if err != nil || info.Mode()&fs.ModeSymlink == 0 {
		// Only a symlinked root is resolved. Resolving every root would rewrite
		// ordinary paths too — on this platform a temp directory sits under a
		// symlinked prefix — so the same document reached by two arguments
		// would be reported under two different names.
		return cleaned
	}
	resolved, err := evalSymlinks(cleaned)
	if err != nil {
		return cleaned
	}
	return resolved
}

// prunedDir decides whether to descend into one directory. The walk root is
// never pruned: naming `testdata` outright and being handed a silent clean pass
// is worse than any tree this skips, and a gate that answers a deliberate
// request with nothing is the one result it must never invent.
func prunedDir(root walkRoot, path entryPath, name entryName) error {
	if string(path) == string(root) {
		return nil
	}
	if pruned(name) || nestedRepository(path) {
		return fs.SkipDir
	}
	return nil
}

// pruned reports the trees that hold somebody else's prose. A dependency's
// changelog is its own business, and reporting it tells this repository to
// delete a file it does not own — a Python virtualenv's vendored licence files
// turned up in the first real sweep, one of them not even valid UTF-8.
//
// This list names only trees that are somebody else's in EVERY repository. What
// a particular repository ignores — a coverage directory, a plugin cache — is
// git's answer, not a list's, and is applied separately; a prune list that tried
// to know both would grow forever and still be wrong for the next repository.
// `testdata` is here for a reason specific to this family: it is where an
// analyzer is proven in both directions, so a fixture that MUST contain a
// violation would otherwise be reported as one.
func pruned(name entryName) bool {
	switch name {
	case ".git", "vendor", "node_modules", "themes", "testdata", ".venv", "venv", ".tox":
		return true
	}
	return false
}

// nestedRepository reports a directory that is its own git checkout — a
// submodule, or a sibling repository sitting inside this tree. Its documents
// belong to that repository and are gated by that repository's own run.
func nestedRepository(path entryPath) bool {
	_, err := statPath(filepath.Join(string(path), ".git"))
	return err == nil
}

// readableSource reports a path whose contents are prose this rule can read.
//
// A FIFO blocks forever on open, and a device or socket is not a document in
// any case. A SYMLINK is followed rather than dropped: publishing a document
// through a link is ordinary, and the walk reports the link's own mode, so
// testing that silently skipped the file the author actually wrote.
func readableSource(path entryPath, d fs.DirEntry) bool {
	if d.Type().IsRegular() {
		return true
	}
	if d.Type()&fs.ModeSymlink == 0 {
		return false
	}
	info, err := statPath(string(path))
	return err == nil && info.Mode().IsRegular()
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
	"changelog": true, "change-log": true, "change_log": true,
	"changes": true, "releasenotes": true, "release-notes": true, "release_notes": true,
}

// isDocument reports a path this rule reads: prose by its extension, or a
// changelog by its name whether or not it has one.
func isDocument(path entryPath) bool {
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
