package internal

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestFileClientFullPath(t *testing.T) {
	tmp := t.TempDir()
	c := &FileClient{BaseDir: tmp}

	p, err := c.fullPath("sub", "file.txt")
	if err != nil {
		t.Fatalf("fullPath returned error: %v", err)
	}

	expected := filepath.Join(tmp, "sub", "file.txt")
	expected, _ = filepath.Abs(expected)
	if p != expected {
		t.Fatalf("expected %s, got %s", expected, p)
	}
}

func TestFileClientFullPathTraversal(t *testing.T) {
	tmp := t.TempDir()
	c := &FileClient{BaseDir: tmp}

	if _, err := c.fullPath("..", "evil.txt"); err == nil {
		t.Fatalf("expected error for path traversal")
	}
}

func TestWriteReadDelete(t *testing.T) {
	tmp := t.TempDir()
	c := &FileClient{BaseDir: tmp}

	filePath := filepath.Join(tmp, "dir", "test.txt")
	data := "hello"
	if err := c.WriteFile(filePath, data); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	read, err := c.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if read != data {
		t.Fatalf("expected %q, got %q", data, read)
	}

	if err := c.Delete(filePath); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("file still exists after delete")
	}

	// Delete again should not error
	if err := c.Delete(filePath); err != nil {
		t.Fatalf("Delete on missing file failed: %v", err)
	}
}

func TestCreateZipFile(t *testing.T) {
	tmp := t.TempDir()
	c := &FileClient{BaseDir: tmp}

	srcPath := filepath.Join(tmp, "source.txt")
	if err := os.WriteFile(srcPath, []byte("content"), 0o644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	zipPath := filepath.Join(tmp, "out", "archive.zip")
	if err := c.CreateZipFile(zipPath, srcPath, "inside.txt"); err != nil {
		t.Fatalf("CreateZipFile failed: %v", err)
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	defer r.Close()
	if len(r.File) != 1 {
		t.Fatalf("expected 1 file in zip, got %d", len(r.File))
	}
	zf := r.File[0]
	if zf.Name != "inside.txt" {
		t.Fatalf("expected file name inside zip to be inside.txt, got %s", zf.Name)
	}
	rc, err := zf.Open()
	if err != nil {
		t.Fatalf("failed to open zipped file: %v", err)
	}
	defer rc.Close()
	bytes, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("failed to read zipped content: %v", err)
	}
	if string(bytes) != "content" {
		t.Fatalf("unexpected zip content: %s", string(bytes))
	}
}
