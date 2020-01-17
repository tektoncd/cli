YAML_FILES := $(shell find . -type f -regex ".*y[a]ml" -print)

ifneq (,$(wildcard ./VERSION))
LDFLAGS := -ldflags "-X github.com/tektoncd/cli/pkg/cmd/version.clientVersion=`cat VERSION`"
endif

ifneq ($(RELEASE_VERSION),)
LDFLAGS := -ldflags "-X github.com/tektoncd/cli/pkg/cmd/version.clientVersion=$(RELEASE_VERSION)"
endif

all: bin/tkn test

FORCE:

vendor:
	@go mod vendor

.PHONY: cross
cross: amd64 386 arm arm64 ## build cross platform binaries

.PHONY: amd64
amd64:
	GOOS=linux GOARCH=amd64 go build -mod=vendor $(LDFLAGS) -o bin/tkn-linux-amd64 ./cmd/tkn
	GOOS=windows GOARCH=amd64 go build -mod=vendor $(LDFLAGS) -o bin/tkn-windows-amd64 ./cmd/tkn
	GOOS=darwin GOARCH=amd64 go build -mod=vendor $(LDFLAGS) -o bin/tkn-darwin-amd64 ./cmd/tkn

.PHONY: 386
386:
	GOOS=linux GOARCH=386 go build -mod=vendor $(LDFLAGS) -o bin/tkn-linux-386 ./cmd/tkn
	GOOS=windows GOARCH=386 go build -mod=vendor $(LDFLAGS) -o bin/tkn-windows-386 ./cmd/tkn
	GOOS=darwin GOARCH=386 go build -mod=vendor $(LDFLAGS) -o bin/tkn-darwin-386 ./cmd/tkn

.PHONY: arm
arm:
	GOOS=linux GOARCH=arm go build -mod=vendor $(LDFLAGS) -o bin/tkn-linux-arm ./cmd/tkn

.PHONY: arm64
arm64:
	GOOS=linux GOARCH=arm64 go build -mod=vendor $(LDFLAGS) -o bin/tkn-linux-arm64 ./cmd/tkn

bin/%: cmd/% FORCE
	go build -mod=vendor $(LDFLAGS) -v -o $@ ./$<

check: lint test

.PHONY: test
test: test-unit ## run all tests

.PHONY: lint
lint: ## run linter(s)
	@echo "Linting..."
	@golangci-lint run ./... --timeout 5m

.PHONY: lint-yaml
lint-yaml: ${YAML_FILES} ## runs yamllint on all yaml files
	@yamllint -c .yamllint $(YAML_FILES)

.PHONY: test-unit
test-unit: ./vendor ## run unit tests
	@echo "Running unit tests..."
	@go test -failfast -v -cover ./...

.PHONY: test-unit-update-golden
test-unit-update-golden: ./vendor ## run unit tests (updating golden files)
	@echo "Running unit tests updating golden files..."
	@./hack/update-golden.sh

.PHONY: test-e2e
test-e2e: bin/tkn ## run e2e tests
	@echo "Running e2e tests..."
	@LOCAL_CI_RUN=true bash ./test/e2e-tests.sh

.PHONY: docs
docs: bin/docs ## update docs
	@echo "Update generated docs"
	@./bin/docs --target=./docs/cmd

.PHONY: man
man: bin/docs ## update manpages
	@echo "Update generated manpages"
	@./bin/docs --target=./docs/man/man1 --kind=man

.PHONY: clean
clean: ## clean build artifacts
	rm -fR bin VERSION

.PHONY: fmt ## formats teh god code(excludes vendors dir)
fmt:
	@go fmt $(go list ./... | grep -v /vendor/)

.PHONY: help
help: ## print this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {gsub("\\\\n",sprintf("\n%22c",""), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
