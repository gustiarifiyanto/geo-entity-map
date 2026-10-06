package storage

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func newFiles(t *testing.T) (*Files, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "uploads")
	f, err := NewFiles(dir)
	if err != nil {
		t.Fatalf("NewFiles: %v", err)
	}
	return f, dir
}

func TestSaveOpenRemove(t *testing.T) {
	f, dir := newFiles(t)
	if err := f.Save("e1", "p1.png", []byte("hello")); err != nil {
		t.Fatalf("Save: %v", err)
	}

	file, err := f.Open("e1", "p1.png")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got, err := io.ReadAll(file)
	file.Close()
	if err != nil || string(got) != "hello" {
		t.Fatalf("read back %q, %v; want \"hello\"", got, err)
	}

	// No temp files are left behind.
	entries, err := os.ReadDir(filepath.Join(dir, "e1"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("entity folder entries = %v, %v; want only p1.png", entries, err)
	}

	if err := f.Remove("e1", "p1.png"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := f.Remove("e1", "p1.png"); err != nil {
		t.Fatalf("second Remove of a missing file: %v", err)
	}
}

func TestRemoveEntity(t *testing.T) {
	f, dir := newFiles(t)
	for _, e := range []string{"e1", "e2"} {
		if err := f.Save(e, "p.jpg", []byte("x")); err != nil {
			t.Fatalf("Save %s: %v", e, err)
		}
	}
	if err := f.RemoveEntity("e1"); err != nil {
		t.Fatalf("RemoveEntity: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "e1")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("e1 folder still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "e2", "p.jpg")); err != nil {
		t.Errorf("other entity's photo was removed: %v", err)
	}
	if err := f.RemoveEntity("never-had-photos"); err != nil {
		t.Errorf("RemoveEntity on a missing folder: %v", err)
	}
}

func TestRejectsUnsafeNames(t *testing.T) {
	f, dir := newFiles(t)
	if err := f.Save("e1", "keep.jpg", []byte("x")); err != nil {
		t.Fatalf("Save: %v", err)
	}

	for _, name := range []string{"", ".", "..", "../e1", "a/b"} {
		if err := f.RemoveEntity(name); !errors.Is(err, errUnsafeName) {
			t.Errorf("RemoveEntity(%q) = %v, want errUnsafeName", name, err)
		}
		if err := f.Save("e1", name, []byte("x")); !errors.Is(err, errUnsafeName) {
			t.Errorf("Save(e1, %q) = %v, want errUnsafeName", name, err)
		}
	}
	// Nothing was deleted by the rejected calls.
	if _, err := os.Stat(filepath.Join(dir, "e1", "keep.jpg")); err != nil {
		t.Fatalf("existing photo is gone: %v", err)
	}
}
