package main

// Bounding what is read. The limit belongs at the single place every path is
// read — not in the walk — because a bound that guards one entry point and not
// the other is not a bound: a named document once cost twice its own size before
// being refused, and a stat that did not follow a symlink read the two gigabytes
// behind it.

import (
	goyze "github.com/gomatic/go-yze"

	docfiles "github.com/gomatic/yze-md-docfiles"
)

// readFile reads a document, bounded, at the ONE place every path is read.
var readFile = func(path string) ([]byte, error) {
	return goyze.BoundedRead(files, goyze.FilePath(path), docfiles.SizeLimit)
}
