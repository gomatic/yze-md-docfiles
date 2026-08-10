package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	errs "github.com/gomatic/go-error"
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

// TestBoundedReadRefusesAnOversizeDocumentThroughASymlink pins the bound
// against the path that defeated it. The size used to be asked for with a stat
// BEFORE the read, which describes a path rather than the bytes behind it: a
// stat that did not follow the link reported the link's own few bytes and then
// read the two gigabytes it pointed at. Discovery deliberately follows
// symlinks, so this is an ordinary document, not a contrived one — and it is
// how the 4.3 GB defect came back after being fixed twice.
func TestBoundedReadRefusesAnOversizeDocumentThroughASymlink(t *testing.T) {
	dir := t.TempDir()
	huge := filepath.Join(dir, "huge.md")
	require.NoError(t, os.WriteFile(huge, nil, 0o600))
	require.NoError(t, os.Truncate(huge, docfiles.SizeLimit+1))
	link := filepath.Join(dir, "link.md")
	require.NoError(t, os.Symlink(huge, link))

	_, err := boundedRead(entryPath(link))

	require.Error(t, err)
	assert.ErrorIs(t, err, docfiles.ErrTooLarge)
}

// TestBoundedReadReadsExactlyUpToTheLimit pins both sides of the boundary, so
// the limit cannot be moved in either direction without a failure.
func TestBoundedReadReadsExactlyUpToTheLimit(t *testing.T) {
	dir := t.TempDir()
	for name, size := range map[string]int64{"at the limit": docfiles.SizeLimit, "over it": docfiles.SizeLimit + 1} {
		at := filepath.Join(dir, name+".md")
		require.NoError(t, os.WriteFile(at, nil, 0o600))
		require.NoError(t, os.Truncate(at, size))

		data, err := boundedRead(entryPath(at))

		if size > docfiles.SizeLimit {
			assert.ErrorIs(t, err, docfiles.ErrTooLarge, name)
			continue
		}
		require.NoError(t, err, name)
		assert.Len(t, data, int(size), name)
	}
}

// TestBoundedReadSurfacesAnOpenFailure pins that a document that cannot be
// opened is an error rather than an empty document read as clean prose.
func TestBoundedReadSurfacesAnOpenFailure(t *testing.T) {
	original := openFile
	openFile = func(string) (io.ReadCloser, error) { return nil, os.ErrPermission }
	t.Cleanup(func() { openFile = original })

	_, err := boundedRead("locked.md")

	require.Error(t, err)
	assert.ErrorIs(t, err, os.ErrPermission)
}

// TestBoundedReadSurfacesAReadFailure pins the other half: a file that opens
// and then fails mid-read is reported, not truncated into a shorter document
// that happens to hold no findings.
func TestBoundedReadSurfacesAReadFailure(t *testing.T) {
	original := openFile
	openFile = func(string) (io.ReadCloser, error) { return io.NopCloser(failingReader{}), nil }
	t.Cleanup(func() { openFile = original })

	_, err := boundedRead("broken.md")

	require.Error(t, err)
	assert.ErrorIs(t, err, errReadFailed)
}

// errReadFailed is the failure a [failingReader] raises.
const errReadFailed errs.Const = "read failed"

// failingReader fails on first read, standing for a file that opens and then
// cannot be read — a disk error, a vanished network mount.
type failingReader struct{}

// Read always fails.
func (failingReader) Read([]byte) (int, error) { return 0, errReadFailed }

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
