package docfiles

// How much of a RUN a report may carry, and what it says when it carries less
// than it found. The size limit bounds one document; nothing bounded a tree of
// them.

// reportLimit bounds how many findings ONE RUN carries.
//
// The per-document limit bounds a document; it does not bound a tree. Two
// thousand documents each at their own limit produced 490 MB of report and
// 2.3 GB resident from 31 MB of input — the same failure the per-document
// limit exists to prevent, reached by a route it does not cover. A run with
// this many findings is one problem too, and the true count is still named.
const reportLimit = 10_000

// runTruncationMessage formats the finding that stands for the rest of a run.
const runTruncationMessage = "%d changelog findings across this run, of which %d are reported; findings from %s " +
	"onward are omitted, because a tree with this many is one problem rather than that many"
