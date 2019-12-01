package main

import (
	"fmt"
	"net/url"
	"os"

	"github.com/liamg/scout/pkg/scan"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "scout [url]",
	Short: "Scout is a portable URL fuzzer",
	Long:  `A fast and portable url fuzzer - see https://github.com/liamg/scout for more information`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) == 0 {
			fmt.Println("You must specify a target URL.")
			os.Exit(1)
		}

		parsedURL, err := url.ParseRequestURI(args[0])
		if err != nil {
			fmt.Printf("Invalid URL: %s\n", err)
			os.Exit(1)
		}

		resultChan := make(chan url.URL)

		scanner := scan.NewScanner(&scan.Options{
			TargetURL:  *parsedURL,
			ResultChan: resultChan,
		})

		waitChan := make(chan struct{})

		go func() {
			for uri := range resultChan {
				fmt.Printf("Discovered URL: %s\n", uri.String())
			}
			close(waitChan)
		}()

		if _, err := scanner.Scan(); err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
		close(resultChan)
		<-waitChan

	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
