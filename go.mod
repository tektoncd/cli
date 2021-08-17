module github.com/tektoncd/cli

go 1.13

require (
	github.com/AlecAivazis/survey/v2 v2.2.12
	github.com/Netflix/go-expect v0.0.0-20200312175327-da48e75238e2
	github.com/blang/semver v3.5.1+incompatible
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/docker/cli v20.10.2+incompatible
	github.com/docker/docker v20.10.2+incompatible
	github.com/fatih/color v1.9.0
	github.com/ghodss/yaml v1.0.0
	github.com/google/go-cmp v0.5.6
	github.com/google/go-containerregistry v0.4.1-0.20210128200529-19c2b639fab1
	github.com/hako/durafmt v0.0.0-20191009132224-3f39dc1ed9f4
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c
	github.com/jonboulle/clockwork v0.1.1-0.20190114141812-62fb9bc030d1
	github.com/ktr0731/go-fuzzyfinder v0.2.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/tektoncd/hub/api v0.0.0-20210707072808-0ae1afc58dd8
	github.com/tektoncd/pipeline v0.27.1
	github.com/tektoncd/plumbing v0.0.0-20210514044347-f8a9689d5bd5
	github.com/tektoncd/triggers v0.14.2
	github.com/tidwall/gjson v1.6.0 // indirect
	go.opencensus.io v0.23.0
	go.uber.org/multierr v1.6.0
	golang.org/x/term v0.0.0-20201210144234-2321bbc49cbf
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
	gotest.tools/v3 v3.0.2
	k8s.io/api v0.20.7
	k8s.io/apimachinery v0.20.7
	k8s.io/cli-runtime v0.20.7
	k8s.io/client-go v0.20.7
	knative.dev/pkg v0.0.0-20210730172132-bb4aaf09c430
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// Needed until kustomize is updated in the k8s repos:
	// https://github.com/kubernetes-sigs/kustomize/issues/1500
	github.com/go-openapi/spec => github.com/go-openapi/spec v0.19.3
	github.com/kr/pty => github.com/creack/pty v1.1.10
	maze.io/x/duration => git.maze.io/go/duration v0.0.0-20160924141736-faac084b6075
)
