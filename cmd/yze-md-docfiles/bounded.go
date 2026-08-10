package main

// Bounding what is read. The limit belongs at the single place every path is
// read — not in the walk — because a bound that guards one entry point and not
// the other is not a bound: a named document still cost twice its own size
// before being refused (a 2 GiB file reached 4.3 GB resident), while the same
// file reached through the walk cost six megabytes.

import (
	"os"

	docfiles "github.com/gomatic/yze-md-docfiles"
)

// boundedRead reads a document only when it is small enough to be prose.
//
// The size comes from a stat, before the file is opened, so an oversize
// document costs nothing to refuse — and it is still REPORTED, because a
// document too large to analyze is exactly where an unchecked changelog hides.
func boundedRead(path entryPath) ([]byte, error) {
	info, err := statPath(string(path))
	if err != nil {
		return nil, err
	}
	if info.Size() > docfiles.SizeLimit {
		return nil, docfiles.ErrTooLarge.With(nil, "path", string(path), "bytes", info.Size())
	}
	return os.ReadFile(string(path))
}
