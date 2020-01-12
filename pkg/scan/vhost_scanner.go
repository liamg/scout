package scan

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/avast/retry-go"
	"github.com/sirupsen/logrus"
)

type VHOSTScanner struct {
	client  *http.Client
	options *VHOSTOptions
}

func NewVHOSTScanner(opt *VHOSTOptions) *VHOSTScanner {

	if opt == nil {
		opt = &DefaultVHOSTOptions
	}

	opt.Inherit()

	client := &http.Client{
		Timeout: opt.Timeout,
	}

	if opt.SkipSSLVerification {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		client.Transport = http.DefaultTransport
	}

	return &VHOSTScanner{
		options: opt,
		client:  client,
	}
}

func (scanner *VHOSTScanner) Scan() ([]string, error) {

	jobs := make(chan string, scanner.options.Parallelism)
	results := make(chan URLResult, scanner.options.Parallelism)

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

	logrus.Debug("Waiting for results...")

	<-waitChan

	if scanner.options.BusyChan != nil {
		close(scanner.options.BusyChan)
	}

	logrus.Debug("Complete!")

	return foundURLs, nil
}

func (scanner *VHOSTScanner) worker(jobs <-chan string, results chan<- URLResult) {
	for j := range jobs {
		if result := scanner.checkVHOST(j); result != nil {
			results <- *result
		}
	}
}

// hit a url - is it one of certain response codes? leave connections open!
func (scanner *VHOSTScanner) checkVHOST(vhost string) *URLResult {

	if scanner.options.BusyChan != nil {
		scanner.options.BusyChan <- vhost
	}

	var code int

	if err := retry.Do(func() error {
		resp, err := scanner.client.Get(fmt.Sprintf())
		if err != nil {
			return nil
		}
		_, _ = io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
		code = resp.StatusCode
		return nil
	}, retry.Attempts(10), retry.DelayType(retry.BackOffDelay)); err != nil {
		return nil
	}

	for _, status := range scanner.options.PositiveStatusCodes {
		if status == code {
			parsedURL, err := url.Parse(uri)
			if err != nil {
				return nil
			}

			return &VHOSTResult{
				StatusCode: code,
				URL:        *parsedURL,
			}
		}
	}

	return nil
}
