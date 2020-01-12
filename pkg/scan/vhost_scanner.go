package scan

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/avast/retry-go"
	"github.com/sirupsen/logrus"
)

type VHOSTScanner struct {
	client  *http.Client
	options *VHOSTOptions
	badCode int
	badHash string
}

func NewVHOSTScanner(opt *VHOSTOptions) *VHOSTScanner {

	if opt == nil {
		opt = &DefaultVHOSTOptions
	}

	opt.Inherit()

	client := &http.Client{
		Timeout:   opt.Timeout,
		Transport: http.DefaultTransport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	return &VHOSTScanner{
		options: opt,
		client:  client,
	}
}

func (scanner *VHOSTScanner) forceRequestsToIP(ip net.IP) {
	dialer := &net.Dialer{
		Timeout: scanner.options.Timeout,
	}
	scanner.client.Transport.(*http.Transport).DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		port := scanner.options.Port
		if port == 0 {
			if scanner.options.UseSSL {
				port = 443
			} else {
				port = 80
			}
		}
		return dialer.DialContext(ctx, network, fmt.Sprintf("%s:%d", ip, port))
	}
}

func md5Hash(input string) string {
	hash := md5.New()
	io.WriteString(hash, input)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func (scanner *VHOSTScanner) Scan() ([]string, error) {

	logrus.Debug("Looking up base domain...")

	ip := scanner.options.IP
	if ip != "" {
		if parsed := net.ParseIP(ip); parsed == nil {
			return nil, fmt.Errorf("invalid IP address specified: %s", ip)
		}
	} else {
		ips, err := net.LookupIP(scanner.options.BaseDomain)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve base domain: %s", err)
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("failed to resolve base domain: no A record found")
		}
		ip = ips[0].String()
	}

	scanner.forceRequestsToIP(net.ParseIP(ip))

	badVHOST := fmt.Sprintf("%s.%s", md5Hash(time.Now().String()), scanner.options.BaseDomain)

	url := "http://" + badVHOST

	if scanner.options.UseSSL {
		url = "https://" + badVHOST
	}

	resp, err := scanner.client.Get(url)
	if err != nil {
		return nil, err
	}
	scanner.badCode = resp.StatusCode
	if scanner.options.ContentHashing {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		scanner.badHash = md5Hash(string(data))
	} else {
		io.Copy(ioutil.Discard, resp.Body)
	}
	resp.Body.Close()

	jobs := make(chan string, scanner.options.Parallelism)
	results := make(chan VHOSTResult, scanner.options.Parallelism)

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
	var foundVHOSTs []string

	go func() {
		for result := range results {
			if scanner.options.ResultChan != nil {
				scanner.options.ResultChan <- result
			}
			foundVHOSTs = append(foundVHOSTs, result.VHOST)
		}
		if scanner.options.ResultChan != nil {
			close(scanner.options.ResultChan)
		}
		close(waitChan)
	}()

	logrus.Debug("Adding jobs...")

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
			vhost := word + "." + scanner.options.BaseDomain
			jobs <- vhost
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

	return foundVHOSTs, nil
}

func (scanner *VHOSTScanner) worker(jobs <-chan string, results chan<- VHOSTResult) {
	for j := range jobs {
		if result := scanner.checkVHOST(j); result != nil {
			results <- *result
		}
	}
}

// hit a url - is it one of certain response codes? leave connections open!
func (scanner *VHOSTScanner) checkVHOST(vhost string) *VHOSTResult {

	if scanner.options.BusyChan != nil {
		scanner.options.BusyChan <- vhost
	}

	var code int
	var hash string

	url := "http://" + vhost

	if scanner.options.UseSSL {
		url = "https://" + vhost
	}

	if err := retry.Do(func() error {
		resp, err := scanner.client.Get(url)
		if err != nil {
			return nil
		}

		if scanner.options.ContentHashing {
			data, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			hash = md5Hash(string(data))
		} else {
			io.Copy(ioutil.Discard, resp.Body)
		}
		resp.Body.Close()

		code = resp.StatusCode
		return nil
	}, retry.Attempts(10), retry.DelayType(retry.BackOffDelay)); err != nil {
		return nil
	}

	if code != scanner.badCode || (scanner.options.ContentHashing && hash != scanner.badHash) {
		return &VHOSTResult{
			StatusCode: code,
			VHOST:      vhost,
		}
	}

	return nil
}
