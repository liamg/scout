package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/liamg/scout/pkg/scan"
	"github.com/liamg/scout/pkg/wordlist"
	"github.com/liamg/tml"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "scout [url]",
	Short: "Scout is a portable URL fuzzer",
	Long:  `A fast and portable url fuzzer - see https://github.com/liamg/scout for more information`,
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

		resultChan := make(chan scan.Result)
		busyChan := make(chan string, 0x400)

		options := &scan.Options{
			TargetURL:           *parsedURL,
			ResultChan:          resultChan,
			BusyChan:            busyChan,
			Parallelism:         parallelism,
			Extensions:          extensions,
			Filename:            filename,
			SkipSSLVerification: skipSSLVerification,
		}
		if wordlistPath != "" {
			options.Wordlist, err = wordlist.FromFile(wordlistPath)
			if err != nil {
				tml.Printf("<bold><red>Error:</red></bold> %s\n", err)
				os.Exit(1)
			}
		}
		options.Inherit()

		var codeStrings []string
		for _, code := range options.PositiveStatusCodes {
			codeStrings = append(codeStrings, fmt.Sprintf("%d", code))
		}

		tml.Printf(
			`
<blue>[</blue><yellow>+</yellow><blue>] Target URL</blue><yellow>      %s
<blue>[</blue><yellow>+</yellow><blue>] Routines</blue><yellow>        %d 
<blue>[</blue><yellow>+</yellow><blue>] Extensions</blue><yellow>      %s 
<blue>[</blue><yellow>+</yellow><blue>] Positive Codes</blue><yellow>  %s

`,
			options.TargetURL.String(),
			options.Parallelism,
			strings.Join(options.Extensions, ","),
			strings.Join(codeStrings, ","),
		)

		scanner := scan.NewScanner(options)

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
					fmt.Printf(output)
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
					fmt.Printf(output)
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

var parallelism = scan.DefaultOptions.Parallelism
var extensions = scan.DefaultOptions.Extensions
var noColours = false
var wordlistPath string
var debug bool
var filename string
var skipSSLVerification bool

func main() {

	rootCmd.Flags().IntVarP(&parallelism, "parallelism", "p", parallelism, "Parallel routines to use for sending requests.")
	rootCmd.Flags().StringArrayVarP(&extensions, "extensions", "x", extensions, "File extensions to detect.")
	rootCmd.Flags().BoolVarP(&noColours, "no-colours", "n", noColours, "Disable coloured output.")
	rootCmd.Flags().StringVarP(&wordlistPath, "wordlist", "w", wordlistPath, "Path to wordlist file. If this is not specified an internal wordlist will be used.")
	rootCmd.Flags().BoolVarP(&debug, "debug", "d", debug, "Enable debug logging.")
	rootCmd.Flags().StringVarP(&filename, "filename", "f", filename, "Filename to seek in the directory being searched. Useful when all directories report 404 status.")
	rootCmd.Flags().BoolVarP(&skipSSLVerification, "skip-ssl-verify", "k", skipSSLVerification, "Skip SSL certificate verification.")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
