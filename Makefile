.PHONY: build test clean lint

build:
	go build -o orasuck main.go

test:
	go test ./...

clean:
	rm -f orasuck

lint:
	golangci-lint run --timeout 5m

install-deps:
	go mod tidy