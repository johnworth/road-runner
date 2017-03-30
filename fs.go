package main

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/pkg/errors"
)

func copyJobFile(uuid, from, toDir string) error {
	inputReader, err := os.Open(from)
	if err != nil {
		return errors.Wrapf(err, "failed to open %s", from)
	}
	outputFilePath := path.Join(toDir, fmt.Sprintf("%s.json", uuid))
	outputWriter, err := os.Create(outputFilePath)
	if err != nil {
		return errors.Wrapf(err, "failed to write to %s", outputFilePath)
	}

	if _, err := io.Copy(outputWriter, inputReader); err != nil {
		return errors.Wrapf(err, "failed to copy contents of %s to %s", from, toDir)
	}

	return nil
}

func deleteJobFile(uuid, toDir string) error {
	filePath := path.Join(toDir, fmt.Sprintf("%s.json", uuid))
	if err := os.Remove(filePath); err != nil {
		return errors.Wrapf(err, "failed to remove %s", filePath)
	}
	return nil
}
