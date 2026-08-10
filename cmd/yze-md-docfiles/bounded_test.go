package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	docfiles "github.com/gomatic/yze-md-docfiles"
)

// TestBoundedReadRefusesWithoutReading pins the bound at the single place every
// path is read. Guarding only the walk was no bound at all for the other half:
// a 2 GiB NAMED document cost 4.3 GB resident — its own size to read and again
// to convert — while the same file through the walk cost six megabytes.
func TestBoundedReadRefusesWithoutReading(t *testing.T) {
	dir := t.TempDir()
	huge := filepath.Join(dir, "huge.md")
	require.NoError(t, os.WriteFile(huge, nil, 0o600))
	require.NoError(t, os.Truncate(huge, docfiles.SizeLimit+1))

	_, err := boundedRead(entryPath(huge))

	require.Error(t, err)
	assert.ErrorIs(t, err, docfiles.ErrTooLarge)

	small := writeDoc(t, dir, "small.md", banned)
	data, err := boundedRead(entryPath(small))
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

// TestBoundedReadSurfacesAStatFailure pins that a document whose size cannot be
// determined is an error rather than an unbounded read.
func TestBoundedReadSurfacesAStatFailure(t *testing.T) {
	original := statPath
	statPath = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	t.Cleanup(func() { statPath = original })

	_, err := boundedRead("absent.md")

	require.Error(t, err)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

// TestAnOversizeDocumentIsReportedByBothEntryPoints pins that the walk and the
// named path agree, and that neither passes over it in silence.
func TestAnOversizeDocumentIsReportedByBothEntryPoints(t *testing.T) {
	dir := t.TempDir()
	huge := filepath.Join(dir, "huge.md")
	require.NoError(t, os.WriteFile(huge, nil, 0o600))
	require.NoError(t, os.Truncate(huge, docfiles.SizeLimit+1))

	for name, args := range map[string][]string{"walked": {dir}, "named": {huge}} {
		buf := swapStdout(t)
		require.Equal(t, 0, run(args), name)
		assert.Contains(t, buf.String(), "huge.md", "%s: reported, not dropped", name)
		assert.Contains(t, buf.String(), "too large", name)
	}
}
