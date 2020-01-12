package scan

import (
	"bytes"
	"net/http"
	"net/url"
	"time"

	"github.com/liamg/scout/internal/app/scout/data"
	"github.com/liamg/scout/pkg/wordlist"
)

type URLOptions struct {
	TargetURL           url.URL        // target url
	PositiveStatusCodes []int          // status codes that indicate the existance of a file/directory
	Timeout             time.Duration  // http request timeout
	Parallelism         int            // parallel routines
	ResultChan          chan URLResult // chan to return results on - otherwise will be returned in slice
	BusyChan            chan string    // chan to use to update current job
	Wordlist            wordlist.Wordlist
	Extensions          []string
	Filename            string
	SkipSSLVerification bool
	BackupExtensions    []string
}

type URLResult struct {
	URL               url.URL
	StatusCode        int
	ExtraWork         []string
	SupplementaryOnly bool
}

var DefaultURLOptions = URLOptions{
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
		http.StatusMovedPermanently,
		http.StatusFound,
	},
	Timeout:          time.Second * 5,
	Parallelism:      10,
	Extensions:       []string{"php", "htm", "html"},
	BackupExtensions: []string{"~", ".bak", ".BAK", ".old", ".backup", ".txt", ".OLD", ".BACKUP", "1", "2", "_"},
}

func (opt *URLOptions) Inherit() {
	if len(opt.PositiveStatusCodes) == 0 {
		opt.PositiveStatusCodes = DefaultURLOptions.PositiveStatusCodes
	}
	if opt.Timeout == 0 {
		opt.Timeout = DefaultURLOptions.Timeout
	}
	if opt.Parallelism == 0 {
		opt.Parallelism = DefaultURLOptions.Parallelism
	}
	if opt.Wordlist == nil {
		wordlistBytes, err := data.Asset("assets/wordlist.txt")
		if err != nil {
			wordlistBytes = []byte{}
		}
		opt.Wordlist = wordlist.FromReader(bytes.NewReader(wordlistBytes))
	}
	if len(opt.Extensions) == 0 {
		opt.Extensions = DefaultURLOptions.Extensions
	}
	if len(opt.BackupExtensions) == 0 {
		opt.BackupExtensions = DefaultURLOptions.BackupExtensions
	}
}
