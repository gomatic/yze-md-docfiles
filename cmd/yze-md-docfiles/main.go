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

	docfiles "github.com/gomatic/yze-md-docfiles"
)

// Injected collaborators, so the command is testable without real I/O.
var (
	osExit                 = os.Exit
	readFile               = os.ReadFile
	statPath               = os.Stat
	walkDir                = filepath.WalkDir
	evalSymlinks           = filepath.EvalSymlinks
	lstatPath              = os.Lstat
	checkIgnore            = gitCheckIgnore
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
	report, err := docfiles.Report(readFile, files)
	if err != nil {
		return fail(err)
	}
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
	seen := map[string]bool{}
	for _, arg := range args {
		found, isDir, err := expand(argument(arg))
		if err != nil {
			return nil, err
		}
		if isDir {
			walked = appendUnseen(walked, seen, found)
			continue
		}
		// A document NAMED outright is analyzed verbatim. Passing it through
		// the ignore filter answered a deliberate request with a silent clean
		// pass — the filter keeps a WALK from claiming files the repository
		// does not own; it does not overrule an author who asked.
		named = appendUnseen(named, seen, found)
	}
	return append(named, tracked(checkIgnore, walked)...), nil
}

// canonical is the path with symlinks resolved, used ONLY as the identity of a
// file. Keyed on the spelling, one document reached through a link and directly
// was reported twice — one file, one changelog, two findings.
func canonical(path entryPath) string {
	resolved, err := evalSymlinks(string(path))
	if err != nil {
		return string(path)
	}
	return resolved
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
