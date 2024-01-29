# Tests

## Unit tests

Unit tests live side by side with the code they are testing and can be run with:

```shell
go test $(go list ./... | grep -v third_party/)
# or
make test-unit
```

## End to end tests

### Some prerequisites to run tests:

1. Spin up a kind/minikube cluster.

2. To run end to end tests, you will need to have a running Tekton pipeline deployed on your cluster, see [Install Pipeline](../DEVELOPMENT.md#install-pipeline).

3. Next install Tekton Triggers in the cluster, see [Install Triggers](https://github.com/tektoncd/triggers/blob/main/docs/install.md#installation)

Set environment variable `TEST_CLUSTERTASK_LIST_EMPTY` to any value if tests are run in an environment which contains `clustertasks`. 

```shell
export TEST_CLUSTERTASK_LIST_EMPTY=empty
```

Set `SYSTEM_NAMESPACE` variable to appropriate value (say tekton-pipelines) based on the namespace where Tekton Pipelines system components reside within the Kubernetes cluster.

```shell 
export SYSTEM_NAMESPACE=<namespace-name> (eg: export SYSTEM_NAMESPACE=tekton-pipelines)
```

### Running

End to end tests live in [this](../test/e2e/) directory. By default `go test` will not run [the end to end tests](#end-to-end-tests),
it need `-tags=e2e` to be enabled in order to run these tests, hence you must provide `go` with `-tags=e2e`. 

```shell
go build -o tkn github.com/tektoncd/cli/cmd/tkn
export TEST_CLIENT_BINARY=<path-to-tkn-binary-directory>/tkn ( eg: export TEST_CLIENT_BINARY=$PWD/tkn ) 
```
By default the tests run against your current kubeconfig context:

```shell
go test -v -count=1 -tags=e2e -timeout=20m ./test/e2e/...
```
You can change the kubeconfig context and other settings with [the flags](#flags).

```shell
go test -v -count=1 -tags=e2e -timeout=20m ./test/e2e/... --kubeconfig ~/special/kubeconfig --cluster myspecialcluster
```

You can also use [all of flags defined in `knative/pkg/test`](https://github.com/knative/pkg/tree/master/test#flags).

### Flags

- By default the e2e tests against the current cluster in `~/.kube/config` using
  the environment specified in
  [your environment variables](/DEVELOPMENT.md#environment-setup).
- Since these tests are fairly slow, running them with logging enabled is
  recommended (`-v`).
- Using [`--logverbose`](#output-verbose-log) to see the verbose log output from
  test as well as from k8s libraries.
- Using `-count=1` is
  the idiomatic way to disable test caching
- The end to end tests take a long time to run so a value like `-timeout=20m`
  can be useful depending on what you're running

You can [use test flags](#flags) to control the environment your tests run
against, i.e. override
[your environment variables](/DEVELOPMENT.md#environment-setup):

```bash
go test -v -count=1 ./test/e2e/... --kubeconfig ~/special/kubeconfig --cluster myspecialcluster
```

### One test case

To run one e2e test case, e.g. TestPipelinesE2E test, use
the `-run` flag with `go test`

```bash
go test -v ./test/... -tags=e2e -run ^TestPipelinesE2E
```

### Prerequisite to run make check and make generated 

Ensure that golangci-lint and yamllint are installed on your development machine. If not then install them following [golangci-lint](https://golangci-lint.run/usage/install/) and [yamllint](https://yamllint.readthedocs.io/en/stable/quickstart.html).

Next step will be to run
```bash
# For linting the golang and YAML files 
make check
# To check diff in goldenfile and [docs](https://github.com/tektoncd/cli/tree/main/docs), update and autogenerate them
make generated
```