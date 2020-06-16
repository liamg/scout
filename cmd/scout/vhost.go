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

var ip string
var port int
var useSSL bool
var contentHashing bool

var vhostCmd = &cobra.Command{
	Use:   "vhost [base_domain]",
	Short: "Discover VHOSTs on a given web server.",
	Long:  "Scout will discover VHOSTs as subdomains of the provided base domain.",
	Run: func(cmd *cobra.Command, args []string) {

		log.SetOutput(ioutil.Discard)

		if debug {
			logrus.SetLevel(logrus.DebugLevel)
		}

		if noColours {
			tml.DisableFormatting()
		}

		if len(args) == 0 {
			tml.Println("<bold><red>Error:</red></bold> You must specify a base domain.")
			os.Exit(1)
		}

		baseDomain := args[0]

		if strings.HasPrefix(baseDomain, "https://") {
			useSSL = true
		}

		if parsedURL, err := url.Parse(args[0]); err == nil && parsedURL.Host != "" {
			baseDomain = parsedURL.Host
		}

		if strings.Contains(baseDomain, "/") {
			baseDomain = baseDomain[:strings.Index(baseDomain, "/")]
		}

		resultChan := make(chan scan.VHOSTResult)
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

		ipStr := ip
		if ipStr == "" {
			ipStr = "-"
		}

		portStr := strconv.Itoa(port)
		if port == 0 {
			portStr = "-"
		}

		options := &scan.VHOSTOptions{
			BaseDomain:     baseDomain,
			Parallelism:    parallelism,
			ResultChan:     resultChan,
			BusyChan:       busyChan,
			UseSSL:         useSSL,
			IP:             ip,
			Port:           port,
			ContentHashing: contentHashing,
		}
		if wordlistPath != "" {
			var err error
			options.Wordlist, err = wordlist.FromFile(wordlistPath)
			if err != nil {
				tml.Printf("<bold><red>Error:</red></bold> %s\n", err)
				os.Exit(1)
			}
		}
		options.Inherit()

		tml.Printf(
			`<blue>[</blue><yellow>+</yellow><blue>] Base Domain</blue><yellow>     %s
<blue>[</blue><yellow>+</yellow><blue>] Routines</blue><yellow>        %d 
<blue>[</blue><yellow>+</yellow><blue>] IP</blue><yellow>              %s 
<blue>[</blue><yellow>+</yellow><blue>] Port</blue><yellow>            %s 
<blue>[</blue><yellow>+</yellow><blue>] Using SSL</blue><yellow>       %t

`,
			options.BaseDomain,
			options.Parallelism,
			ipStr,
			portStr,
			options.UseSSL,
		)

		scanner := scan.NewVHOSTScanner(options)

		waitChan := make(chan struct{})

		genericOutputChan := make(chan string)
		importantOutputChan := make(chan string)

		go func() {
			for result := range resultChan {
				importantOutputChan <- tml.Sprintf("%s\n", result.VHOST)
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

func init() {

	vhostCmd.Flags().BoolVar(&useSSL, "ssl", useSSL, "Use HTTPS when connecting to the server.")
	vhostCmd.Flags().StringVar(&ip, "ip", ip, "IP address to connect to - defaults to the DNS A record for the base domain.")
	vhostCmd.Flags().IntVar(&port, "port", port, "Port to connect to - defaults to 80 or 443 if --ssl is set.")
	vhostCmd.Flags().BoolVar(&contentHashing, "hash-contents", contentHashing, "Hash each response body to detect differences for catch-all scenarios.")

	rootCmd.AddCommand(vhostCmd)
}
