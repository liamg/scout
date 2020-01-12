# Scout

[![Travis Build Status](https://travis-ci.org/liamg/scout.svg?branch=master)](https://travis-ci.org/liamg/scout)

Scout is a URL fuzzer for discovering undisclosed files and directories on a web server. 

<p align="center">
  <img width="746" height="417" src="./demo.png" />
</p>

A full word list is included in the binary, meaning maximum portability and minimal configuration. Aim and fire!

## Usage

```bash

Usage:
  scout [url] [flags]

Flags:
  -d, --debug                    Enable debug logging.
  -x, --extensions stringArray   File extensions to detect. (default [php,htm,html])
  -h, --help                     help for scout
  -n, --no-colours               Disable coloured output.
  -p, --parallelism int          Parallel routines to use for sending requests. (default 10)
  -w, --wordlist string          Path to wordlist file. If this is not specified an internal wordlist will be used.

```


### Discover URLs

### Discover VHOSTs


## Installation

```bash
curl -s "https://raw.githubusercontent.com/liamg/scout/master/scripts/install.sh" | bash
```
