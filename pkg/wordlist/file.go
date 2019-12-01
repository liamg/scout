package wordlist

import (
	"bufio"
	"io"
	"os"
)

type ReaderWordlist struct {
	handle  io.ReadCloser
	scanner *bufio.Scanner
}

// FromFile creates a WordList from a file on disk. The file should includes words separated by new lines.
func FromFile(path string) (Wordlist, error) {
	fileWordlist := &ReaderWordlist{}
	var err error
	fileWordlist.handle, err = os.Open(path)
	if err != nil {
		return nil, err
	}
	fileWordlist.scanner = bufio.NewScanner(fileWordlist.handle)
	return fileWordlist, err
}
