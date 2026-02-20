.PHONY: build run test lint clean generate-server

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

generate-server:
	go run github.com/ogen-go/ogen/cmd/ogen@v1.18.0 \
		--config internal/ogenserver/ogen.yaml \
		--target internal/ogenserver/gen \
		--clean \
		api/openapi/server-openapi.yaml
