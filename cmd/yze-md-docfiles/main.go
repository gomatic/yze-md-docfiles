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
	var files []string
	seen := map[string]bool{}
	for _, arg := range args {
		found, err := expand(argument(arg))
		if err != nil {
			return nil, err
		}
		files = appendUnseen(files, seen, found)
	}
	return tracked(checkIgnore, ignoreRoot(files), files), nil
}

// appendUnseen adds the documents not already collected, in the order they were
// found.
func appendUnseen(files []string, seen map[string]bool, found []string) []string {
	for _, file := range found {
		if !seen[file] {
			seen[file] = true
			files = append(files, file)
		}
	}
	return files
}

// ignoreRoot is the directory the ignore question is asked from: the first
// document's own directory, which is inside the repository being analyzed. A
// run covers one repository — that is what a gate does — so one root answers
// for every path in it.
func ignoreRoot(files []string) repoDir {
	if len(files) == 0 {
		return "."
	}
	return repoDir(filepath.Dir(files[0]))
}

// expand is one argument's documents.
func expand(arg argument) ([]string, error) {
	info, err := statPath(string(arg))
	switch {
	case err != nil:
		return nil, err
	case info.IsDir():
		return documentsUnder(searchDir(arg))
	case !info.Mode().IsRegular():
		// Naming a FIFO or a device outright skips the walk's guard, and
		// reading one hangs the gate rather than failing it. Refusing by name
		// is loud; hanging is the one outcome nobody can diagnose.
		return nil, docfiles.ErrNotRegularFile.With(nil, "path", string(arg))
	}
	// Cleaned, so `a.md`, `./a.md` and `sub/../a.md` are one document rather
	// than three. Deduplication keyed on the raw string told an author there
	// were three changelogs to delete when there was one.
	return []string{filepath.Clean(string(arg))}, nil
}

// searchDir is a directory argument expanded recursively to the documents it
// contains.
type searchDir string

// argument is one path this command was asked to analyze.
type argument string
