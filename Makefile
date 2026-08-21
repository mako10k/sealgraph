GOCYCLO_VERSION := v0.6.0
DEADCODE_VERSION := v0.49.0
MAX_CYCLOMATIC_COMPLEXITY := 20
VERSION ?= 0.1.0-dev
DIST_DIR ?= dist
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
BASH_COMPLETION_DIR ?= $(PREFIX)/share/bash-completion/completions

.PHONY: fmt vet test test-race completion-check clone-check complexity-check deadcode-check static-analysis check build install uninstall build-git-sidecar-placeholder beta-artifacts beta-artifact-smoke

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test ./...

test-race:
	go test -race ./...

completion-check: build
	bash tests/completion_regression.sh ./bin/sealgraph

clone-check:
	npm run clone-check

complexity-check:
	go run github.com/fzipp/gocyclo/cmd/gocyclo@$(GOCYCLO_VERSION) -over $(MAX_CYCLOMATIC_COMPLEXITY) internal cmd

deadcode-check:
	go run golang.org/x/tools/cmd/deadcode@$(DEADCODE_VERSION) -test ./...

static-analysis: vet clone-check complexity-check deadcode-check

check: fmt static-analysis test completion-check

build:
	mkdir -p bin
	go build -buildvcs=false -trimpath -ldflags "-X github.com/mako10k/sealgraph/internal/cli.Version=$(VERSION)" -o bin/sealgraph ./cmd/sealgraph

install: build
	install -d "$(DESTDIR)$(BINDIR)" "$(DESTDIR)$(BASH_COMPLETION_DIR)"
	install -m 0755 bin/sealgraph "$(DESTDIR)$(BINDIR)/sealgraph"
	install -m 0644 completions/sealgraph.bash "$(DESTDIR)$(BASH_COMPLETION_DIR)/sealgraph"

uninstall:
	rm -f "$(DESTDIR)$(BINDIR)/sealgraph" "$(DESTDIR)$(BASH_COMPLETION_DIR)/sealgraph"

build-git-sidecar-placeholder:
	mkdir -p bin
	go build -o bin/git-sealgraph ./cmd/git-sealgraph

beta-artifacts:
	./scripts/build-release.sh $(VERSION) $(DIST_DIR)

beta-artifact-smoke: beta-artifacts
	./scripts/artifact-smoke.sh $(DIST_DIR)/sealgraph_$(VERSION)_linux_amd64.tar.gz $(VERSION)
