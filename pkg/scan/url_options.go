package scan

import (
	"net/url"
	"time"

	"github.com/liamg/scout/pkg/wordlist"
)

type URLOption func(s *URLScanner)

// WithTargetURL sets the url to initiate scans from
func WithTargetURL(target url.URL) URLOption {
	return func(s *URLScanner) {
		s.targetURL = target
	}
} // target url

// WithPositiveStatusCodes provides status codes that indicate the existence of a file/directory
func WithPositiveStatusCodes(codes []int) URLOption {
	return func(s *URLScanner) {
		s.positiveStatusCodes = codes
	}
}

func WithTimeout(timeout time.Duration) URLOption {
	return func(s *URLScanner) {
		s.timeout = timeout
	}
} // http request timeout

func WithParallelism(routines int) URLOption {
	return func(s *URLScanner) {
		s.parallelism = routines
	}
} // parallel routines

func WithResultChan(c chan URLResult) URLOption {
	return func(s *URLScanner) {
		s.resultChan = c
	}
} // chan to return results on - otherwise will be returned in slice

func WithBusyChan(c chan string) URLOption {
	return func(s *URLScanner) {
		s.busyChan = c
	}
} // chan to use to update current job

func WithWordlist(w wordlist.Wordlist) URLOption {
	return func(s *URLScanner) {
		s.words = w
	}
}

// you must include the .
func WithExtensions(extensions []string) URLOption {
	return func(s *URLScanner) {
		s.extensions = extensions
	}
}
func WithFilename(filename string) URLOption {
	return func(s *URLScanner) {
		s.filename = filename
	}
}
func WithSpidering(spider bool) URLOption {
	return func(s *URLScanner) {
		s.enableSpidering = spider
	}
}

func WithSkipSSLVerification(skipSSL bool) URLOption {
	return func(s *URLScanner) {
		s.skipSSLVerification = skipSSL
	}
}
func WithBackupExtensions(backupExtensions []string) URLOption {
	return func(s *URLScanner) {
		s.backupExtensions = backupExtensions
	}
}

func WithExtraHeaders(headers []string) URLOption {
	return func(s *URLScanner) {
		s.extraHeaders = append(s.extraHeaders, headers...)
	}
}

type URLResult struct {
	URL        url.URL
	StatusCode int
}
