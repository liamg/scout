#!/bin/bash

set -x

go install github.com/go-bindata/go-bindata/go-bindata@latest
go-bindata -o internal/app/scout/data/wordlists.go assets/
cp internal/app/scout/data/wordlists.go internal/app/scout/data/wordlists.go.old
sed -e 's/package main/package data/g' internal/app/scout/data/wordlists.go.old > internal/app/scout/data/wordlists.go

rm internal/app/scout/data/wordlists.go.old