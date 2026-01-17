.PHONY: build run test lint clean

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

test:
	go test -v ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/
