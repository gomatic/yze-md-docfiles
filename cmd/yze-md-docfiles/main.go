// Command yze-md-docfiles reports changelog files — the file, generated or
// not, and the section inside another document that is one by another name —
// emitting the lean stickler-json report the stickler runner consumes.
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
	if err := report(args); err != nil {
		return fail(err)
	}
	return 0
}

// report is the run itself, as an ERROR rather than an exit code. run answers
// the process, which cannot be matched: with the refusal only ever reaching an
// int and a line of stderr, this command's sentinel could be swapped for any
// other with the whole suite green, so the failure the sentinel exists to name
// had nothing behind it.
func report(args []string) error {
	if len(args) == 0 {
		// Being given nothing is an error, not a clean pass. A runner whose
		// root placeholder expands to nothing would otherwise green the gate
		// over a repository no analyzer ever looked at.
		return docfiles.ErrNoPaths.With(nil)
	}
	found, err := discovery().Expand(args)
	if err != nil {
		return err
	}
	// The two halves of the rule read the walk's two lists. The name half judges
	// NAMES — every spelling, symlinks unresolved, because a symlink called
	// `CHANGELOG.md` is a banned name whatever innocent document it points at,
	// and it survives a clone as mode 120000. The section half reads FILES, one
	// spelling per inode, because reading one document twice under two names
	// reports one defect as two.
	//
	// Report cannot fail: an unreadable or unparseable document becomes a
	// finding against that file rather than the run's error.
	out := docfiles.Report(readFile, found.Names, found.Files)
	out.Diagnostics = append(docfiles.Unreadable(found.Unreadable), out.Diagnostics...)
	return json.NewEncoder(stdout).Encode(out)
}

// fail prints err to stderr and returns the failure exit code.
func fail(err error) int {
	_, _ = fmt.Fprintln(os.Stderr, "yze-md-docfiles:", err)
	return 1
}
