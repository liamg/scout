package scan

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/liamg/scout/internal/app/scout/data"

	"github.com/liamg/scout/pkg/wordlist"

	"github.com/avast/retry-go"
	"github.com/sirupsen/logrus"
)

type URLScanner struct {
	client              *http.Client
	jobChan             chan string
	targetURL           url.URL        // target url
	positiveStatusCodes []int          // status codes that indicate the existance of a file/directory
	timeout             time.Duration  // http request timeout
	parallelism         int            // parallel routines
	resultChan          chan URLResult // chan to return results on - otherwise will be returned in slice
	busyChan            chan string    // chan to use to update current job
	words               wordlist.Wordlist
	extensions          []string
	filename            string
	skipSSLVerification bool
	backupExtensions    []string
	extraHeader         string
	enableSpidering     bool
	checked             map[string]struct{}
	checkMutex          sync.Mutex
	queueChan           chan string
	jobsLoaded          int32
}

func NewURLScanner(options ...URLOption) *URLScanner {

	scanner := &URLScanner{
		checked: make(map[string]struct{}),
		positiveStatusCodes: []int{
			http.StatusOK,
			http.StatusBadRequest,
			http.StatusInternalServerError,
			http.StatusMethodNotAllowed,
			http.StatusNoContent,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusFound,
			http.StatusMovedPermanently,
		},
		timeout:          time.Second * 5,
		parallelism:      10,
		extensions:       []string{"php", "htm", "html", "txt"},
		backupExtensions: []string{"~", ".bak", ".BAK", ".old", ".backup", ".txt", ".OLD", ".BACKUP", "1", "2", "_", ".1", ".2"},
		enableSpidering:  false,
		queueChan:        make(chan string, 0x10000),
	}

	for _, option := range options {
		option(scanner)
	}

	if scanner.words == nil {
		wordlistBytes, err := data.Asset("assets/wordlist.txt")
		if err != nil {
			wordlistBytes = []byte{}
		}
		scanner.words = wordlist.FromReader(bytes.NewReader(wordlistBytes))
	}

	scanner.client = &http.Client{
		Timeout: scanner.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	if scanner.skipSSLVerification {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		scanner.client.Transport = http.DefaultTransport
	}

	return scanner
}

func (scanner *URLScanner) Scan() ([]url.URL, error) {

	atomic.StoreInt32(&(scanner.jobsLoaded), 0)

	scanner.jobChan = make(chan string, scanner.parallelism)
	results := make(chan URLResult, scanner.parallelism)

	wg := sync.WaitGroup{}

	logrus.Debug("Starting workers...")

	for i := 0; i < scanner.parallelism; i++ {
		wg.Add(1)
		go func() {
			scanner.worker(results)
			wg.Done()
		}()
	}

	logrus.Debugf("Started %d workers!", scanner.parallelism)

	logrus.Debug("Starting results gatherer...")

	waitChan := make(chan struct{})
	var foundURLs []url.URL

	go func() {
		for result := range results {
			if scanner.resultChan != nil {
				scanner.resultChan <- result
			}
			foundURLs = append(foundURLs, result.URL)
		}
		if scanner.resultChan != nil {
			close(scanner.resultChan)
		}
		close(waitChan)
	}()

	logrus.Debug("Adding jobs...")

	// add urls to scan...
	prefix := scanner.targetURL.String()
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}
	for {
		if word, err := scanner.words.Next(); err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		} else {
			if word == "" {
				continue
			}
			uri := prefix + word
			if scanner.filename != "" {
				scanner.jobChan <- uri + "/" + scanner.filename
			} else {
				scanner.jobChan <- uri
				for _, ext := range scanner.extensions {
					scanner.jobChan <- uri + "." + ext
				}
			}
		}
	}

	atomic.StoreInt32(&(scanner.jobsLoaded), 1)

	logrus.Debug("Waiting for workers to complete...")

	wg.Wait()
	close(scanner.jobChan)
	close(results)

	logrus.Debug("Waiting for results...")

	<-waitChan

	if scanner.busyChan != nil {
		close(scanner.busyChan)
	}

	logrus.Debug("Complete!")

	return foundURLs, nil
}

