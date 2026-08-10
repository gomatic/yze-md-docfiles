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
	osExit             = os.Exit
	readFile           = os.ReadFile
	statPath           = os.Stat
	walkDir            = filepath.WalkDir
	stdout   io.Writer = os.Stdout
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
		found, err := expand(arg)
		if err != nil {
			return nil, err
		}
		files = appendUnseen(files, seen, found)
	}
	return files, nil
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

// expand is one argument's documents.
func expand(arg string) ([]string, error) {
	info, err := statPath(arg)
	switch {
	case err != nil:
		return nil, err
	case info.IsDir():
		return documentsUnder(searchDir(arg))
	case !info.Mode().IsRegular():
		// Naming a FIFO or a device outright skips the walk's guard, and
		// reading one hangs the gate rather than failing it. Refusing by name
		// is loud; hanging is the one outcome nobody can diagnose.
		return nil, docfiles.ErrNotRegularFile.With(nil, "path", arg)
	}
	return []string{arg}, nil
}

// searchDir is a directory argument expanded recursively to the documents it
// contains.
type searchDir string
