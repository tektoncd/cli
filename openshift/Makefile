DETECT_RELEASE_YAML=$(shell bash pipeline-latest-release.sh)
STABLE_RELEASE_YAML=$(shell bash pipeline-latest-release.sh --only-stable-release)
RELEASE_YAML=
# Turn the modules on as we're using `-mod=vendor` flag while building the binaries
# and the source code is cloned under $GOPATH
export GO111MODULE=on

# Temporary hack to use the stable release if nightly doesn't exist, in case release fails
test-e2e-downstream-nightly:
	make test-e2e-downstream RELEASE_YAML=$(DETECT_RELEASE_YAML)
.PHONY: test-e2e-downstream-nightly

test-e2e-downstream-stable:
	@make test-e2e-downstream RELEASE_YAML=$(STABLE_RELEASE_YAML)
.PHONY: test-e2e-downstream-stable

test-e2e-downstream:
	@make -C ../ bin/tkn
	@env RELEASE_YAML=$(RELEASE_YAML) LOCAL_CI_RUN=true ../test/e2e-tests.sh
.PHONY: test-e2e-downstream
