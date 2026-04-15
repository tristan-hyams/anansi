# Apologies for any failed commands. This Makefile was built on Windows.

.PHONY: build test lint run clean tidy update setup docker docker-run

BINARY  := bin\anansi.exe
URL     ?= https://crawlme.monzo.com/
ARGS    ?=

export CGO_ENABLED=0

build:
	go build -o $(BINARY) ./cmd/anansi

test:
	go test -cover ./...

lint:
	revive -config revive.toml ./...

run: build
	$(BINARY) $(ARGS) $(URL)

clean:
	if exist bin rmdir /s /q bin

tidy:
	go mod tidy

update: tidy
	go get -u ./...
	go mod tidy

setup:
	go install golang.org/x/tools/gopls@latest
	go install github.com/go-delve/delve/cmd/dlv@latest
	go install github.com/mgechev/revive@latest

docker:
	docker build -t anansi .

docker-run: docker
	docker run --rm anansi $(ARGS) $(URL)
