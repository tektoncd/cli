module github.com/tektoncd/cli

go 1.25.8

require (
	github.com/ActiveState/vt10x v1.3.1
	github.com/AlecAivazis/survey/v2 v2.3.7
	github.com/Masterminds/semver/v3 v3.4.0
	github.com/Netflix/go-expect v0.0.0-20220104043353-73e0943537d2
	github.com/blang/semver v3.5.1+incompatible
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/creack/pty v1.1.24
	github.com/docker/cli v29.4.3+incompatible
	github.com/docker/docker v28.5.2+incompatible
	github.com/fatih/color v1.19.0
	github.com/go-jose/go-jose/v4 v4.1.4
	github.com/google/go-cmp v0.7.0
	github.com/google/go-containerregistry v0.21.6
	github.com/hako/durafmt v0.0.0-20210608085754-5c1018a4e16b
	github.com/hashicorp/errwrap v1.1.0
	github.com/hashicorp/go-cleanhttp v0.5.2
	github.com/hashicorp/golang-lru v1.0.2
	github.com/hinshun/vt10x v0.0.0-20220228203356-1ab2cad5fd82
	github.com/joho/godotenv v1.5.1
	github.com/jonboulle/clockwork v0.5.0
	github.com/ktr0731/go-fuzzyfinder v0.9.0
	github.com/letsencrypt/boulder v0.20260420.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/sigstore/cosign/v2 v2.6.3
	github.com/sigstore/sigstore v1.10.6
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10
	github.com/spf13/viper v1.21.0
	github.com/stretchr/testify v1.11.1
	github.com/tektoncd/chains v0.26.3
	github.com/tektoncd/pipeline v1.12.0
	github.com/tektoncd/plumbing v0.0.0-20250430145243-3b7cd59879c1
	github.com/tektoncd/triggers v0.35.1-0.20260401091813-1aad8a1898ec
	github.com/theupdateframework/go-tuf v0.7.0
	github.com/titanous/rocacheck v0.0.0-20171023193734-afe73141d399
	go.opentelemetry.io/otel v1.43.0
	go.uber.org/multierr v1.11.0
	go.uber.org/zap v1.28.0
	goa.design/goa/v3 v3.26.0
	golang.org/x/crypto v0.52.0
	golang.org/x/term v0.43.0
	golang.org/x/text v0.37.0
	google.golang.org/grpc v1.81.1
	google.golang.org/protobuf v1.36.11
	gopkg.in/h2non/gock.v1 v1.1.2
	gotest.tools v2.2.0+incompatible
	gotest.tools/v3 v3.5.2
	k8s.io/api v0.35.5
	k8s.io/apimachinery v0.35.5
	k8s.io/cli-runtime v0.29.15
	k8s.io/client-go v0.35.5
	knative.dev/pkg v0.0.0-20260329160701-396dbaacd652
	sigs.k8s.io/yaml v1.6.0
)

replace github.com/alibabacloud-go/cr-20160607 => github.com/vdemeester/cr-20160607 v1.0.1

