
default: build

build:
	./scripts/build.sh

test:
	GO111MODULE=on go test -v ./....

import-wordlist:
 	./scripts/import-wordlist.sh
