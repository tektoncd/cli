/*
Copyright 2021 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sigstore/sigstore/pkg/tuf"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	cm "knative.dev/pkg/configmap"
)

type Config struct {
	Artifacts       ArtifactConfigs
	Storage         StorageConfigs
	Signers         SignerConfigs
	Builder         BuilderConfig
	Transparency    TransparencyConfig
	BuildDefinition BuildDefinitionConfig
}

// ArtifactConfigs contains the configuration for how to sign/store/format the signatures for each artifact type
type ArtifactConfigs struct {
	OCI          Artifact
	PipelineRuns Artifact
	TaskRuns     Artifact
}

// Artifact contains the configuration for how to sign/store/format the signatures for a single artifact
type Artifact struct {
	Format                string
	StorageBackend        sets.Set[string]
	Signer                string
	DeepInspectionEnabled bool
}

// StorageConfigs contains the configuration to instantiate different storage providers
type StorageConfigs struct {
	GCS        GCSStorageConfig
	OCI        OCIStorageConfig
	Tekton     TektonStorageConfig
	DocDB      DocDBStorageConfig
	Grafeas    GrafeasConfig
	PubSub     PubSubStorageConfig
	Archivista ArchivistaStorageConfig
}

// SignerConfigs contains the configuration to instantiate different signers
type SignerConfigs struct {
	X509 X509Signer
	KMS  KMSSigner
}

type BuilderConfig struct {
	ID string
}

type BuildDefinitionConfig struct {
	BuildType string
}

type X509Signer struct {
	FulcioEnabled     bool
	FulcioAddr        string
	FulcioOIDCIssuer  string
	FulcioProvider    string
	IdentityTokenFile string
	TUFMirrorURL      string
}

type KMSSigner struct {
	KMSRef string
	Auth   KMSAuth
}

// KMSAuth configures authentication to the KMS server
type KMSAuth struct {
	Address   string
	Token     string
	TokenPath string
	OIDC      KMSAuthOIDC
	Spire     KMSAuthSpire
}

// KMSAuthOIDC configures settings to authenticate with OIDC
type KMSAuthOIDC struct {
	Path string
	Role string
}

// KMSAuthSpire configures settings to get an auth token from spire
type KMSAuthSpire struct {
	Sock     string
	Audience string
}

type GCSStorageConfig struct {
	Bucket string
}

type OCIStorageConfig struct {
	Repository string
	Insecure   bool
}

type TektonStorageConfig struct {
}

type DocDBStorageConfig struct {
	URL                string
	MongoServerURL     string
	MongoServerURLDir  string
	MongoServerURLPath string
}

type GrafeasConfig struct {
	// project id that is used to store notes and occurences
	ProjectID string
	// note id used to create a note that an occurrence will be attached to
	NoteID string

	// NoteHint is used to set the attestation note
	NoteHint string
}

type PubSubStorageConfig struct {
	Provider string
	Topic    string
	Kafka    KafkaStorageConfig
}

type KafkaStorageConfig struct {
	BootstrapServers string
}

type TransparencyConfig struct {
	Enabled          bool
	VerifyAnnotation bool
	URL              string
}

// ArchivistaStorageConfig holds configuration for the Archivista storage backend.
type ArchivistaStorageConfig struct {
	// URL is the endpoint for the Archivista service.
	URL string `json:"url"`
}

const (
	taskrunFormatKey  = "artifacts.taskrun.format"
	taskrunStorageKey = "artifacts.taskrun.storage"
	taskrunSignerKey  = "artifacts.taskrun.signer"

	pipelinerunFormatKey               = "artifacts.pipelinerun.format"
	pipelinerunStorageKey              = "artifacts.pipelinerun.storage"
	pipelinerunSignerKey               = "artifacts.pipelinerun.signer"
	pipelinerunEnableDeepInspectionKey = "artifacts.pipelinerun.enable-deep-inspection"

	ociFormatKey  = "artifacts.oci.format"
	ociStorageKey = "artifacts.oci.storage"
	ociSignerKey  = "artifacts.oci.signer"

	gcsBucketKey               = "storage.gcs.bucket"
	ociRepositoryKey           = "storage.oci.repository"
	ociRepositoryInsecureKey   = "storage.oci.repository.insecure"
	docDBUrlKey                = "storage.docdb.url"
	docDBMongoServerURLKey     = "storage.docdb.mongo-server-url"
	docDBMongoServerURLDirKey  = "storage.docdb.mongo-server-url-dir"
	docDBMongoServerURLPathKey = "storage.docdb.mongo-server-url-path"

	archivistaURLKey = "storage.archivista.url"

	grafeasProjectIDKey = "storage.grafeas.projectid"
	grafeasNoteIDKey    = "storage.grafeas.noteid"
	grafeasNoteHint     = "storage.grafeas.notehint"

	// PubSub - General
	pubsubProvider = "storage.pubsub.provider"
	pubsubTopic    = "storage.pubsub.topic"

	// No config for PubSub - In-Memory

	// PubSub - Kafka
	pubsubKafkaBootstrapServer = "storage.pubsub.kafka.bootstrap.servers"

	// KMS
	kmsSignerKMSRef      = "signers.kms.kmsref"
	kmsAuthAddress       = "signers.kms.auth.address"
	kmsAuthToken         = "signers.kms.auth.token"
	kmsAuthOIDCPath      = "signers.kms.auth.oidc.path"
	kmsAuthTokenPath     = "signers.kms.auth.token-path" // #nosec G101
	kmsAuthOIDCRole      = "signers.kms.auth.oidc.role"
	kmsAuthSpireSock     = "signers.kms.auth.spire.sock"
	kmsAuthSpireAudience = "signers.kms.auth.spire.audience"

	// Fulcio
	x509SignerFulcioEnabled     = "signers.x509.fulcio.enabled"
	x509SignerFulcioAddr        = "signers.x509.fulcio.address"
	x509SignerFulcioOIDCIssuer  = "signers.x509.fulcio.issuer"
	x509SignerFulcioProvider    = "signers.x509.fulcio.provider"
	x509SignerIdentityTokenFile = "signers.x509.identity.token.file"
	x509SignerTUFMirrorURL      = "signers.x509.tuf.mirror.url"

	// Builder config
	builderIDKey = "builder.id"

	transparencyEnabledKey = "transparency.enabled"
	transparencyURLKey     = "transparency.url"

	// Build type
	buildTypeKey = "builddefinition.buildtype"

	ChainsConfig = "chains-config"
)

func (artifact *Artifact) Enabled() bool {
	// If signer is "none", signing is disabled
	if artifact.Signer == "none" {
		return false
	}
	return !(artifact.StorageBackend.Len() == 1 && artifact.StorageBackend.Has(""))
}

func defaultConfig() *Config {
	return &Config{
		Artifacts: ArtifactConfigs{
			TaskRuns: Artifact{
				Format:         "in-toto",
				StorageBackend: sets.New[string]("tekton"),
				Signer:         "x509",
			},
			PipelineRuns: Artifact{
				Format:                "in-toto",
				StorageBackend:        sets.New[string]("tekton"),
				Signer:                "x509",
				DeepInspectionEnabled: false,
			},
			OCI: Artifact{
				Format:         "simplesigning",
				StorageBackend: sets.New[string]("oci"),
				Signer:         "x509",
			},
		},
		Transparency: TransparencyConfig{
			URL: "https://rekor.sigstore.dev",
		},
		Signers: SignerConfigs{
			X509: X509Signer{
				FulcioAddr:       "https://fulcio.sigstore.dev",
				FulcioOIDCIssuer: "https://oauth2.sigstore.dev/auth",
				TUFMirrorURL:     tuf.DefaultRemoteRoot,
			},
		},
		Storage: StorageConfigs{
			Grafeas: GrafeasConfig{
				NoteHint: "This attestation note was generated by Tekton Chains",
			},
		},
		Builder: BuilderConfig{
			ID: "https://tekton.dev/chains/v2",
		},
		BuildDefinition: BuildDefinitionConfig{
			BuildType: "https://tekton.dev/chains/v2/slsa",
		},
	}
}

// NewConfigFromMap creates a Config from the supplied map
func NewConfigFromMap(data map[string]string) (*Config, error) {
	cfg := defaultConfig()

	if err := cm.Parse(data,
		// Artifact-specific configs
		// TaskRuns
		asString(taskrunFormatKey, &cfg.Artifacts.TaskRuns.Format, "in-toto", "slsa/v1", "slsa/v2alpha3", "slsa/v2alpha4"),
		asStringSet(taskrunStorageKey, &cfg.Artifacts.TaskRuns.StorageBackend, sets.New[string]("tekton", "oci", "gcs", "docdb", "grafeas", "kafka", "archivista")),
		asString(taskrunSignerKey, &cfg.Artifacts.TaskRuns.Signer, "x509", "kms", "none"),

		// PipelineRuns
		asString(pipelinerunFormatKey, &cfg.Artifacts.PipelineRuns.Format, "in-toto", "slsa/v1", "slsa/v2alpha3", "slsa/v2alpha4"),
		asStringSet(pipelinerunStorageKey, &cfg.Artifacts.PipelineRuns.StorageBackend, sets.New[string]("tekton", "oci", "gcs", "docdb", "grafeas", "archivista")),
		asString(pipelinerunSignerKey, &cfg.Artifacts.PipelineRuns.Signer, "x509", "kms", "none"),
		asBool(pipelinerunEnableDeepInspectionKey, &cfg.Artifacts.PipelineRuns.DeepInspectionEnabled),

		// OCI
		asString(ociFormatKey, &cfg.Artifacts.OCI.Format, "simplesigning"),
		asStringSet(ociStorageKey, &cfg.Artifacts.OCI.StorageBackend, sets.New[string]("tekton", "oci", "gcs", "docdb", "grafeas", "kafka", "archivista")),
		asString(ociSignerKey, &cfg.Artifacts.OCI.Signer, "x509", "kms", "none"),

		// PubSub - General
		asString(pubsubProvider, &cfg.Storage.PubSub.Provider, "inmemory", "kafka"),
		asString(pubsubTopic, &cfg.Storage.PubSub.Topic),

		// PubSub - Kafka
		asString(pubsubKafkaBootstrapServer, &cfg.Storage.PubSub.Kafka.BootstrapServers),

		// Storage level configs
		asString(gcsBucketKey, &cfg.Storage.GCS.Bucket),
		asString(ociRepositoryKey, &cfg.Storage.OCI.Repository),
		asBool(ociRepositoryInsecureKey, &cfg.Storage.OCI.Insecure),
		asString(docDBUrlKey, &cfg.Storage.DocDB.URL),
		asString(docDBMongoServerURLKey, &cfg.Storage.DocDB.MongoServerURL),
		asString(docDBMongoServerURLDirKey, &cfg.Storage.DocDB.MongoServerURLDir),
		asString(docDBMongoServerURLPathKey, &cfg.Storage.DocDB.MongoServerURLPath),

		asString(archivistaURLKey, &cfg.Storage.Archivista.URL),

		asString(grafeasProjectIDKey, &cfg.Storage.Grafeas.ProjectID),
		asString(grafeasNoteIDKey, &cfg.Storage.Grafeas.NoteID),
		asString(grafeasNoteHint, &cfg.Storage.Grafeas.NoteHint),

		oneOf(transparencyEnabledKey, &cfg.Transparency.Enabled, "true", "manual"),
		oneOf(transparencyEnabledKey, &cfg.Transparency.VerifyAnnotation, "manual"),
		asString(transparencyURLKey, &cfg.Transparency.URL),

		asString(kmsSignerKMSRef, &cfg.Signers.KMS.KMSRef),
		asString(kmsAuthAddress, &cfg.Signers.KMS.Auth.Address),
		asString(kmsAuthToken, &cfg.Signers.KMS.Auth.Token),
		asString(kmsAuthTokenPath, &cfg.Signers.KMS.Auth.TokenPath),
		asString(kmsAuthOIDCPath, &cfg.Signers.KMS.Auth.OIDC.Path),
		asString(kmsAuthOIDCRole, &cfg.Signers.KMS.Auth.OIDC.Role),
		asString(kmsAuthSpireSock, &cfg.Signers.KMS.Auth.Spire.Sock),
		asString(kmsAuthSpireAudience, &cfg.Signers.KMS.Auth.Spire.Audience),

		// Fulcio
		asBool(x509SignerFulcioEnabled, &cfg.Signers.X509.FulcioEnabled),
		asString(x509SignerFulcioAddr, &cfg.Signers.X509.FulcioAddr),
		asString(x509SignerFulcioOIDCIssuer, &cfg.Signers.X509.FulcioOIDCIssuer),
		asString(x509SignerFulcioProvider, &cfg.Signers.X509.FulcioProvider),
		asString(x509SignerIdentityTokenFile, &cfg.Signers.X509.IdentityTokenFile),
		asString(x509SignerTUFMirrorURL, &cfg.Signers.X509.TUFMirrorURL),

		// Build config
		asString(builderIDKey, &cfg.Builder.ID),

		// Build type
		asString(buildTypeKey, &cfg.BuildDefinition.BuildType, "https://tekton.dev/chains/v2/slsa", "https://tekton.dev/chains/v2/slsa-tekton"),
	); err != nil {
		return nil, fmt.Errorf("failed to parse data: %w", err)
	}

	return cfg, nil
}

// NewConfigFromConfigMap creates a Config from the supplied ConfigMap
func NewConfigFromConfigMap(configMap *corev1.ConfigMap) (*Config, error) {
	return NewConfigFromMap(configMap.Data)
}

// oneOf sets target to true if it maches any of the values
func oneOf(key string, target *bool, values ...string) cm.ParseFunc {
	return func(data map[string]string) error {
		raw, ok := data[key]
		if !ok {
			return nil
		}
		if values == nil {
			return nil
		}
		for _, v := range values {
			if v == raw {
				*target = true
			}
		}
		return nil
	}
}

// allow additional supported values for a "true" decision
// in additional to the usual ones provided by strconv.ParseBool
func asBool(key string, target *bool) cm.ParseFunc {
	return func(data map[string]string) error {
		raw, ok := data[key]
		if !ok {
			return nil
		}
		val, err := strconv.ParseBool(raw)
		if err == nil {
			*target = val
			return nil
		}
		return nil
	}
}

// asString passes the value at key through into the target, if it exists.
// TODO(mattmoor): This might be a nice variation on cm.AsString to upstream.
func asString(key string, target *string, values ...string) cm.ParseFunc {
	return func(data map[string]string) error {
		raw, ok := data[key]
		if !ok {
			return nil
		}
		if len(values) > 0 {
			vals := sets.New[string](values...)
			if !vals.Has(raw) {
				return fmt.Errorf("invalid value %q wanted one of %v", raw, sets.List[string](vals))
			}
		}
		*target = raw
		return nil
	}
}

// asStringSet parses the value at key as a sets.Set[string] (split by ',') into the target, if it exists.
func asStringSet(key string, target *sets.Set[string], allowed sets.Set[string]) cm.ParseFunc {
	return func(data map[string]string) error {
		if raw, ok := data[key]; ok {
			if raw == "" {
				*target = sets.New[string]("")
				return nil
			}
			splitted := strings.Split(raw, ",")
			if allowed.Len() > 0 {
				for i, v := range splitted {
					splitted[i] = strings.TrimSpace(v)
					if !allowed.Has(splitted[i]) {
						return fmt.Errorf("invalid value %q wanted one of %v", splitted[i], sets.List[string](allowed))
					}
				}
			}
			*target = sets.New[string](splitted...)
		}
		return nil
	}
}
