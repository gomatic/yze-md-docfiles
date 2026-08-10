// Command yze-md-docfiles reports hand-maintained changelogs — the file, and
// the section inside another document that is one by another name — emitting
// the lean stickler-json report the stickler runner consumes.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	goyze "github.com/gomatic/go-yze"

	docfiles "github.com/gomatic/yze-md-docfiles"
)

// Injected collaborators, so the command is testable without real I/O.
//
// The filesystem is ONE value rather than a scatter of package variables. Each
// of the three source analyzers kept its own set and they drifted; this was the
// last of the three still walking its own tree, and it still had the defect the
// other two had already been repaired for.
var (
	osExit           = os.Exit
	files            = goyze.OSFileSystem()
	stdout io.Writer = os.Stdout
)

func main() { osExit(run(os.Args[1:])) }

// run expands the arguments to documents, runs the analyzer, and emits the
// report.
func run(args []string) int {
	if len(args) == 0 {
		// Being given nothing is an error, not a clean pass. A runner whose
		// root placeholder expands to nothing would otherwise green the gate
		// over a repository no analyzer ever looked at.
		return fail(docfiles.ErrNoPaths.With(nil))
	}
	found, err := discovery().Expand(args)
	if err != nil {
		return fail(err)
	}
	// Report cannot fail: an unreadable or unparseable document becomes a
	// finding against that file rather than the run's error.
	report := docfiles.Report(readFile, found.Files)
	report.Diagnostics = append(docfiles.Unreadable(found.Unreadable), report.Diagnostics...)
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
