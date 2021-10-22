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
	@go mod vendor

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
	go build -mod=vendor $(LDFLAGS) -v -o $@ ./$<

check: lint test

.PHONY: test
test: test-unit ## run all tests

.PHONY: lint
lint: lint-go lint-yaml ## run all linters

.PHONY: lint-go
lint-go: ## runs go linter on all go files
	@echo "Linting go files..."
	@golangci-lint run ./... --modules-download-mode=vendor \
							--max-issues-per-linter=0 \
							--max-same-issues=0 \
							--deadline 5m

.PHONY: lint-yaml
lint-yaml: ${YAML_FILES} ## runs yamllint on all yaml files
	@echo "Linting yaml files..."
	@yamllint -c .yamllint $(YAML_FILES)

.PHONY: test-unit
test-unit: ./vendor ## run unit tests
	@echo "Running unit tests..."
	@go test -failfast -v -cover ./...

.PHONY: update-golden
update-golden: ./vendor ## run unit tests (updating golden files)
	@echo "Running unit tests to update golden files..."
	@./hack/update-golden.sh

.PHONY: test-e2e
test-e2e: bin/tkn ## run e2e tests
	@echo "Running e2e tests..."
	@LOCAL_CI_RUN=true bash ./test/e2e-tests.sh

.PHONY: docs
docs: bin/docs ## update docs
	@echo "Generating docs"
	@./bin/docs --target=./docs/cmd
	@./bin/docs --target=./docs/man/man1 --kind=man
	@rm -f ./bin/docs

.PHONY: generated
generated: update-golden docs fmt ## generate all files that needs to be generated

.PHONY: clean
clean: ## clean build artifacts
	rm -fR bin VERSION

.PHONY: fmt ## formats the GO code(excludes vendors dir)
fmt:
	@go fmt `go list ./... | grep -v /vendor/`

.PHONY: help
help: ## print this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {gsub("\\\\n",sprintf("\n%22c",""), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
