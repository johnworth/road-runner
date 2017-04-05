package fs

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/pkg/errors"
)

// adapted from https://talks.golang.org/2012/10things.slide#8

// FS is a FileSystem that interacts with the local filesystem.
var FS FileSystem = localFS{}

// FileSystem defines the filesystem operations used in this file of road-runner
type FileSystem interface {
	Open(path string) (File, error)
	Create(path string) (File, error)
	Remove(path string) error
}

// File defines the operations supported by File objects.
type File interface {
	io.Closer
	io.Reader
	io.Writer
}

// localFS implements FileSystem using the local filesystem.
type localFS struct{}

func (localFS) Open(path string) (File, error)   { return os.Open(path) }
func (localFS) Create(path string) (File, error) { return os.Create(path) }
func (localFS) Remove(path string) error         { return os.Remove(path) }

// CopyJobFile copies the contents of from to a file called <uuid>.json inside
// the directory specified by toDir.
func CopyJobFile(fs FileSystem, uuid, from, toDir string) error {
	inputReader, err := fs.Open(from)
	if err != nil {
		return errors.Wrapf(err, "failed to open %s", from)
	}
	defer inputReader.Close()
	outputFilePath := path.Join(toDir, fmt.Sprintf("%s.json", uuid))
	outputWriter, err := fs.Create(outputFilePath)
	if err != nil {
		return errors.Wrapf(err, "failed to write to %s", outputFilePath)
	}
	defer outputWriter.Close()
	if _, err := io.Copy(outputWriter, inputReader); err != nil {
		return errors.Wrapf(err, "failed to copy contents of %s to %s", from, toDir)
	}
	return nil
}

// DeleteJobFile deletes the file <uuid>.json from the directory specified by
// toDir.
func DeleteJobFile(fs FileSystem, uuid, toDir string) error {
	filePath := path.Join(toDir, fmt.Sprintf("%s.json", uuid))
	if err := fs.Remove(filePath); err != nil {
		return errors.Wrapf(err, "failed to remove %s", filePath)
	}
	return nil
}
