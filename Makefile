
default: build

build:
	./scripts/build.sh

test:
	GO111MODULE=on go test -v -race -timeout 30m ./...

import-wordlists:
	./scripts/import-wordlists.sh
