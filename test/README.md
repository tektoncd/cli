# Tests

To run tests:

```shell
# Unit tests
go test ./...

# Integration tests (against your current kube cluster)
go test -v -count=1 -tags=e2e ./test
```

## Unit tests

Unit tests live side by side with the code they are testing and can be run with:

```shell
go test ./...
```

By default `go test` will not run [the end to end tests](#end-to-end-tests),
which need `-tags=e2e` to be enabled.


## End to end tests

To run end to end tests, you will need to have a running Tekton
pipeline deployed on your cluster, see [Install Pipeline](../DEVELOPMENT.md#install-pipeline)

End to end tests live in e2e directory. To run these tests, you must provide
`go` with `-tags=e2e`. By default the tests run against your current kubeconfig
context, but you can change that and other settings with [the flags](#flags):

```shell
go test -v -count=1 -tags=e2e -timeout=20m github.com/tektoncd/cli/test/e2e 
go test -v -count=1 -tags=e2e -timeout=20m github.com/tektoncd/cli/test/e2e  --kubeconfig ~/special/kubeconfig --cluster myspecialcluster
```
Its Runs all tests in parallel and each test runs on its own namespace(testcli-*)

To Generate junit.xml file at Runtime 

## Installation

Go version 1.1 or higher is required. Install or update using the `go get`
command:

```bash
go get -u github.com/jstemmer/go-junit-report
```
Command to Execute to generate .xml file at runtime

```shell
go test -v -count=1 -tags=e2e -timeout=20m github.com/tektoncd/cli/test/e2e 2>&1 |  go-junit-report > <path_to_create_junit.xml>/junit.xml

```

You can also use
[all of flags defined in `knative/pkg/test`](https://github.com/knative/pkg/tree/master/test#flags).


### Flags

- By default the e2e tests against the current cluster in `~/.kube/config` using
  the environment specified in
  [your environment variables](/DEVELOPMENT.md#environment-setup).
- Since these tests are fairly slow, running them with logging enabled is
  recommended (`-v`).
- Using [`--logverbose`](#output-verbose-log) to see the verbose log output from
  test as well as from k8s libraries.
- Using `-count=1` is
  [the idiomatic way to disable test caching](https://golang.org/doc/go1.10#test).
- The end to end tests take a long time to run so a value like `-timeout=20m`
  can be useful depending on what you're running

You can [use test flags](#flags) to control the environment your tests run
against, i.e. override
[your environment variables](/DEVELOPMENT.md#environment-setup):

```bash
go test -v -tags=e2e -count=1 ./test --kubeconfig ~/special/kubeconfig --cluster myspecialcluster
```

### One test case

To run one e2e test case, e.g. TestTaskRun, use
[the `-run` flag with `go test`](https://golang.org/cmd/go/#hdr-Testing_flags):

```shell
go test -v -count=1 -tags=e2e -timeout=20m github.com/tektoncd/cli/test/e2e -run ^TestTaskRunListe2eUsingCli$
go test -v -count=1 -tags=e2e -timeout=20m github.com/tektoncd/cli/test/e2e  --kubeconfig ~/special/kubeconfig --cluster myspecialcluster -run ^TestTaskRunListe2eUsingCli$
```

### Smoke test case

To run end to end tests, you will need to have a running Tekton
pipeline deployed on your cluster, see [Install Pipeline](../DEVELOPMENT.md#install-pipeline)

End to end tests live in e2e directory. To run these tests, you must provide
`go` with `-tags=smoke`. By default the tests run against your current kubeconfig
context, but you can change that and other settings with [the flags](#flags):

```shell
go test -v -count=1 -tags=smoke -timeout=20m github.com/tektoncd/cli/test/e2e 
go test -v -count=1 -tags=smoke -timeout=20m github.com/tektoncd/cli/test/e2e  --kubeconfig ~/special/kubeconfig --cluster myspecialcluster
```

