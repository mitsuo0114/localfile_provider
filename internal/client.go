package internal

import (
	"archive/zip"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// FileClient encapsulates file system operations relative to a base
// directory.  This struct is passed to resources and data sources to
// simplify path handling and ensure all file operations are scoped
// within the configured base directory.
type FileClient struct {
	BaseDir string
}

// fullPath constructs an absolute path for a given location and name
// within the base directory.  It cleans the path and ensures it does
// not escape the base directory.  If the resulting path is outside
// the base directory, an error is returned.
func (c *FileClient) fullPath(location, name string) (string, error) {
	// Join the segments and clean the result
	p := filepath.Join(c.BaseDir, location, name)
	full := filepath.Clean(p)
	// Prevent directory traversal by ensuring the final path has the
	// base directory as a prefix.  filepath.Abs resolves symbolic
	// links and normalizes the path.
	baseAbs, err := filepath.Abs(c.BaseDir)
	if err != nil {
		return "", err
	}
	fullAbs, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	// Ensure the absolute path starts with the base directory
	if len(fullAbs) < len(baseAbs) || fullAbs[:len(baseAbs)] != baseAbs {
		return "", errors.New("path escapes base directory")
	}
	return fullAbs, nil
}

// WriteFile writes the provided data to the specified path.  It
// creates parent directories as needed and overwrites any existing
// file.
func (c *FileClient) WriteFile(path string, data string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(data), 0o644)
}

// ReadFile reads and returns the contents of the specified file.
func (c *FileClient) ReadFile(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Delete removes the specified file.  It does not remove parent
// directories.  If the file does not exist, no error is returned.
func (c *FileClient) Delete(path string) error {
	// Use Remove; it will return nil if the file doesn't exist
	err := os.Remove(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// CreateZipFile creates a zip archive at zipPath containing the
// file at srcPath.  The file will be stored in the archive using
// nameInZip.  Any existing zip will be overwritten.  Parent
// directories of zipPath are created as needed.
func (c *FileClient) CreateZipFile(zipPath string, srcPath string, nameInZip string) error {
	dir := filepath.Dir(zipPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// Create the zip file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()
	zw := zip.NewWriter(zipFile)
	defer zw.Close()
	// Open source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	// Create zip header
	hdr := &zip.FileHeader{Name: nameInZip, Method: zip.Deflate}
	hdr.SetMode(0o644)
	writer, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	// Copy contents
	if _, err := io.Copy(writer, srcFile); err != nil {
		return err
	}
	return nil
}
