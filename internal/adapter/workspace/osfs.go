package workspace

import (
	"os"
	"path/filepath"
)

type OSFileSystem struct{}

func (OSFileSystem) ReadDir(name string) ([]os.DirEntry, error) {
	return os.ReadDir(name)
}

func (OSFileSystem) EvalSymlinks(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}

func (OSFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}
