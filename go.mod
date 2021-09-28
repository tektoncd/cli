module github.com/tektoncd/cli

go 1.15

require (
	github.com/AlecAivazis/survey/v2 v2.2.12
	github.com/Netflix/go-expect v0.0.0-20200312175327-da48e75238e2
	github.com/blang/semver v3.5.1+incompatible
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/docker/cli v20.10.8+incompatible
	github.com/docker/docker v20.10.8+incompatible
	github.com/fatih/color v1.9.0
	github.com/ghodss/yaml v1.0.0
	github.com/google/go-cmp v0.5.6
	github.com/google/go-containerregistry v0.5.2-0.20210709161016-b448abac9a70
	github.com/hako/durafmt v0.0.0-20191009132224-3f39dc1ed9f4
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c
	github.com/jonboulle/clockwork v0.1.1-0.20190114141812-62fb9bc030d1
	github.com/ktr0731/go-fuzzyfinder v0.2.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/tektoncd/hub/api v0.0.0-20210922050526-aa589697cb58
	github.com/tektoncd/pipeline v0.28.0
	github.com/tektoncd/plumbing v0.0.0-20210514044347-f8a9689d5bd5
	github.com/tektoncd/triggers v0.16.0
	go.opencensus.io v0.23.0
	go.uber.org/multierr v1.7.0
	golang.org/x/term v0.0.0-20210220032956-6a3ed077a48d
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
	gotest.tools/v3 v3.0.3
	k8s.io/api v0.21.4
	k8s.io/apimachinery v0.21.4
	k8s.io/cli-runtime v0.21.4
	k8s.io/client-go v0.21.4
	knative.dev/pkg v0.0.0-20210827184538-2bd91f75571c
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/kr/pty => github.com/creack/pty v1.1.16
	maze.io/x/duration => git.maze.io/go/duration v0.0.0-20160924141736-faac084b6075
)
