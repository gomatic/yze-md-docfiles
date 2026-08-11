package docfiles

// Inline code spans, and the backtick runs that delimit them. A comment opened
// inside a span is not opened at all, so this is what keeps a document that
// SHOWS an HTML comment from being read as one that writes it.

// codeSpans is where each inline code span begins and how wide it is, computed
// in ONE pass over the line.
//
// It used to be answered per position, by scanning forward for a matching
// backtick run — which rescans the whole line for every run that never closes.
// A line of runs of strictly increasing length closes none of them, so the cost
// was superlinear in its length: eight megabytes of it, a size the limit
// deliberately admits, took 27 seconds and reported nothing, and four such files
// wedged the gate for two minutes.
func codeSpans(text line) map[int]width {
	runs := backtickRuns(text)
	next := searching(runsByLength(runs))
	spans := map[int]width{}
	for i := runIndex(0); int(i) < len(runs); i++ {
		closer, isClosed := next.after(runs, i)
		if !isClosed {
			// An unclosed run is literal text, and so is everything after it
			// until some later run closes.
			continue
		}
		spans[runs[i].at] = width(runs[closer].at + int(runs[closer].length) - runs[i].at)
		i = closer
	}
	return spans
}

// backtickRun is one run of backticks: where it starts and how long it is.
type backtickRunAt struct {
	at     int
	length runLength
}

// backtickRuns is every backtick run in the line, left to right.
func backtickRuns(text line) []backtickRunAt {
	var runs []backtickRunAt
	for at := 0; at < len(text); {
		if text[at] != '`' {
			at++
			continue
		}
		run := backtickRun(text[at:])
		runs = append(runs, backtickRunAt{at: at, length: run})
		at += int(run)
	}
	return runs
}

// runsByLength indexes the runs by length, so finding the next one of a given
// length is a step rather than a scan.
func runsByLength(runs []backtickRunAt) map[runLength][]int {
	byLength := map[runLength][]int{}
	for i, run := range runs {
		byLength[run.length] = append(byLength[run.length], i)
	}
	return byLength
}

// search finds each run's closer, remembering how far it has already looked at
// every length.
//
// The cursor is what makes the whole scan linear, and its absence is why the
// one-pass rewrite was still quadratic on the shape it was written for. Indexing
// by length turned "scan the line again" into "scan this length's runs again",
// which is the same cost when a line is nothing BUT runs of one length: ordinary
// prose with code spans has openers at candidate positions 0, 2, 4, …, and each
// restarted from the front. A megabyte of `x` spans took 39 seconds — 112 times
// slower than the pathological shape the regression test uses, at half the size
// — so an eight-megabyte file, which the size limit deliberately admits, was
// forty minutes of CPU in one call.
//
// A cursor never needs to go back: codeSpans asks about openers in increasing
// order, and a closer already passed can never be the answer for a later one.
type search struct {
	byLength map[runLength][]int
	cursor   map[runLength]int
}

// searching starts a search over runs already indexed by length.
func searching(byLength map[runLength][]int) search {
	return search{byLength: byLength, cursor: map[runLength]int{}}
}

// after is the first run past opener whose length matches — the closer
// CommonMark asks for, which is a backtick string of the same length and no
// other.
func (s search) after(runs []backtickRunAt, opener runIndex) (runIndex, bool) {
	length := runs[opener].length
	candidates := s.byLength[length]
	at := s.cursor[length]
	for at < len(candidates) && runIndex(candidates[at]) <= opener {
		at++
	}
	s.cursor[length] = at
	if at == len(candidates) {
		return 0, false
	}
	return runIndex(candidates[at]), true
}

// runIndex is a position in the list of backtick runs on one line.
type runIndex int
