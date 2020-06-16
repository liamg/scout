package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/liamg/scout/internal/app/scout/version"
	"github.com/spf13/cobra"
)

var parallelism = 10
var noColours = false
var wordlistPath string
var debug bool
var skipSSLVerification bool
var positiveStatusCodes = []int{
	http.StatusOK,
	http.StatusBadRequest,
	http.StatusInternalServerError,
	http.StatusMethodNotAllowed,
	http.StatusNoContent,
	http.StatusUnauthorized,
	http.StatusForbidden,
	http.StatusFound,
	http.StatusMovedPermanently,
}

var rootCmd = &cobra.Command{
	Use:   "scout",
	Short: "Scout is a portable URL fuzzer and spider",
	Long:  `A fast and portable url fuzzer and spider - see https://github.com/liamg/scout for more information`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		fmt.Printf(`
                          __ 
   ______________  __  __/ /_   
  / ___/ ___/ __ \/ / / / __/    %s
 (__  ) /__/ /_/ / /_/ / /_      http://github.com/liamg/scout
/____/\___/\____/\__,_/\__/

`, version.Version)
	},
}

func init() {
	for _, code := range positiveStatusCodes {
		statusCodes = append(statusCodes, strconv.Itoa(code))
	}

	rootCmd.PersistentFlags().IntVarP(&parallelism, "parallelism", "p", parallelism, "Parallel routines to use for sending requests.")
	rootCmd.PersistentFlags().BoolVarP(&noColours, "no-colours", "n", noColours, "Disable coloured output.")
	rootCmd.PersistentFlags().StringVarP(&wordlistPath, "wordlist", "w", wordlistPath, "Path to wordlist file. If this is not specified an internal wordlist will be used.")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", debug, "Enable debug logging.")
	rootCmd.PersistentFlags().BoolVarP(&skipSSLVerification, "skip-ssl-verify", "k", skipSSLVerification, "Skip SSL certificate verification.")
}
