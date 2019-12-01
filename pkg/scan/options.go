package scan

import (
	"bytes"
	"net/http"
	"net/url"
	"time"

	"github.com/liamg/scout/internal/app/scout/data"
	"github.com/liamg/scout/pkg/wordlist"
)

type Options struct {
	TargetURL           url.URL       // target url
	Recursive           bool          // scan path recursively. e.g. when /secret is discovered, whether we keep looking for /secret/*
	PositiveStatusCodes []int         // status codes that indicate the existance of a file/directory
	Timeout             time.Duration // http request timeout
	Parallelism         int           // parallel routines
	ResultChan          chan url.URL  // chan to return results on - otherwise will be returned in slice
	Wordlist            wordlist.Wordlist
	Extensions          []string
}

var defaultOptions = Options{
	Recursive: false,
	PositiveStatusCodes: []int{
		http.StatusOK,
		http.StatusFound,
		http.StatusMovedPermanently,
		http.StatusBadRequest,
		http.StatusForbidden,
		http.StatusInternalServerError,
		http.StatusMethodNotAllowed,
		http.StatusNoContent,
		http.StatusUnauthorized,
	},
	Timeout:     time.Second * 5,
	Parallelism: 5,
	Extensions:  []string{".php", ".asp", ".htm", ".html", ".txt"},
}

func (opt *Options) inherit() {
	if opt.PositiveStatusCodes == nil {
		opt.PositiveStatusCodes = defaultOptions.PositiveStatusCodes
	}
	if opt.Timeout == 0 {
		opt.Timeout = defaultOptions.Timeout
	}
	if opt.Parallelism == 0 {
		opt.Parallelism = defaultOptions.Parallelism
	}
	if opt.Wordlist == nil {
		wordlistBytes, err := data.Asset("assets/wordlist.txt")
		if err != nil {
			wordlistBytes = []byte{}
		}
		opt.Wordlist = wordlist.FromReader(bytes.NewReader(wordlistBytes))
	}
	if opt.Extensions == nil {
		opt.Extensions = defaultOptions.Extensions
	}
}
