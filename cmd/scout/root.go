package main

import (
	"strconv"

	"github.com/liamg/scout/pkg/scan"
	"github.com/spf13/cobra"
)

var parallelism = scan.DefaultURLOptions.Parallelism
var noColours = false
var wordlistPath string
var debug bool
var skipSSLVerification bool

var rootCmd = &cobra.Command{
	Use:   "scout",
	Short: "Scout is a portable URL fuzzer",
	Long:  `A fast and portable url fuzzer - see https://github.com/liamg/scout for more information`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {

	},
}

func init() {
	for _, code := range scan.DefaultURLOptions.PositiveStatusCodes {
		statusCodes = append(statusCodes, strconv.Itoa(code))
	}

	rootCmd.PersistentFlags().IntVarP(&parallelism, "parallelism", "p", parallelism, "Parallel routines to use for sending requests.")
	rootCmd.PersistentFlags().BoolVarP(&noColours, "no-colours", "n", noColours, "Disable coloured output.")
	rootCmd.PersistentFlags().StringVarP(&wordlistPath, "wordlist", "w", wordlistPath, "Path to wordlist file. If this is not specified an internal wordlist will be used.")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", debug, "Enable debug logging.")
	rootCmd.PersistentFlags().BoolVarP(&skipSSLVerification, "skip-ssl-verify", "k", skipSSLVerification, "Skip SSL certificate verification.")
}
