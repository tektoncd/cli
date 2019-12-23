module github.com/tektoncd/cli

go 1.13

require (
	contrib.go.opencensus.io/exporter/prometheus v0.1.0 // indirect
	contrib.go.opencensus.io/exporter/stackdriver v0.12.8 // indirect
	github.com/AlecAivazis/survey/v2 v2.0.4
	github.com/Netflix/go-expect v0.0.0-20190729225929-0e00d9168667
	github.com/blang/semver v3.5.1+incompatible
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/fatih/color v1.7.0
	github.com/ghodss/yaml v1.0.0
	github.com/google/go-cmp v0.3.1
	github.com/google/uuid v1.1.1 // indirect
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/hako/durafmt v0.0.0-20180520121703-7b7ae1e72ead
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/jonboulle/clockwork v0.1.1-0.20190114141812-62fb9bc030d1
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/markbates/inflect v1.0.4 // indirect
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/errors v0.8.1
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.3
	github.com/stretchr/testify v1.4.0
	github.com/tektoncd/pipeline v0.9.2
	github.com/tektoncd/plumbing v0.0.0-20191218171343-56a836c50336
	github.com/tektoncd/triggers v0.1.1-0.20191218175743-c5b8b4d9ee00
	gopkg.in/inf.v0 v0.9.1 // indirect
	gotest.tools/v3 v3.0.0
	k8s.io/api v0.0.0-20191004102255-dacd7df5a50b
	k8s.io/apimachinery v0.0.0-20191004074956-01f8b7d1121a
	k8s.io/cli-runtime v0.0.0-20191004110054-fe9b9282443f
	k8s.io/client-go v0.0.0-20191004102537-eb5b9a8cfde7
	k8s.io/klog v1.0.0 // indirect
	k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a // indirect
	knative.dev/caching v0.0.0-20191218191730-996755ee03e4 // indirect
	knative.dev/pkg v0.0.0-20190909195211-528ad1c1dd62
)

replace github.com/kr/pty => github.com/creack/pty v1.1.7

replace github.com/spf13/cobra => github.com/chmouel/cobra v0.0.0-20191021105835-a78788917390

replace git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999
