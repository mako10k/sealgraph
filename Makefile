GOCYCLO_VERSION := v0.6.0
DEADCODE_VERSION := v0.49.0
MAX_CYCLOMATIC_COMPLEXITY := 20

.PHONY: fmt vet test clone-check complexity-check deadcode-check static-analysis check build

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test ./...

clone-check:
	npm run clone-check

complexity-check:
	go run github.com/fzipp/gocyclo/cmd/gocyclo@$(GOCYCLO_VERSION) -over $(MAX_CYCLOMATIC_COMPLEXITY) internal cmd

deadcode-check:
	go run golang.org/x/tools/cmd/deadcode@$(DEADCODE_VERSION) -test ./...

static-analysis: vet clone-check complexity-check deadcode-check

check: fmt static-analysis test

build:
	mkdir -p bin
	go build -o bin/sealgraph ./cmd/sealgraph
	go build -o bin/git-sealgraph ./cmd/git-sealgraph
