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

type FileStore struct {
	basePath string
}

func New(basePath string) *FileStore {
	if basePath == "" {
		basePath = "property_images"
	}
	return &FileStore{basePath: basePath}
}

func (fsys *FileStore) Save(propertyID int, filename string, data []byte) (string, error) {
	if propertyID == 0 {
		return "", NewErrInvalidInput("property_id", propertyID, "invalid or zero id")
	}
	if filename == "" || len(data) == 0 {
		return "", NewErrInvalidInput("file", nil, "empty filename or data")
	}

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png", ".jpg", ".jpeg":

	default:
		return "", NewErrUnsupportedFormat(filename, ext)
	}

	dir := filepath.Join(fsys.basePath, fmt.Sprintf("%d", propertyID))
	return fsys.SaveToDir(dir, filename, data)
}

func (fsys *FileStore) SaveToDir(dir, filename string, data []byte) (string, error) {
	if dir == "" {
		return "", NewErrInvalidInput("dir", dir, "empty directory")
	}
	if filename == "" || len(data) == 0 {
		return "", NewErrInvalidInput("file", nil, "empty filename or data")
	}

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

	base := filepath.Base(filename)
	ext := strings.ToLower(filepath.Ext(base))
	if ext == "" {

		base = base + "." + detectedExt
		ext = "." + detectedExt
	}

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

func (fsys *FileStore) DeletePropertyDir(propertyID int) error {
	if propertyID == 0 {
		return NewErrInvalidInput("property_id", propertyID, "invalid or zero id")
	}
	dir := filepath.Join(fsys.basePath, fmt.Sprintf("%d", propertyID))

	if err := os.RemoveAll(dir); err != nil {
		return NewErrStorage("remove_all", err.Error())
	}
	return nil
}

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
