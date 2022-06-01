module github.com/tektoncd/cli

go 1.16

require (
	github.com/AlecAivazis/survey/v2 v2.3.2
	github.com/Netflix/go-expect v0.0.0-20200312175327-da48e75238e2
	github.com/armon/go-metrics v0.3.11
	github.com/armon/go-radix v1.0.0
	github.com/blang/semver v3.5.1+incompatible
	github.com/containerd/containerd v1.6.2 // indirect
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/docker/cli v20.10.14+incompatible
	github.com/docker/docker v20.10.14+incompatible
	github.com/fatih/color v1.13.0
	github.com/ghodss/yaml v1.0.0
	github.com/golang/snappy v0.0.4
	github.com/google/go-cmp v0.5.8
	github.com/google/go-containerregistry v0.8.1-0.20220414143355-892d7a808387
	github.com/hako/durafmt v0.0.0-20210608085754-5c1018a4e16b
	github.com/hashicorp/errwrap v1.1.0
	github.com/hashicorp/go-hclog v1.2.0
	github.com/hashicorp/go-immutable-radix v1.3.1
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-plugin v1.4.3
	github.com/hashicorp/go-secure-stdlib/mlock v0.1.2
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2
	github.com/hashicorp/go-sockaddr v1.0.2
	github.com/hashicorp/go-uuid v1.0.3
	github.com/hashicorp/go-version v1.4.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/hashicorp/hcl v1.0.0
	github.com/hashicorp/vault/sdk v0.4.1
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c
	github.com/jonboulle/clockwork v0.2.2
	github.com/ktr0731/go-fuzzyfinder v0.2.0
	github.com/letsencrypt/boulder v0.0.0-20220331220046-b23ab962616e
	github.com/mitchellh/copystructure v1.2.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/go-testing-interface v1.14.1
	github.com/mitchellh/mapstructure v1.5.0
	github.com/pierrec/lz4 v2.6.1+incompatible
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.4.0
	github.com/spf13/pflag v1.0.5
	github.com/tektoncd/chains v0.9.0
	github.com/tektoncd/hub v1.7.0
	github.com/tektoncd/pipeline v0.35.0
	github.com/tektoncd/plumbing v0.0.0-20220329085922-d765a5cba75f
	github.com/tektoncd/triggers v0.20.0
	github.com/titanous/rocacheck v0.0.0-20171023193734-afe73141d399
	go.opencensus.io v0.23.0
	go.uber.org/atomic v1.9.0
	go.uber.org/multierr v1.8.0
	go.uber.org/zap v1.21.0
	golang.org/x/crypto v0.0.0-20220411220226-7b82a4e95df4
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
	golang.org/x/xerrors v0.0.0-20220411194840-2f41105eb62f
	google.golang.org/grpc v1.46.0
	google.golang.org/protobuf v1.28.0
	gopkg.in/square/go-jose.v2 v2.6.0
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
	gotest.tools/v3 v3.1.0
	k8s.io/api v0.23.5
	k8s.io/apimachinery v0.23.5
	k8s.io/cli-runtime v0.23.5
	k8s.io/client-go v1.5.2
	knative.dev/pkg v0.0.0-20220329144915-0a1ec2e0d46c
	sigs.k8s.io/yaml v1.3.0
)

replace (
	github.com/kr/pty => github.com/creack/pty v1.1.16
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc => go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.20.0
	go.opentelemetry.io/otel => go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/sdk => go.opentelemetry.io/otel/sdk v0.20.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.23.5
	k8s.io/client-go => k8s.io/client-go v0.23.5
	k8s.io/code-generator => k8s.io/code-generator v0.22.5
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20211109043538-20434351676c
)
