
default: build

build:
	./scripts/build.sh

test:
	GO111MODULE=on go test -v -race -timeout 30m ./...

import-wordlist:
	./scripts/import-wordlist.sh
