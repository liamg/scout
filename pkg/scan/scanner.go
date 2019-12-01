package scan

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
)

type Scanner struct {
	client  *http.Client
	options *Options
}

func NewScanner(opt *Options) *Scanner {

	if opt == nil {
		opt = &defaultOptions
	}

	opt.inherit()

	return &Scanner{
		options: opt,
		client: &http.Client{
			Timeout: opt.Timeout,
		},
	}
}

func (scanner *Scanner) Scan() ([]url.URL, error) {

	// todo check url is well formed, check hostname exists etc.

	jobs := make(chan string, scanner.options.Parallelism)
	results := make(chan string, scanner.options.Parallelism)

	wg := sync.WaitGroup{}

	for i := 0; i < scanner.options.Parallelism; i++ {
		wg.Add(1)
		go func() {
			scanner.worker(jobs, results)
			wg.Done()
		}()
	}

	// add urls to scan...
	prefix := scanner.options.TargetURL
	for {
		if word, err := scanner.options.Wordlist.Next(); err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		} else {
			segment, err := url.Parse(word)
			if err != nil {
				continue
			}
			uri := prefix.ResolveReference(segment).String()
			jobs <- uri
			for _, ext := range scanner.options.Extensions {
				jobs <- uri + ext
			}
		}
	}
	close(jobs)

	waitChan := make(chan struct{})
	var foundURLs []url.URL

	go func() {
		for result := range results {
			parsedURL, err := url.Parse(result)
			if err == nil {
				if scanner.options.ResultChan != nil {
					scanner.options.ResultChan <- *parsedURL
				} else {
					foundURLs = append(foundURLs, *parsedURL)
				}
			}
		}
		close(waitChan)
	}()

	wg.Wait()
	close(results)
	<-waitChan

	return foundURLs, nil
}

func (scanner *Scanner) worker(jobs <-chan string, results chan<- string) {
	for j := range jobs {
		if scanner.checkURL(j) {
			results <- j
		}
	}
}

// hit a url - is it one of certain response codes? leave connections open!
func (scanner *Scanner) checkURL(url string) bool {

	resp, err := scanner.client.Get(url)
	if err != nil {
		return false
	}

	_, _ = io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	for _, status := range scanner.options.PositiveStatusCodes {
		if status == resp.StatusCode {
			return true
		}
	}

	return false
}

// strategy to generate urls? directories + what file extensions?

// word list? built in or external?

// recursive?
