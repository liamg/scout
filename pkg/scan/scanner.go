package scan

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

type Scanner struct {
	client  *http.Client
	options *Options
}

func NewScanner(opt *Options) *Scanner {

	if opt == nil {
		opt = &DefaultOptions
	}

	opt.Inherit()

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
	results := make(chan Result, scanner.options.Parallelism)

	wg := sync.WaitGroup{}

	logrus.Debug("Starting workers...")

	for i := 0; i < scanner.options.Parallelism; i++ {
		wg.Add(1)
		go func() {
			scanner.worker(jobs, results)
			wg.Done()
		}()
	}

	logrus.Debugf("Started %d workers!", scanner.options.Parallelism)

	logrus.Debug("Starting results gatherer...")

	waitChan := make(chan struct{})
	var foundURLs []url.URL

	go func() {
		for result := range results {
			if scanner.options.ResultChan != nil {
				scanner.options.ResultChan <- result
			}
			foundURLs = append(foundURLs, result.URL)
		}
		if scanner.options.ResultChan != nil {
			close(scanner.options.ResultChan)
		}
		close(waitChan)
	}()

	logrus.Debug("Adding jobs...")

	// add urls to scan...
	prefix := scanner.options.TargetURL.String()
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}
	for {
		if word, err := scanner.options.Wordlist.Next(); err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		} else {
			if word == "" {
				continue
			}
			uri := prefix + word
			if scanner.options.Filename != "" {
				jobs <- uri + "/" + scanner.options.Filename
			} else {
				jobs <- uri
				for _, ext := range scanner.options.Extensions {
					jobs <- uri + "." + ext
				}
			}
		}
	}
	close(jobs)

	logrus.Debug("Waiting for workers to complete...")

	wg.Wait()
	close(results)

	if scanner.options.BusyChan != nil {
		close(scanner.options.BusyChan)
	}

	logrus.Debug("Waiting for results...")

	<-waitChan

	logrus.Debug("Complete!")

	return foundURLs, nil
}

func (scanner *Scanner) worker(jobs <-chan string, results chan<- Result) {
	for j := range jobs {
		if result := scanner.checkURL(j); result != nil {
			results <- *result
		}
	}
}

// hit a url - is it one of certain response codes? leave connections open!
func (scanner *Scanner) checkURL(uri string) *Result {

	if scanner.options.BusyChan != nil {
		scanner.options.BusyChan <- uri
	}

	resp, err := scanner.client.Get(uri)
	if err != nil {
		return nil
	}

	_, _ = io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	//fmt.Printf("[%d] %s\n", resp.StatusCode, url)

	for _, status := range scanner.options.PositiveStatusCodes {
		if status == resp.StatusCode {
			parsedURL, err := url.Parse(uri)
			if err != nil {
				return nil
			}

			return &Result{
				StatusCode: resp.StatusCode,
				URL:        *parsedURL,
			}
		}
	}

	return nil
}

// strategy to generate urls? directories + what file extensions?

// word list? built in or external?

// recursive?
