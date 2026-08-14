.PHONY: fmt vet test check build

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test ./...

check: fmt vet test

build:
	mkdir -p bin
	go build -o bin/sealgraph ./cmd/sealgraph
	go build -o bin/git-sealgraph ./cmd/git-sealgraph
