module github.com/tektoncd/cli

go 1.16

require (
	github.com/AlecAivazis/survey/v2 v2.2.12
	github.com/Netflix/go-expect v0.0.0-20200312175327-da48e75238e2
	github.com/blang/semver v3.5.1+incompatible
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/docker/cli v20.10.12+incompatible
	github.com/docker/docker v20.10.12+incompatible
	github.com/fatih/color v1.13.0
	github.com/ghodss/yaml v1.0.0
	github.com/google/go-cmp v0.5.7
	github.com/google/go-containerregistry v0.8.1-0.20220216220642-00c59d91847c
	github.com/hako/durafmt v0.0.0-20191009132224-3f39dc1ed9f4
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c
	github.com/jonboulle/clockwork v0.2.2
	github.com/ktr0731/go-fuzzyfinder v0.2.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.3.0
	github.com/spf13/pflag v1.0.5
	github.com/tektoncd/hub/api v0.0.0-20220128080824-5941f280d06a
	github.com/tektoncd/pipeline v0.33.1
	github.com/tektoncd/plumbing v0.0.0-20211012143332-c7cc43d9bc0c
	github.com/tektoncd/triggers v0.19.0
	github.com/tidwall/gjson v1.12.1 // indirect
	go.opencensus.io v0.23.0
	go.uber.org/multierr v1.7.0
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
	gotest.tools/v3 v3.1.0
	k8s.io/api v0.23.4
	k8s.io/apimachinery v0.23.4
	k8s.io/cli-runtime v0.21.4
	k8s.io/client-go v1.5.2
	knative.dev/pkg v0.0.0-20220131144930-f4b57aef0006
	sigs.k8s.io/yaml v1.3.0
)

replace (
	github.com/google/go-containerregistry => github.com/google/go-containerregistry v0.6.0
	github.com/kr/pty => github.com/creack/pty v1.1.16
	k8s.io/api => k8s.io/api v0.22.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.5
	k8s.io/client-go => k8s.io/client-go v0.22.5
	k8s.io/code-generator => k8s.io/code-generator v0.22.5
)
