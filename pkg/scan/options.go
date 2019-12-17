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
	PositiveStatusCodes []int         // status codes that indicate the existance of a file/directory
	Timeout             time.Duration // http request timeout
	Parallelism         int           // parallel routines
	ResultChan          chan Result   // chan to return results on - otherwise will be returned in slice
	BusyChan            chan string   // chan to use to update current job
	Wordlist            wordlist.Wordlist
	Extensions          []string
	Filename            string
	SkipSSLVerification bool
}

type Result struct {
	URL        url.URL
	StatusCode int
}

var DefaultOptions = Options{
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
	Parallelism: 10,
	Extensions:  []string{"php", "htm", "html"},
}

func (opt *Options) Inherit() {
	if len(opt.PositiveStatusCodes) == 0 {
		opt.PositiveStatusCodes = DefaultOptions.PositiveStatusCodes
	}
	if opt.Timeout == 0 {
		opt.Timeout = DefaultOptions.Timeout
	}
	if opt.Parallelism == 0 {
		opt.Parallelism = DefaultOptions.Parallelism
	}
	if opt.Wordlist == nil {
		wordlistBytes, err := data.Asset("assets/wordlist.txt")
		if err != nil {
			wordlistBytes = []byte{}
		}
		opt.Wordlist = wordlist.FromReader(bytes.NewReader(wordlistBytes))
	}
	if len(opt.Extensions) == 0 {
		opt.Extensions = DefaultOptions.Extensions
	}
}
