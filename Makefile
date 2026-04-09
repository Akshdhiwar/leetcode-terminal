BINARY=lc
MODULE=github.com/user/leetcode-cli
LDFLAGS=-ldflags="-s -w"

.PHONY: build run install clean build-all

build:
	go build $(LDFLAGS) -o $(BINARY) .

install: build
	mv $(BINARY) /usr/local/bin/$(BINARY)
	@echo "Installed to /usr/local/bin/$(BINARY)"

run:
	go run . $(ARGS)

clean:
	rm -rf dist/ $(BINARY)

build-all:
	mkdir -p dist
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o dist/lc-linux-amd64 .
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o dist/lc-linux-arm64 .
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o dist/lc-darwin-amd64 .
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o dist/lc-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/lc-windows-amd64.exe .
	@echo "All binaries built in dist/"
	@ls -lh dist/
