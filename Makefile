.PHONY: build

VERSION := `git fetch --tags && git tag | sort -V | tail -1`
PKG=github.com/r35krag0th/win-loss-rux

# Global LD Flags to use
VERSION_LDFLAG_VALUE="github.com/r35krag0th/win-loss-rux/v1/version.Version=$(VERSION)"
LDFLAGS=-ldflags "-X=$(VERSION_LDFLAG_VALUE)"
COVER=--cover --coverprofile=cover.out

test-cover:
	go test ./... --race $(COVER) $(PKG) -v
	go tool cover -html=cover.out

format:
	go fmt ./...

test: format
	go install golang.org/x/lint/golint@latest
	go vet ./...
	golint ./...
	go test ./... --race $(PKG) -v

build: format
	golint ./...
	go vet ./...
	go mod tidy
	go build $(LDFLAGS) -o win-loss-rux

clean:
	@echo "\033[32m 🬯\033[0m Cleaning up"
	rm -rf build cover.out win-loss-rux

release-builds:
	rm -rf build
	mkdir build
	env GOOS="windows" GOARCH="amd64" go build -o "build/win-loss-rux-windows-amd64.exe" $(LDFLAGS)
	env GOOS="windows" GOARCH="386" go build -o "build/win-loss-rux-windows-386.exe" $(LDFLAGS)
	env GOOS="linux" GOARCH="amd64" go build -o "build/win-loss-rux-linux-amd64" $(LDFLAGS)
	env GOOS="linux" GOARCH="arm" go build -o "build/win-loss-rux-linux-arm" $(LDFLAGS)
	env GOOS="linux" GOARCH="mips" go build -o "build/win-loss-rux-linux-mips" $(LDFLAGS)
	env GOOS="linux" GOARCH="mips" go build -o "build/win-loss-rux-linux-mips" $(LDFLAGS)
	env GOOS="darwin" GOARCH="amd64" go build -o "build/win-loss-rux-darwin-amd64" $(LDFLAGS)

dockerbuild:
	docker build --build-arg ldflags=$(VERSION_LDFLAG_VALUE) -f Dockerfile -t ghcr.io/r35krag0th/win-loss-rux:latest -t ghcr.io/r35krag0th/win-loss-rux:$(VERSION) .
