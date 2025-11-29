package filestore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var pngSig = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}

func TestSaveToDir_WithAndWithoutExtension(t *testing.T) {
	tmp := t.TempDir()
	fsys := New(tmp)

	dir := filepath.Join(tmp, "p123")

	p1, err := fsys.SaveToDir(dir, "1.png", pngSig)
	require.NoError(t, err)
	require.FileExists(t, p1)

	data, err := os.ReadFile(p1)
	require.NoError(t, err)
	require.Equal(t, pngSig, data)

	p2, err := fsys.SaveToDir(dir, "2", pngSig)
	require.NoError(t, err)
	require.FileExists(t, p2)
	require.Equal(t, ".png", filepath.Ext(p2))
}

func TestRead_Delete_DeletePropertyDir(t *testing.T) {
	tmp := t.TempDir()
	fsys := New(tmp)

	propDir := filepath.Join(tmp, "555")
	if err := os.MkdirAll(propDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	fpath := filepath.Join(propDir, "1.png")
	require.NoError(t, os.WriteFile(fpath, pngSig, 0o644))

	got, err := fsys.Read(fpath)
	require.NoError(t, err)
	require.Equal(t, pngSig, got)

	require.NoError(t, fsys.Delete(fpath))
	require.NoFileExists(t, fpath)

	require.NoError(t, fsys.Delete(""))

	propDir2 := filepath.Join(tmp, "777")
	require.NoError(t, os.MkdirAll(propDir2, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(propDir2, "x.png"), pngSig, 0o644))

	fsys2 := New(tmp)
	require.NoError(t, fsys2.DeletePropertyDir(777))
	require.NoDirExists(t, filepath.Join(tmp, "777"))

	err = fsys2.DeletePropertyDir(0)
	require.Error(t, err)
	var inv ErrInvalidInput
	require.ErrorAs(t, err, &inv)
}