func (scanner *URLScanner) worker(results chan<- URLResult) {

	for {
		select {
		case job := <-scanner.jobChan:
			if result := scanner.checkURL(job); result != nil {
				results <- *result
			}
		EXTRA:
			for {
				select {
				case extra := <-scanner.queueChan:
					if result := scanner.checkURL(extra); result != nil {
						results <- *result
					}
				default:
					break EXTRA
				}
			}
		default:
			if atomic.LoadInt32(&scanner.jobsLoaded) > 0 {
				return
			}

			time.Sleep(time.Millisecond * 100)
		}
	}
}

func (scanner *URLScanner) queue(uri string) {
	scanner.queueChan <- uri
}

func (scanner *URLScanner) visited(uri string) bool {
	scanner.checkMutex.Lock()
	defer scanner.checkMutex.Unlock()
	if _, ok := scanner.checked[uri]; ok {
		return true
	}
	scanner.checked[uri] = struct{}{}
	return false
}

// hit a url - is it one of certain response codes? leave connections open!
func (scanner *URLScanner) checkURL(uri string) *URLResult {

	if scanner.visited(uri) {
		return nil
	}

	if scanner.busyChan != nil {
		scanner.busyChan <- uri
	}

	var code int
	var location string
	var result *URLResult

	if err := retry.Do(func() error {

		req, err := http.NewRequest(http.MethodGet, uri, nil)
		if err != nil {
			return err
		}

		if scanner.extraHeader != "" {
			parts := strings.SplitN(scanner.extraHeader, ":", 2)
			if len(parts) == 2 {
				req.Header.Set(parts[0], strings.TrimPrefix(parts[1], " "))
			}
		}

		resp, err := scanner.client.Do(req)
		if err != nil {
			return nil
		}
		defer func() { _ = resp.Body.Close() }()

		code = resp.StatusCode
		location = resp.Header.Get("Location")

		if location != "" {
			if !strings.Contains(location, "://") {
				if parsed, err := url.Parse(uri); err == nil {
					if relative, err := url.Parse(location); err == nil {
						scanner.queue(parsed.ResolveReference(relative).String())
					}
				}
			} else {
				scanner.queue(location)
			}
		}

		for _, status := range scanner.positiveStatusCodes {
			if status == code {
				parsedURL, err := url.Parse(uri)
				if err != nil {
					return nil
				}

				for _, ext := range scanner.backupExtensions {
					scanner.queue(uri + ext)
				}

				contentType := resp.Header.Get("Content-Type")

				if scanner.enableSpidering && (contentType == "" || strings.Contains(contentType, "html")) {
					body, err := ioutil.ReadAll(resp.Body)
					if err == nil {
						for _, link := range findLinks(uri, body) {
							scanner.queue(link)
						}
					}
				} else {
					_, _ = io.Copy(ioutil.Discard, resp.Body)
				}

				result = &URLResult{
					StatusCode: code,
					URL:        *parsedURL,
				}

				break
			}
		}

		return nil

	}, retry.Attempts(10), retry.DelayType(retry.BackOffDelay)); err != nil {
		return nil
	}

	return result
}

var linkAttributes = []string{"src", "href"}

func findLinks(currentURL string, html []byte) []string {

	base, err := url.Parse(currentURL)
	if err != nil {
		return nil
	}

	var results []string

	source := string(html)

	var bestIndex int
	var bestAttr string
	var link string

	for source != "" {
		bestIndex = -1
		for _, attr := range linkAttributes {
			index := strings.Index(source, fmt.Sprintf("%s=", attr))
			if index < bestIndex || bestIndex == -1 {
				bestIndex = index
				bestAttr = attr
			}
		}
		if bestIndex == -1 {
			break
		}
		source = source[bestIndex+len(bestAttr)+1:]
		switch source[0] {
		case '"':
			source = source[1:]
			index := strings.Index(source, "\"")
			if index == -1 {
				continue
			}
			link = source[:index]
		case '\'':
			source = source[1:]
			index := strings.Index(source, "'")
			if index == -1 {
				continue
			}
			link = source[:index]
		default:
			spaceIndex := strings.Index(source, " ")
			bIndex := strings.Index(source, ">")
			if spaceIndex == -1 && bIndex == -1 {
				continue
			}
			if spaceIndex == -1 {
				link = source[:bIndex]
			} else if bIndex == -1 {
				link = source[:spaceIndex]
			} else {
				if spaceIndex < bIndex {
					bIndex = spaceIndex
				}
				link = source[:bIndex]
			}
		}

		u, err := url.Parse(link)
		if err != nil {
			return nil
		}

		if u.Host != "" && u.Host != base.Host {
			continue
		}

		results = append(results, base.ResolveReference(u).String())
	}

	return results
}
