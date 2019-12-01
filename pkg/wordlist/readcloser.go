package wordlist

import (
	"bufio"
	"io"
)

// FromReadCloser creates a wordlist from a readcloser
func FromReadCloser(rc io.ReadCloser) Wordlist {
	fileWordlist := &ReaderWordlist{}
	fileWordlist.handle = rc
	fileWordlist.scanner = bufio.NewScanner(fileWordlist.handle)
	return fileWordlist
}

func (fw *ReaderWordlist) Next() (string, error) {
	if !fw.scanner.Scan() {
		defer func() { _ = fw.handle.Close() }()
		if err := fw.scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return fw.scanner.Text(), nil
}
