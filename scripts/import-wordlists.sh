#!/bin/bash

go get -u github.com/go-bindata/go-bindata/...
go-bindata -o internal/app/scout/data/wordlists.go assets/
sed -i 's/package main/package data/g' internal/app/scout/data/wordlists.go
