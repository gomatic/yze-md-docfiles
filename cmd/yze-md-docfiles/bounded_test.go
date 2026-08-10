package main

// What this command refuses to read. The bound sits at the single place every
// path is read, so the walk and a named argument cannot disagree about it — and
// it bounds the READ rather than a size asked for beforehand, which described
// the path rather than the bytes and read two gigabytes behind a symlink.

import (
	"os"
	"path/filepath"
	"testing"

	goyze "github.com/gomatic/go-yze"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	docfiles "github.com/gomatic/yze-md-docfiles"
)

// TestReadFileBoundsEveryPathItReads pins both sides of the boundary, so the
// limit cannot be moved in either direction without a failure.
func TestReadFileBoundsEveryPathItReads(t *testing.T) {
	dir := t.TempDir()

	for name, size := range map[string]goyze.ByteCount{
		"at the limit": docfiles.SizeLimit,
		"over it":      docfiles.SizeLimit + 1,
	} {
		at := filepath.Join(dir, name+".md")
		require.NoError(t, os.WriteFile(at, nil, 0o600))
		require.NoError(t, os.Truncate(at, int64(size)))

		data, err := readFile(at)

		if size > docfiles.SizeLimit {
			assert.ErrorIs(t, err, goyze.ErrTooLarge, name)
			continue
		}
		require.NoError(t, err, name)
		assert.Len(t, data, int(size), name)
	}
}

// TestReadFileRefusesAnOversizeDocumentThroughASymlink pins the bound against
// the path that defeated it. The size used to be asked for with a stat BEFORE
// the read, which describes a path rather than the bytes behind it: a stat that
// did not follow the link reported the link's own few bytes and read the two
// gigabytes it pointed at. The walk deliberately follows symlinks, so this is an
// ordinary document, not a contrived one.
func TestReadFileRefusesAnOversizeDocumentThroughASymlink(t *testing.T) {
	dir := t.TempDir()
	huge := filepath.Join(dir, "huge.md")
	require.NoError(t, os.WriteFile(huge, nil, 0o600))
	require.NoError(t, os.Truncate(huge, int64(docfiles.SizeLimit)+1))
	link := filepath.Join(dir, "link.md")
	require.NoError(t, os.Symlink(huge, link))

	_, err := readFile(link)

	assert.ErrorIs(t, err, goyze.ErrTooLarge)
}

// TestAnOversizeDocumentIsReportedByBothEntryPoints pins that the walk and the
// named path agree, and that neither passes over it in silence — a document too
// large to analyze is exactly where an unchecked changelog hides.
func TestAnOversizeDocumentIsReportedByBothEntryPoints(t *testing.T) {
	dir := t.TempDir()
	huge := filepath.Join(dir, "huge.md")
	require.NoError(t, os.WriteFile(huge, nil, 0o600))
	require.NoError(t, os.Truncate(huge, int64(docfiles.SizeLimit)+1))

	for name, args := range map[string][]string{"walked": {dir}, "named": {huge}} {
		buf := swapStdout(t)
		require.Equal(t, 0, run(args), name)
		assert.Contains(t, buf.String(), "huge.md", "%s: reported, not dropped", name)
		assert.Contains(t, buf.String(), "too large", name)
	}
}
