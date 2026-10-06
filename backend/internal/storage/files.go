// Package storage keeps uploaded files on disk, one folder per entity:
// <dir>/<entityID>/<photoID><ext>. Callers pass server-generated names only,
// never names from a client; every name is still checked so a path can never
// escape dir.
package storage

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Files stores photo files under a root directory.
type Files struct {
	dir string
}

// NewFiles returns a store rooted at dir, creating it if needed.
func NewFiles(dir string) (*Files, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload directory: %w", err)
	}
	return &Files{dir: dir}, nil
}

// errUnsafeName guards against an empty or path-like name, which could make
// RemoveEntity delete the whole upload directory.
var errUnsafeName = errors.New("unsafe file name")

// checkName accepts only a plain, non-empty file or folder name.
func checkName(name string) error {
	if name == "" || name == "." || name == ".." || filepath.Base(name) != name {
		return fmt.Errorf("%w: %q", errUnsafeName, name)
	}
	return nil
}

// entityDir returns the folder of one entity's photos.
func (f *Files) entityDir(entityID string) (string, error) {
	if err := checkName(entityID); err != nil {
		return "", err
	}
	return filepath.Join(f.dir, entityID), nil
}

func (f *Files) path(entityID, name string) (string, error) {
	dir, err := f.entityDir(entityID)
	if err != nil {
		return "", err
	}
	if err := checkName(name); err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}

// Save writes data as entityID/name. It writes to a temporary file first so a
// reader never sees a half-written photo.
func (f *Files) Save(entityID, name string, data []byte) error {
	target, err := f.path(entityID, name)
	if err != nil {
		return err
	}
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create entity upload directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, name+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return fmt.Errorf("write photo: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("close photo: %w", err)
	}
	if err := os.Rename(tmp.Name(), target); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("store photo: %w", err)
	}
	return nil
}

// Open opens entityID/name for reading. The caller must close it.
func (f *Files) Open(entityID, name string) (*os.File, error) {
	p, err := f.path(entityID, name)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(p)
	if err != nil {
		return nil, fmt.Errorf("open photo: %w", err)
	}
	return file, nil
}

// Remove deletes entityID/name. A missing file is not an error.
func (f *Files) Remove(entityID, name string) error {
	p, err := f.path(entityID, name)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove photo: %w", err)
	}
	return nil
}

// RemoveEntity deletes every file of an entity. A missing folder is not an error.
func (f *Files) RemoveEntity(entityID string) error {
	dir, err := f.entityDir(entityID)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove entity photos: %w", err)
	}
	return nil
}
