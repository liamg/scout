package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/liamg/scout/pkg/scan"
	"github.com/liamg/scout/pkg/wordlist"
	"github.com/liamg/tml"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var statusCodes []string
var filename string
var headers []string
var extensions = []string{"php", "htm", "html", "txt"}
var enableSpidering bool
var proxy string

var urlCmd = &cobra.Command{
	Use:   "url [url]",
	Short: "Discover URLs on a given web server.",
	Long:  "Scout will discover URLs relative to the provided one.",
	Run: func(cmd *cobra.Command, args []string) {

		log.SetOutput(ioutil.Discard)

		if debug {
			logrus.SetLevel(logrus.DebugLevel)
		}

		if noColours {
			tml.DisableFormatting()
		}

		if len(args) == 0 {
			tml.Println("<bold><red>Error:</red></bold> You must specify a target URL.")
			os.Exit(1)
		}

		parsedURL, err := url.ParseRequestURI(args[0])
		if err != nil {
			tml.Printf("<bold><red>Error:</red></bold> Invalid URL: %s\n", err)
			os.Exit(1)
		}

		resultChan := make(chan scan.URLResult)
		busyChan := make(chan string, 0x400)

		var intStatusCodes []int

		for _, code := range statusCodes {
			i, err := strconv.Atoi(code)
			if err != nil {
				tml.Printf("<bold><red>Error:</red></bold> Invalid status code entered: %s.\n", code)
				os.Exit(1)
			}
			intStatusCodes = append(intStatusCodes, i)
		}

		options := []scan.URLOption{
			scan.WithPositiveStatusCodes(intStatusCodes),
			scan.WithTargetURL(*parsedURL),
			scan.WithResultChan(resultChan),
			scan.WithBusyChan(busyChan),
			scan.WithParallelism(parallelism),
			scan.WithExtensions(extensions),
			scan.WithFilename(filename),
			scan.WithSkipSSLVerification(skipSSLVerification),
			scan.WithExtraHeaders(headers),
			scan.WithSpidering(enableSpidering),
		}

		if wordlistPath != "" {
			words, err := wordlist.FromFile(wordlistPath)
			if err != nil {
				tml.Printf("<bold><red>Error:</red></bold> %s\n", err)
				os.Exit(1)
			}
			options = append(options, scan.WithWordlist(words))
		}

		tml.Printf(
			`<blue>[</blue><yellow>+</yellow><blue>] Target URL</blue><yellow>      %s
<blue>[</blue><yellow>+</yellow><blue>] Routines</blue><yellow>        %d 
<blue>[</blue><yellow>+</yellow><blue>] Extensions</blue><yellow>      %s 
<blue>[</blue><yellow>+</yellow><blue>] Positive Codes</blue><yellow>  %s
<blue>[</blue><yellow>+</yellow><blue>] Spider</blue><yellow>          %t

`,
			parsedURL.String(),
			parallelism,
			strings.Join(extensions, ","),
			strings.Join(statusCodes, ","),
			enableSpidering,
		)

		if proxy != "" {
			proxyUrl, err := url.Parse(proxy)
			if err != nil {
				tml.Printf("<bold><red>Error:</red></bold> Invalid Proxy URL: %s\n", err)
				os.Exit(1)
			}
			options = append(options, scan.WithProxy(proxyUrl))
		}

		scanner := scan.NewURLScanner(options...)

		waitChan := make(chan struct{})

		genericOutputChan := make(chan string)
		importantOutputChan := make(chan string)

		go func() {
			for result := range resultChan {
				importantOutputChan <- tml.Sprintf("<blue>[</blue><yellow>%d</yellow><blue>]</blue> %s\n", result.StatusCode, result.URL.String())
			}
			close(waitChan)
		}()

		go func() {
			defer func() {
				_ = recover()
			}()
			for uri := range busyChan {
				genericOutputChan <- tml.Sprintf("Checking %s...", uri)
			}
		}()

		outChan := make(chan struct{})
		go func() {

			defer close(outChan)

			for {
				select {
				case output := <-importantOutputChan:
					clearLine()
					fmt.Print(output)
				FLUSH:
					for {
						select {
						case str := <-genericOutputChan:
							if str == "" {
								break FLUSH
							}
						default:
							break FLUSH
						}
					}
				case <-waitChan:
					return
				case output := <-genericOutputChan:
					clearLine()
					fmt.Print(output)
				}
			}

		}()

		results, err := scanner.Scan()
		if err != nil {
			clearLine()
			tml.Printf("<bold><red>Error:</red></bold> %s\n", err)
			os.Exit(1)
		}
		logrus.Debug("Waiting for output to flush...")
		<-waitChan
		close(genericOutputChan)
		<-outChan

		clearLine()
		tml.Printf("\n<bold><green>Scan complete. %d results found.</green></bold>\n\n", len(results))

	},
}

func clearLine() {
	fmt.Printf("\033[2K\r")
}

func init() {
	urlCmd.Flags().StringVarP(&filename, "filename", "f", filename, "Filename to seek in the directory being searched. Useful when all directories report 404 status.")
	urlCmd.Flags().StringSliceVarP(&statusCodes, "status-codes", "c", statusCodes, "HTTP status codes which indicate a positive find.")
	urlCmd.Flags().StringSliceVarP(&extensions, "extensions", "x", extensions, "File extensions to detect.")
	urlCmd.Flags().StringSliceVarP(&headers, "header", "H", headers, "Extra header to send with requests (can be specified multiple times).")
	urlCmd.Flags().BoolVarP(&enableSpidering, "spider", "s", enableSpidering, "Spider links within page content")
	urlCmd.Flags().StringVarP(&proxy, "proxy", "x", proxy, "HTTP Porxy to use")

	rootCmd.AddCommand(urlCmd)
}
