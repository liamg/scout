package wordlist

import (
	"io"
	"io/ioutil"
)

// FromReader creates a wordlist from a reader
func FromReader(r io.Reader) Wordlist {
	return FromReadCloser(ioutil.NopCloser(r))
}
