.PHONY: fmt vet test clone-check check build

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test ./...

clone-check:
	npm run clone-check

check: fmt vet test clone-check

build:
	mkdir -p bin
	go build -o bin/sealgraph ./cmd/sealgraph
	go build -o bin/git-sealgraph ./cmd/git-sealgraph
