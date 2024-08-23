GO   = go
PKGS = $(or $(PKG),$(shell env GO111MODULE=on $(GO) list ./... | grep -v 'github.com/tektoncd/cli/third_party/'))
BIN  = $(CURDIR)/.bin

export GO111MODULE=on

V = 0
Q = $(if $(filter 1,$V),,@)
M = $(shell printf "\033[34;1m🐱\033[0m")

TIMEOUT_UNIT = 5m
TIMEOUT_E2E  = 20m

# Get golangci_version from tools/go.mod to eliminate the manual bump
GOLANGCI_VERSION = $(shell cat tools/go.mod | grep golangci-lint | awk '{ print $$3 }')

YAML_FILES := $(shell find . -type f -regex ".*y[a]ml" -print)

ifneq ($(NAMESPACE),)
	NAMESPACELDFLAG := -X github.com/tektoncd/cli/pkg/cmd/version.namespace=$(NAMESPACE)
endif

ifneq ($(SKIP_CHECK_FLAG),)
	SKIPLDFLAG := -X github.com/tektoncd/cli/pkg/cmd/version.skipCheckFlag=$(SKIP_CHECK_FLAG)
endif

ifneq (,$(wildcard ./VERSION))
	VERSIONLDFLAG := -X github.com/tektoncd/cli/pkg/cmd/version.clientVersion=`cat VERSION`
endif

ifneq ($(RELEASE_VERSION),)
	VERSIONLDFLAG := -X github.com/tektoncd/cli/pkg/cmd/version.clientVersion=$(RELEASE_VERSION)
endif

FLAGS := $(VERSIONLDFLAG)
ifneq ($(SKIPLDFLAG),)
	FLAGS := $(FLAGS) $(SKIPLDFLAG)
endif

ifneq ($(NAMESPACELDFLAG),)
	FLAGS := $(FLAGS) $(NAMESPACELDFLAG)
endif

ifneq ($(FLAGS),)
	LDFLAGS := -ldflags "$(FLAGS)"
endif

all: bin/tkn test

FORCE:

.PHONY: vendor
vendor:
	@$(GO) mod vendor

.PHONY: cross
cross: amd64 386 arm arm64 s390x ppc64le ## build cross platform binaries

.PHONY: amd64
amd64:
	GOOS=linux GOARCH=amd64 go build -mod=vendor $(LDFLAGS) -o bin/tkn-linux-amd64 ./cmd/tkn
	GOOS=windows GOARCH=amd64 go build -mod=vendor $(LDFLAGS) -o bin/tkn-windows-amd64 ./cmd/tkn
	GOOS=darwin GOARCH=amd64 go build -mod=vendor $(LDFLAGS) -o bin/tkn-darwin-amd64 ./cmd/tkn

.PHONY: 386
386:
	GOOS=linux GOARCH=386 go build -mod=vendor $(LDFLAGS) -o bin/tkn-linux-386 ./cmd/tkn
	GOOS=windows GOARCH=386 go build -mod=vendor $(LDFLAGS) -o bin/tkn-windows-386 ./cmd/tkn

.PHONY: arm
arm:
	GOOS=linux GOARCH=arm go build -mod=vendor $(LDFLAGS) -o bin/tkn-linux-arm ./cmd/tkn

.PHONY: arm64
arm64:
	GOOS=linux GOARCH=arm64 go build -mod=vendor $(LDFLAGS) -o bin/tkn-linux-arm64 ./cmd/tkn
	GOOS=darwin GOARCH=arm64 go build -mod=vendor $(LDFLAGS) -o bin/tkn-darwin-arm64 ./cmd/tkn

.PHONY: s390x
s390x:
	GOOS=linux GOARCH=s390x go build -mod=vendor $(LDFLAGS) -o bin/tkn-linux-s390x ./cmd/tkn

.PHONY: ppc64le
ppc64le:
	GOOS=linux GOARCH=ppc64le go build -mod=vendor $(LDFLAGS) -o bin/tkn-linux-ppc64le ./cmd/tkn

bin/%: cmd/% FORCE
	$(Q) $(GO) build -mod=vendor $(LDFLAGS) -v -o $@ ./$<

check: lint test

.PHONY: test
test: test-unit ## run all tests

.PHONY: lint
lint: lint-go goimports lint-yaml  ## run all linters

GOLANGCILINT = $(BIN)/golangci-lint
$(BIN)/golangci-lint: ; $(info $(M) getting golangci-lint $(GOLANGCI_VERSION))
	cd tools; GOBIN=$(BIN) $(GO) install -mod=mod github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_VERSION)

.PHONY: lint-go
lint-go: | $(GOLANGCILINT) ; $(info $(M) running golangci-lint…) @ ## Run golangci-lint
	$Q $(GOLANGCILINT) run --modules-download-mode=vendor --max-issues-per-linter=0 --max-same-issues=0 --timeout 10m
	@rm -f $(GOLANGCILINT)

GOIMPORTS = $(BIN)/goimports
$(GOIMPORTS): ; $(info $(M) getting goimports )
	GOBIN=$(BIN) $(GO) install -mod=mod golang.org/x/tools/cmd/goimports@latest

.PHONY: goimports
goimports: | $(GOIMPORTS) ; $(info $(M) running goimports…) ## Run goimports
	$Q $(GOIMPORTS) -l -e -w pkg cmd test
	@rm -f $(GOIMPORTS)

.PHONY: lint-yaml
lint-yaml: ${YAML_FILES}  ; $(info $(M) running yamllint…) ## runs yamllint on all yaml files
	@yamllint -c .yamllint $(YAML_FILES)

## Tests
TEST_UNIT_TARGETS := test-unit-verbose test-unit-race
test-unit-verbose: ARGS=-v
test-unit-race:    ARGS=-race
$(TEST_UNIT_TARGETS): test-unit
.PHONY: $(TEST_UNIT_TARGETS) test-unit
test-unit: ; $(info $(M) running unit tests…) ## Run unit tests
	$(GO) test -timeout $(TIMEOUT_UNIT) $(ARGS) $(shell go list ./... | grep -v third_party/)

.PHONY: update-golden
update-golden: ./vendor ; $(info $(M) Running unit tests to update golden files…) ## run unit tests (updating golden files)
	@./hack/update-golden.sh

.PHONY: test-e2e
test-e2e: bin/tkn ; $(info $(M) Running e2e tests…) ## run e2e tests
	@LOCAL_CI_RUN=true bash ./test/e2e-tests.sh

.PHONY: docs
docs: bin/docs ; $(info $(M) Generating docs…) ## update docs
	@mkdir -p ./docs/cmd ./docs/man/man1
	@./bin/docs --target=./docs/cmd
	@./bin/docs --target=./docs/man/man1 --kind=man
	@rm -f ./bin/docs

.PHONY: generated
generated: update-golden docs fmt ## generate all files that needs to be generated

.PHONY: clean
clean: ## clean build artifacts
	rm -fR bin VERSION

.PHONY: fmt ## formats the GO code(excludes vendors dir)
fmt: ; $(info $(M) running gofmt…) @ ## Run gofmt on all source files
	$Q $(GO) fmt $(PKGS)

.PHONY: help
help: ## print this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {gsub("\\\\n",sprintf("\n%22c",""), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