require (
	cel.dev/expr v0.25.1 // indirect
	cloud.google.com/go v0.123.0 // indirect
	cloud.google.com/go/auth v0.18.2 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	cloud.google.com/go/firestore v1.21.0 // indirect
	cloud.google.com/go/iam v1.5.3 // indirect
	cloud.google.com/go/longrunning v0.8.0 // indirect
	cloud.google.com/go/monitoring v1.24.3 // indirect
	cloud.google.com/go/storage v1.61.3 // indirect
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.21.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.13.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/containers/azcontainerregistry v0.2.3 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.2 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20250102033503-faa5f7b0171c // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.6.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.31.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.55.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.55.0 // indirect
	github.com/IBM/sarama v1.45.2 // indirect
	github.com/ThalesIgnite/crypto11 v1.2.5 // indirect
	github.com/allegro/bigcache/v3 v3.1.0 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-sdk-go v1.55.8 // indirect
	github.com/aws/aws-sdk-go-v2 v1.41.5 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.32.12 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.12 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.20 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecr v1.55.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecrpublic v1.38.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.9 // indirect
	github.com/aws/smithy-go v1.24.2 // indirect
	github.com/awslabs/amazon-ecr-credential-helper/ecr-login v0.12.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blendle/zapdriver v1.3.1 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/clipperhouse/uax29/v2 v2.3.0 // indirect
	github.com/cloudevents/sdk-go/v2 v2.16.2 // indirect
	github.com/cncf/xds/go v0.0.0-20260202195803-dba9d589def2 // indirect
	github.com/coreos/go-oidc/v3 v3.17.0 // indirect
	github.com/cyberphone/json-canonicalization v0.0.0-20241213102144-19d51d7fe467 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/digitorus/pkcs7 v0.0.0-20230818184609-3a137a874352 // indirect
	github.com/digitorus/timestamp v0.0.0-20231217203849-220c5c2851b7 // indirect
	github.com/docker/docker-credential-helpers v0.9.5 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/eapache/go-resiliency v1.7.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20230731223053-c322873962e3 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/edwarnicke/gitoid v0.0.0-20220710194850-1be5bfda1f9d // indirect
	github.com/emicklei/go-restful/v3 v3.12.2 // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.37.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.3.3 // indirect
	github.com/evanphx/json-patch v5.9.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/gaganhr94/docker-credential-acr v1.0.2 // indirect
	github.com/gdamore/encoding v1.0.1 // indirect
	github.com/gdamore/tcell/v2 v2.9.0 // indirect
	github.com/go-chi/chi/v5 v5.2.5 // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/analysis v0.24.3 // indirect
	github.com/go-openapi/errors v0.22.7 // indirect
	github.com/go-openapi/jsonpointer v0.22.5 // indirect
	github.com/go-openapi/jsonreference v0.21.5 // indirect
	github.com/go-openapi/loads v0.23.3 // indirect
	github.com/go-openapi/runtime v0.29.3 // indirect
	github.com/go-openapi/spec v0.22.4 // indirect
	github.com/go-openapi/strfmt v0.26.1 // indirect
	github.com/go-openapi/swag v0.25.5 // indirect
	github.com/go-openapi/swag/cmdutils v0.25.5 // indirect
	github.com/go-openapi/swag/conv v0.25.5 // indirect
	github.com/go-openapi/swag/fileutils v0.25.5 // indirect
	github.com/go-openapi/swag/jsonname v0.25.5 // indirect
	github.com/go-openapi/swag/jsonutils v0.25.5 // indirect
	github.com/go-openapi/swag/loading v0.25.5 // indirect
	github.com/go-openapi/swag/mangling v0.25.5 // indirect
	github.com/go-openapi/swag/netutils v0.25.5 // indirect
	github.com/go-openapi/swag/stringutils v0.25.5 // indirect
	github.com/go-openapi/swag/typeutils v0.25.5 // indirect
	github.com/go-openapi/swag/yamlutils v0.25.5 // indirect
	github.com/go-openapi/validate v0.25.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/cel-go v0.28.0 // indirect
	github.com/google/certificate-transparency-go v1.3.2 // indirect
	github.com/google/gnostic-models v0.7.0 // indirect
	github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20260414223304-7a662782a11f // indirect
	github.com/google/go-containerregistry/pkg/authn/kubernetes v0.0.0-20250225234217-098045d5e61f // indirect
	github.com/google/go-github/v73 v73.0.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/google/wire v0.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.14 // indirect
	github.com/googleapis/gax-go/v2 v2.19.0 // indirect
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674 // indirect
	github.com/grafeas/grafeas v0.2.3 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.28.0 // indirect
	github.com/h2non/parth v0.0.0-20190131123155-b4df798d6542 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.8 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/in-toto/archivista v0.11.1 // indirect
	github.com/in-toto/attestation v1.2.0 // indirect
	github.com/in-toto/go-witness v0.9.2 // indirect
	github.com/in-toto/in-toto-golang v0.11.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.4 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/jedisct1/go-minisign v0.0.0-20230811132847-661be99b8267 // indirect
	github.com/jmespath/go-jmespath v0.4.1-0.20220621161143-b0104c826a24 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/kr/pty v1.1.1 // indirect
	github.com/ktr0731/go-ansisgr v0.1.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/lucasb-eyer/go-colorful v1.3.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/miekg/pkcs11 v1.1.2 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nozzle/throttler v0.0.0-20180817012639-2ea982251481 // indirect
	github.com/nsf/termbox-go v1.1.1 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/otlptranslator v1.0.0 // indirect
	github.com/prometheus/procfs v0.19.2 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20250401214520-65e299d6c5c9 // indirect
	github.com/russross/blackfriday v1.6.0 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sassoftware/relic v7.2.1+incompatible // indirect
	github.com/secure-systems-lab/go-securesystemslib v0.10.0 // indirect
	github.com/shibumi/go-pathspec v1.3.0 // indirect
	github.com/sigstore/protobuf-specs v0.5.0 // indirect
	github.com/sigstore/rekor v1.5.0 // indirect
	github.com/sigstore/rekor-tiles/v2 v2.0.1 // indirect
	github.com/sigstore/sigstore-go v1.1.4 // indirect
	github.com/sigstore/timestamp-authority/v2 v2.0.6 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spiffe/go-spiffe/v2 v2.6.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20220721030215-126854af5e6d // indirect
	github.com/thales-e-security/pool v0.0.2 // indirect
	github.com/theupdateframework/go-tuf/v2 v2.4.1 // indirect
	github.com/tidwall/gjson v1.14.2 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/transparency-dev/formats v0.0.0-20251017110053-404c0d5b696c // indirect
	github.com/transparency-dev/merkle v0.0.2 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/xlab/treeprint v1.2.0 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	gitlab.com/gitlab-org/api/client-go v0.143.3 // indirect
	go.mongodb.org/mongo-driver v1.17.6 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.42.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.63.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.67.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.42.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.42.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.64.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.42.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	go.starlark.net v0.0.0-20230525235612-a134d8f9ddca // indirect
	go.step.sm/crypto v0.77.2 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	gocloud.dev v0.43.0 // indirect
	gocloud.dev/docstore/mongodocstore v0.43.0 // indirect
	gocloud.dev/pubsub/kafkapubsub v0.43.0 // indirect
	golang.org/x/exp v0.0.0-20250620022241-b7579e27df2b // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/net v0.54.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	gomodules.xyz/jsonpatch/v2 v2.5.0 // indirect
	google.golang.org/api v0.272.0 // indirect
	google.golang.org/genproto v0.0.0-20260316180232-0b37fe3546d5 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260401024825-9d38bb4040a9 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260401024825-9d38bb4040a9 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.35.4 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-openapi v0.0.0-20250910181357-589584f1c912 // indirect
	k8s.io/utils v0.0.0-20251002143259-bc988d571ff4 // indirect
	knative.dev/eventing v0.48.1 // indirect
	knative.dev/networking v0.0.0-20231017124814-2a7676e912b7 // indirect
	knative.dev/serving v0.39.4 // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/kustomize/api v0.13.5-0.20230601165947-6ce0bf390ce3 // indirect
	sigs.k8s.io/kustomize/kyaml v0.14.3-0.20230601165947-6ce0bf390ce3 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.3.0 // indirect
)
