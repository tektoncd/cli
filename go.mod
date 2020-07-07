module github.com/tektoncd/cli

go 1.13

require (
	github.com/AlecAivazis/survey/v2 v2.0.4
	github.com/Netflix/go-expect v0.0.0-20200312175327-da48e75238e2
	github.com/blang/semver v3.5.1+incompatible
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/fatih/color v1.9.0
	github.com/ghodss/yaml v1.0.0
	github.com/google/go-cmp v0.4.1
	github.com/hako/durafmt v0.0.0-20191009132224-3f39dc1ed9f4
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c
	github.com/jonboulle/clockwork v0.1.1-0.20190114141812-62fb9bc030d1
	github.com/ktr0731/go-fuzzyfinder v0.2.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v0.0.6
	github.com/spf13/pflag v1.0.5
	github.com/tektoncd/pipeline v0.14.0
	github.com/tektoncd/plumbing v0.0.0-20200430135134-e53521e1d887
	github.com/tektoncd/triggers v0.5.0
	github.com/tidwall/gjson v1.6.0 // indirect
	go.opencensus.io v0.22.3
	go.uber.org/multierr v1.5.0
	golang.org/x/crypto v0.0.0-20200323165209-0ec3e9974c59
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
	gopkg.in/yaml.v2 v2.3.0
	gotest.tools/v3 v3.0.1
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/cli-runtime v0.17.3
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	knative.dev/pkg v0.0.0-20200528142800-1c6815d7e4c9
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/kr/pty => github.com/creack/pty v1.1.10
	github.com/spf13/cobra => github.com/chmouel/cobra v0.0.0-20200107083527-379e7a80af0c
)

// Knative deps (release-0.15)
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.12.9-0.20191108183826-59d068f8d8ff
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.4.0+incompatible
	knative.dev/caching => knative.dev/caching v0.0.0-20200521155757-e78d17bc250e
	knative.dev/pkg => knative.dev/pkg v0.0.0-20200528142800-1c6815d7e4c9
)

// Pin k8s deps to 1.16.5
replace (
	k8s.io/api => k8s.io/api v0.16.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.5
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.16.5
	k8s.io/client-go => k8s.io/client-go v0.16.5
	k8s.io/code-generator => k8s.io/code-generator v0.16.5
)
