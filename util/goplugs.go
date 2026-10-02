package util

import (
	"io"
	"os"
)

func MustOpen(filename string) io.ReadCloser {
	open, _ := os.Open(filename)
	return open
}

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
