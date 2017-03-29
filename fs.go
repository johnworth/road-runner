package main

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/cyverse-de/logcabin"
)

func copyJobFile(uuid, from, toDir string) error {
	inputReader, err := os.Open(from)
	if err != nil {
		return err
	}

	outputFilePath := path.Join(toDir, fmt.Sprintf("%s.json", uuid))
	outputWriter, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(outputWriter, inputReader); err != nil {
		return err
	}

	return nil
}

func deleteJobFile(uuid, toDir string) {
	filePath := path.Join(toDir, fmt.Sprintf("%s.json", uuid))
	if err := os.Remove(filePath); err != nil {
		logcabin.Error.Print(err)
	}
}
