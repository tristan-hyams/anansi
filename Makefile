# Apologies for any failed commands. This Makefile was built on Windows.

.PHONY: build test lint run clean tidy update docker docker-run

BINARY  := bin\anansi.exe
URL     ?= https://crawlme.monzo.com/

export CGO_ENABLED=0

build:
	go build -o $(BINARY) ./cmd/anansi

test:
	go test -cover ./...

lint:
	revive -config revive.toml ./...

run: build
	$(BINARY) $(URL)

clean:
	if exist bin rmdir /s /q bin

tidy:
	go mod tidy

update: tidy
	go get -u ./...
	go mod tidy

docker:
	docker build -t anansi .

docker-run: docker
	docker run --rm anansi $(URL)
