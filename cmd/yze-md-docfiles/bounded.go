package main

// Bounding what is read. The limit belongs at the single place every path is
// read — not in the walk — because a bound that guards one entry point and not
// the other is not a bound: a named document still cost twice its own size
// before being refused (a 2 GiB file reached 4.3 GB resident), while the same
// file reached through the walk cost six megabytes.

import (
	"io"

	docfiles "github.com/gomatic/yze-md-docfiles"
)

// boundedRead reads a document only as far as it could still be prose.
//
// The bound is on the READ, not on a size asked for beforehand. Asking first
// answered a different question than the one that matters: it described the
// path rather than the bytes, so a stat that did not follow a symlink reported
// the link's own few bytes and read the two gigabytes behind it, and a file
// that grew between the question and the answer was read at its new size. One
// extra byte is read past the limit purely to tell "at the limit" from "over
// it"; nothing larger is ever resident.
func boundedRead(path entryPath) ([]byte, error) {
	file, err := openFile(string(path))
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, docfiles.SizeLimit+1))
	if err != nil {
		return nil, err
	}
	if len(data) > docfiles.SizeLimit {
		// The true size is deliberately NOT reported: learning it means reading
		// or stating the whole file, which is the cost this refusal exists to
		// avoid. The limit is the actionable number.
		return nil, docfiles.ErrTooLarge.With(nil, "path", string(path), "limit", docfiles.SizeLimit)
	}
	return data, nil
}
