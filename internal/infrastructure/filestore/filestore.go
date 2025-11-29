package filestore

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/h2non/filetype"
)

// FileStore saves image files to disk under a configurable base path.
type FileStore struct {
	basePath string
}

// New creates a FileStore with the given basePath.
func New(basePath string) *FileStore {
	if basePath == "" {
		basePath = "property_images"
	}
	return &FileStore{basePath: basePath}
}

// Save writes file data into basePath/{propertyID}/{filename} and returns the full path.
// It validates that the filename has an allowed image extension (png, jpg, jpeg) and also
// checks the magic bytes for basic validation.
func (fsys *FileStore) Save(propertyID int, filename string, data []byte) (string, error) {
	if propertyID == 0 {
		return "", NewErrInvalidInput("property_id", propertyID, "invalid or zero id")
	}
	if filename == "" || len(data) == 0 {
		return "", NewErrInvalidInput("file", nil, "empty filename or data")
	}

	// validate extension quickly (SaveToDir will perform final detection and may append
	// a missing extension). This avoids duplicating the content-based detection twice.
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png", ".jpg", ".jpeg":
		// ok
	default:
		return "", NewErrUnsupportedFormat(filename, ext)
	}

	dir := filepath.Join(fsys.basePath, fmt.Sprintf("%d", propertyID))
	return fsys.SaveToDir(dir, filename, data)
}

// SaveToDir writes a file into an explicitly provided directory. It performs the same
// validation as Save (extension + magic-bytes). If the provided filename has no
// extension, SaveToDir will use the detected extension and append it to the filename.
func (fsys *FileStore) SaveToDir(dir, filename string, data []byte) (string, error) {
	if dir == "" {
		return "", NewErrInvalidInput("dir", dir, "empty directory")
	}
	if filename == "" || len(data) == 0 {
		return "", NewErrInvalidInput("file", nil, "empty filename or data")
	}

	// Use filetype to validate content and detect MIME/extension
	kind, err := filetype.Match(data)
	if err != nil {
		return "", NewErrInvalidInput("file_data", nil, "unable to detect file type")
	}
	if kind == filetype.Unknown {
		return "", NewErrUnsupportedFormat(filename, "unknown")
	}

	detectedExt := strings.ToLower(kind.Extension)
	if detectedExt == "jpg" {
		detectedExt = "jpeg"
	}
	if detectedExt != "png" && detectedExt != "jpeg" {
		return "", NewErrUnsupportedFormat(filename, kind.MIME.Value)
	}

	// sanitize filename base
	base := filepath.Base(filename)
	ext := strings.ToLower(filepath.Ext(base))
	if ext == "" {
		// append detected extension if caller didn't provide one
		base = base + "." + detectedExt
		ext = "." + detectedExt
	}

	// ensure the extension matches detected type
	if ext != ".png" && ext != ".jpeg" && ext != ".jpg" {
		return "", NewErrUnsupportedFormat(filename, ext)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir failed: %w", err)
	}

	full := filepath.Join(dir, base)
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return "", NewErrStorage("write", err.Error())
	}
	return full, nil
}

// Delete removes a file at path; returns nil if file removed or if it doesn't exist.
func (fsys *FileStore) Delete(path string) error {
	if path == "" {
		return nil
	}
	err := os.Remove(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return NewErrStorage("delete", err.Error())
	}
	return nil
}

// DeletePropertyDir removes the entire directory for a given property (basePath/{propertyID}).
// This is useful to remove all files for a property in one operation.
func (fsys *FileStore) DeletePropertyDir(propertyID int) error {
	if propertyID == 0 {
		return NewErrInvalidInput("property_id", propertyID, "invalid or zero id")
	}
	dir := filepath.Join(fsys.basePath, fmt.Sprintf("%d", propertyID))
	// RemoveAll returns nil if path does not exist, which is acceptable.
	if err := os.RemoveAll(dir); err != nil {
		return NewErrStorage("remove_all", err.Error())
	}
	return nil
}

// Read reads file data from an absolute path (as stored in the DB) and returns the bytes.
func (fsys *FileStore) Read(path string) ([]byte, error) {
	if path == "" {
		return nil, NewErrInvalidInput("path", path, "empty path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, NewErrStorage("read", "file not found")
		}
		return nil, NewErrStorage("read", err.Error())
	}
	return data, nil
}
