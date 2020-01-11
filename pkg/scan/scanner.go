package scan

import (
	"crypto/tls"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/avast/retry-go"
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

	client := &http.Client{
		Timeout: opt.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	if opt.SkipSSLVerification {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		client.Transport = http.DefaultTransport
	}

	return &Scanner{
		options: opt,
		client:  client,
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

	var extraWork []string

	go func() {
		for result := range results {
			if result.ExtraWork != nil {
				extraWork = append(extraWork, result.ExtraWork...)
			}
			if result.SupplementaryOnly {
				continue
			}
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

	logrus.Debug("Supplementing results...")

	for _, work := range extraWork {
		if result := scanner.checkURL(work); result != nil {
			foundURLs = append(foundURLs, result.URL)
		}
	}

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

	var code int
	var location string
	if err := retry.Do(func() error {
		resp, err := scanner.client.Get(uri)
		if err != nil {
			return nil
		}
		_, _ = io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
		code = resp.StatusCode
		location = resp.Header.Get("Location")
		return nil
	}, retry.Attempts(10), retry.DelayType(retry.BackOffDelay)); err != nil {
		return nil
	}

	var extraWork []string

	if location != "" {
		if !strings.Contains(location, "://") {
			if parsed, err := url.Parse(uri); err == nil {
				if relative, err := url.Parse(location); err == nil {
					extraWork = append(extraWork, parsed.ResolveReference(relative).String())
				}
			}
		} else {
			extraWork = append(extraWork, location)
		}
	}

	for _, status := range scanner.options.PositiveStatusCodes {
		if status == code {
			parsedURL, err := url.Parse(uri)
			if err != nil {
				return nil
			}

			return &Result{
				StatusCode: code,
				URL:        *parsedURL,
				ExtraWork:  extraWork,
			}
		}
	}

	if len(extraWork) > 0 {
		return &Result{
			SupplementaryOnly: true,
			ExtraWork:         extraWork,
		}
	}

	return nil
}
