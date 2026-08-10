// Command yze-md-docfiles reports hand-maintained changelogs — the file, and
// the section inside another document that is one by another name — emitting
// the lean stickler-json report the stickler runner consumes.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	goyze "github.com/gomatic/go-yze"

	docfiles "github.com/gomatic/yze-md-docfiles"
)

// Injected collaborators, so the command is testable without real I/O.
var (
	osExit                 = os.Exit
	readFile               = func(path string) ([]byte, error) { return boundedRead(entryPath(path)) }
	openFile               = func(path string) (io.ReadCloser, error) { return os.Open(path) }
	statPath               = os.Stat
	walkDir                = filepath.WalkDir
	evalSymlinks           = filepath.EvalSymlinks
	lstatPath              = os.Lstat
	absolutePath           = filepath.Abs
	checkIgnore            = goyze.GitCheckIgnore
	stdout       io.Writer = os.Stdout
)

func main() { osExit(run(os.Args[1:])) }

// run expands the arguments to documents, runs the analyzer, and emits the
// report.
func run(args []string) int {
	files, err := documents(args)
	if err != nil {
		return fail(err)
	}
	// Report cannot fail: an unreadable or unparseable document becomes a
	// finding against that file rather than the run's error.
	report := docfiles.Report(readFile, files)
	if err := json.NewEncoder(stdout).Encode(report); err != nil {
		return fail(err)
	}
	return 0
}

// fail prints err to stderr and returns the failure exit code.
func fail(err error) int {
	_, _ = fmt.Fprintln(os.Stderr, "yze-md-docfiles:", err)
	return 1
}

// documents expands each argument: a directory contributes the documents
// beneath it, and any other path is taken verbatim.
//
// The result is DEDUPLICATED. Overlapping arguments are ordinary — a runner
// that passes both a directory and a file inside it is not making a mistake —
// and reporting one document three times tells its author there are three
// changelogs to delete.
func documents(args []string) ([]string, error) {
	var named, walked []string
	// SEPARATE identity sets: sharing one let a directory argument claim a
	// named document and hand it to the ignore filter, so the same arguments in
	// the opposite order gave the opposite verdict.
	isNamed, isWalked := map[string]bool{}, map[string]bool{}
	for _, arg := range args {
		found, isDir, err := expand(argument(arg))
		if err != nil {
			return nil, err
		}
		if isDir {
			walked = appendUnseen(walked, isWalked, found)
			continue
		}
		// A document NAMED outright is analyzed verbatim. Passing it through
		// the ignore filter answered a deliberate request with a silent clean
		// pass — the filter keeps a WALK from claiming files the repository
		// does not own; it does not overrule an author who asked.
		named = appendUnseen(named, isNamed, found)
	}
	return append(named, goyze.Tracked(checkIgnore, without(isNamed, walked))...), nil
}

// canonical is the path with symlinks resolved, used ONLY as the identity of a
// file. Keyed on the spelling, one document reached through a link and directly
// was reported twice — one file, one changelog, two findings.
func canonical(path entryPath) string {
	// ABSOLUTE FIRST, then resolved. EvalSymlinks resolves links without
	// absolutising, so a relative spelling kept the working directory's own
	// unresolved prefix while an absolute one did not — `CHANGELOG.md` and
	// `$PWD/CHANGELOG.md` came out as two identities for one document, and it
	// was reported twice. Absolutising afterwards does not fix that; the
	// prefix has to be resolved too, which only happens if it is there first.
	absolute, err := absolutePath(string(path))
	if err != nil {
		absolute = string(path)
	}
	resolved, err := evalSymlinks(absolute)
	if err != nil {
		return absolute
	}
	return resolved
}

// without drops the walked documents already claimed by name.
func without(named map[string]bool, walked []string) []string {
	kept := make([]string, 0, len(walked))
	for _, file := range walked {
		if !named[canonical(entryPath(file))] {
			kept = append(kept, file)
		}
	}
	return kept
}

// appendUnseen adds the documents not already collected, in the order they were
// found.
func appendUnseen(files []string, seen map[string]bool, found []string) []string {
	for _, file := range found {
		identity := canonical(entryPath(file))
		if !seen[identity] {
			seen[identity] = true
			files = append(files, file)
		}
	}
	return files
}

// expand is one argument's documents.
func expand(arg argument) ([]string, bool, error) {
	info, err := statPath(string(arg))
	switch {
	case err != nil:
		return nil, false, err
	case info.IsDir():
		found, walkErr := documentsUnder(searchDir(arg))
		return found, true, walkErr
	case !info.Mode().IsRegular():
		// Naming a FIFO or a device outright skips the walk's guard, and
		// reading one hangs the gate rather than failing it. Refusing by name
		// is loud; hanging is the one outcome nobody can diagnose.
		return nil, false, docfiles.ErrNotRegularFile.With(nil, "path", string(arg))
	}
	// Cleaned, so `a.md`, `./a.md` and `sub/../a.md` are one document rather
	// than three. Deduplication keyed on the raw string told an author there
	// were three changelogs to delete when there was one.
	return []string{filepath.Clean(string(arg))}, false, nil
}

// searchDir is a directory argument expanded recursively to the documents it
// contains.
type searchDir string

// argument is one path this command was asked to analyze.
type argument string
