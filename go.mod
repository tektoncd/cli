module github.com/tektoncd/cli

go 1.13

require (
	github.com/AlecAivazis/survey/v2 v2.0.4
	github.com/Netflix/go-expect v0.0.0-20190729225929-0e00d9168667
	github.com/blang/semver v3.5.1+incompatible
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/fatih/color v1.7.0
	github.com/ghodss/yaml v1.0.0
	github.com/google/cel-go v0.3.2 // indirect
	github.com/google/go-cmp v0.4.0
	github.com/google/go-github v17.0.0+incompatible // indirect
	github.com/hako/durafmt v0.0.0-20191009132224-3f39dc1ed9f4
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c
	github.com/jonboulle/clockwork v0.1.1-0.20190114141812-62fb9bc030d1
	github.com/ktr0731/go-fuzzyfinder v0.2.0
	github.com/mattn/go-isatty v0.0.9 // indirect
	github.com/pkg/errors v0.8.1
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/tektoncd/pipeline v0.11.3
	github.com/tektoncd/plumbing v0.0.0-20200430135134-e53521e1d887
	github.com/tektoncd/triggers v0.4.0
	github.com/tidwall/gjson v1.6.0 // indirect
	github.com/tidwall/sjson v1.0.4 // indirect
	go.opencensus.io v0.22.1
	go.uber.org/multierr v1.4.0
	golang.org/x/crypto v0.0.0-20191206172530-e9b2fee46413
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
	gopkg.in/yaml.v2 v2.2.5
	gotest.tools/v3 v3.0.1
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/cli-runtime v0.17.0
	k8s.io/client-go v0.17.0
	knative.dev/pkg v0.0.0-20200207155214-fef852970f43
	sigs.k8s.io/yaml v1.1.0
)

replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.12.9-0.20191108183826-59d068f8d8ff
	github.com/kr/pty => github.com/creack/pty v1.1.9
	github.com/spf13/cobra => github.com/chmouel/cobra v0.0.0-20200107083527-379e7a80af0c
	knative.dev/pkg => knative.dev/pkg v0.0.0-20200113182502-b8dc5fbc6d2f
)

// Pin k8s deps to 1.16.7

replace (
	k8s.io/api => k8s.io/api v0.16.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.5
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.16.5
	k8s.io/client-go => k8s.io/client-go v0.16.5
	k8s.io/code-generator => k8s.io/code-generator v0.16.5
	k8s.io/gengo => k8s.io/gengo v0.0.0-20190327210449-e17681d19d3a
)
