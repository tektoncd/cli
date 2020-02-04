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
	github.com/google/go-cmp v0.3.1
	github.com/google/go-github v17.0.0+incompatible // indirect
	github.com/google/go-licenses v0.0.0-20191220124820-2ee7a02f6ae4 // indirect
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/hako/durafmt v0.0.0-20180520121703-7b7ae1e72ead
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c
	github.com/jonboulle/clockwork v0.1.1-0.20190114141812-62fb9bc030d1
	github.com/mattn/go-isatty v0.0.9 // indirect
	github.com/pkg/errors v0.8.1
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/tektoncd/pipeline v0.10.1
	github.com/tektoncd/plumbing v0.0.0-20191218171343-56a836c50336
	github.com/tektoncd/triggers v0.2.1-preview.0.20200130181252-68f571334722
	go.uber.org/multierr v1.1.0
	golang.org/x/crypto v0.0.0-20191206172530-e9b2fee46413
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gotest.tools/v3 v3.0.0
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.1
	k8s.io/cli-runtime v0.0.0-20191004110054-fe9b9282443f
	k8s.io/client-go v0.17.0
	knative.dev/pkg v0.0.0-20191111150521-6d806b998379
)

replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.12.0
	git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999
	github.com/kr/pty => github.com/creack/pty v1.1.9
	github.com/spf13/cobra => github.com/chmouel/cobra v0.0.0-20200107083527-379e7a80af0c
	knative.dev/pkg => knative.dev/pkg v0.0.0-20190909195211-528ad1c1dd62
	knative.dev/pkg/vendor/github.com/spf13/pflag => github.com/spf13/pflag v1.0.5
)

// Pin k8s deps to 1.13.4

replace (
	k8s.io/api => k8s.io/api v0.0.0-20191004102255-dacd7df5a50b
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004074956-01f8b7d1121a
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191004102537-eb5b9a8cfde7
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20181117043124-c2090bec4d9b
	k8s.io/gengo => k8s.io/gengo v0.0.0-20190327210449-e17681d19d3a
)
