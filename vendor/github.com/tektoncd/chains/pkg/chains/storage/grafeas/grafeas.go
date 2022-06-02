/*
Copyright 2020 The Tekton Authors
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

package grafeas

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	attestationpb "github.com/grafeas/grafeas/proto/v1beta1/attestation_go_proto"
	commonpb "github.com/grafeas/grafeas/proto/v1beta1/common_go_proto"
	pb "github.com/grafeas/grafeas/proto/v1beta1/grafeas_go_proto"
	"github.com/pkg/errors"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/status"
)

// TODOs
// - switch to grafeas v1 api once it gets updated to support slsa 0.2 predicate
// - create one BUILD occurrence per image built with same content but differing in URI
// - create one BUILD note per taskrun
// - change unit tests accordingly
// - make HumanReadableName in the attestation note configurable

const (
	StorageBackendGrafeas = "grafeas"
	projectNameFormat     = "projects/%s"
	noteNameFormat        = "projects/%s/notes/%s"
)

// Backend is a storage backend that stores signed payloads in the storage that
// is built on the top of grafeas i.e. container analysis.
type Backend struct {
	logger *zap.SugaredLogger
	client pb.GrafeasV1Beta1Client
	cfg    config.Config
}

// NewStorageBackend returns a new Grafeas StorageBackend that stores signatures in a Grafeas server
func NewStorageBackend(ctx context.Context, logger *zap.SugaredLogger, cfg config.Config) (*Backend, error) {
	// build connection through grpc
	// implicit uses Application Default Credentials to authenticate.
	// Requires `gcloud auth application-default login` to work locally
	creds, err := oauth.NewApplicationDefault(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, err
	}

	// TODO: make grafeas server configurable including checking if hostname is trusted
	server := "dns:///containeranalysis.googleapis.com"

	conn, err := grpc.Dial(server,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
		grpc.WithDefaultCallOptions(grpc.PerRPCCredentials(creds)),
	)
	if err != nil {
		return nil, err
	}

	// connection client
	client := pb.NewGrafeasV1Beta1Client(conn)

	// create backend instance
	return &Backend{
		logger: logger,
		client: client,
		cfg:    cfg,
	}, nil
}

// StorePayload implements the storage.Backend interface.
func (b *Backend) StorePayload(ctx context.Context, tr *v1beta1.TaskRun, rawPayload []byte, signature string, opts config.StorageOpts) error {
	// We only support simplesigning for OCI images, and in-toto for taskrun.
	if opts.PayloadFormat == formats.PayloadTypeTekton {
		return errors.New("Grafeas storage backend only supports OCI images and in-toto attestations")
	}

	// Check if projectID is configured. If not, stop and return an error
	if b.cfg.Storage.Grafeas.ProjectID == "" {
		return errors.New("Project ID needs to be configured!")
	}

	// check if noteID is configured. If not, we give it a name as `tekton-<namespace>`
	if b.cfg.Storage.Grafeas.NoteID == "" {
		generatedNoteID := fmt.Sprintf("tekton-%s", tr.GetNamespace())
		b.cfg.Storage.Grafeas.NoteID = generatedNoteID
	}

	b.logger.Infof("Trying to store payload on TaskRun %s/%s", tr.Namespace, tr.Name)

	// step1: create note
	if err := b.createNote(ctx); err != nil && status.Code(err) != codes.AlreadyExists {
		return err
	}

	// step2: create occurrence request
	occurrenceReq, err := b.createOccurrenceRequest(tr, rawPayload, signature, opts)
	if err != nil {
		return err
	}

	// step3: create/store occurrence
	occurrence, err := b.client.CreateOccurrence(ctx, occurrenceReq)
	if err != nil {
		return err
	}

	b.logger.Infof("Successfully created an occurrence %s (Occurrence_ID is automatically generated) for Taskrun %s/%s", occurrence.GetName(), tr.Namespace, tr.Name)
	return nil
}

// Retrieve payloads from grafeas server and store it in a map
func (b *Backend) RetrievePayloads(ctx context.Context, tr *v1beta1.TaskRun, opts config.StorageOpts) (map[string]string, error) {
	// initialize an empty map for result
	result := make(map[string]string)

	// get all occurrences created using this backend
	occurrences, err := b.getOccurrences(tr, ctx)
	if err != nil {
		return nil, err
	}

	for _, occ := range occurrences {
		// get payload identifier
		name := occ.GetResource().GetUri()
		// get "Payload" field from the occurrence
		payload := occ.GetAttestation().GetAttestation().GetGenericSignedAttestation().GetSerializedPayload()

		result[name] = string(payload)
	}

	return result, nil
}

// Retrieve signatures from grafeas server and store it in a map
func (b *Backend) RetrieveSignatures(ctx context.Context, tr *v1beta1.TaskRun, opts config.StorageOpts) (map[string][]string, error) {
	// initialize an empty map for result
	result := make(map[string][]string)

	// get all occurrences created using this backend
	occurrences, err := b.getOccurrences(tr, ctx)
	if err != nil {
		return nil, err
	}

	for _, occ := range occurrences {
		// get the Signature identifier
		name := occ.GetResource().GetUri()
		// get "Signatures" field from the occurrence
		signatures := occ.GetAttestation().GetAttestation().GetGenericSignedAttestation().GetSignatures()
		// unmarshal signatures
		unmarshalSigs := []string{}
		for _, sig := range signatures {
			unmarshalSigs = append(unmarshalSigs, string(sig.GetSignature()))
		}

		result[name] = unmarshalSigs
	}

	return result, nil
}

func (b *Backend) Type() string {
	return StorageBackendGrafeas
}

// ----------------------------- Helper Functions ----------------------------
func (b *Backend) createNote(ctx context.Context) error {
	noteID := b.cfg.Storage.Grafeas.NoteID

	b.logger.Infof("Creating a note with note name %s", noteID)

	// create note request
	noteReq := &pb.CreateNoteRequest{
		Parent: b.getProjectPath(),
		NoteId: noteID,
		Note: &pb.Note{
			ShortDescription: "An attestation note",
			Kind:             commonpb.NoteKind_ATTESTATION,
			Type: &pb.Note_AttestationAuthority{
				AttestationAuthority: &attestationpb.Authority{
					Hint: &attestationpb.Authority_Hint{
						HumanReadableName: "This note was auto-generated by Tekton Chains",
					},
				},
			},
		},
	}

	// store note
	if _, err := b.client.CreateNote(ctx, noteReq); err != nil {
		return err
	}

	return nil
}

func (b *Backend) createOccurrenceRequest(tr *v1beta1.TaskRun, payload []byte, signature string, opts config.StorageOpts) (*pb.CreateOccurrenceRequest, error) {
	occurrenceDetails := &pb.Occurrence_Attestation{
		Attestation: &attestationpb.Details{
			Attestation: &attestationpb.Attestation{
				Signature: &attestationpb.Attestation_GenericSignedAttestation{
					GenericSignedAttestation: &attestationpb.GenericSignedAttestation{
						ContentType:       b.getContentType(opts),
						SerializedPayload: payload,
						Signatures: []*commonpb.Signature{
							{
								Signature:   []byte(signature),
								PublicKeyId: b.cfg.Signers.KMS.KMSRef, // TODO: currently we only support storing kms keyID, will add other keys' ids later i.e. k8s secret, fulcio
							},
						},
					},
				},
			},
		},
	}

	envelope := &commonpb.Envelope{
		Payload:     payload,
		PayloadType: "in-toto attestations containing a slsa.dev/provenance predicate",
		Signatures: []*commonpb.EnvelopeSignature{
			{
				Sig:   []byte(signature),
				Keyid: b.cfg.Signers.KMS.KMSRef, // TODO: currently we only support storing kms keyID, will add other keys' ids later i.e. k8s secret, fulcio
			},
		},
	}

	uri, err := b.getResourceURI(tr, opts)
	if err != nil {
		return nil, err
	}

	occurrence := &pb.Occurrence{
		Resource: &pb.Resource{
			Uri: uri,
		},
		NoteName: b.getNotePath(),
		Details:  occurrenceDetails,
		Envelope: envelope,
	}

	occurrenceRequest := &pb.CreateOccurrenceRequest{
		Parent:     b.getProjectPath(),
		Occurrence: occurrence,
	}

	return occurrenceRequest, nil
}

func (b *Backend) getProjectPath() string {
	projectID := b.cfg.Storage.Grafeas.ProjectID
	return fmt.Sprintf(projectNameFormat, projectID)
}

func (b *Backend) getNotePath() string {
	projectID := b.cfg.Storage.Grafeas.ProjectID
	noteID := b.cfg.Storage.Grafeas.NoteID
	return fmt.Sprintf(noteNameFormat, projectID, noteID)
}

// decide the attestation content type based on its format (simplesigning or in-toto)
func (b *Backend) getContentType(opts config.StorageOpts) attestationpb.GenericSignedAttestation_ContentType {
	// for simplesigning
	if opts.PayloadFormat == formats.PayloadTypeSimpleSigning {
		return attestationpb.GenericSignedAttestation_SIMPLE_SIGNING_JSON
	}

	// for in-toto
	return attestationpb.GenericSignedAttestation_CONTENT_TYPE_UNSPECIFIED
}

// retrieve all occurrences created under a taskrun by filtering resource URI
func (b *Backend) getOccurrences(tr *v1beta1.TaskRun, ctx context.Context) ([]*pb.Occurrence, error) {
	// step 1: get all resource URIs created under the taskrun
	uriFilters := []string{}
	uriFilters = append(uriFilters, b.retrieveAllOCIURIs(tr)...)
	uriFilters = append(uriFilters, b.getTaskRunURI(tr))

	// step 2: find all occurrences by using ListOccurrences filters
	occs, err := b.findOccurrencesForCriteria(ctx, b.getProjectPath(), uriFilters)
	if err != nil {
		return nil, err
	}
	return occs, nil
}

// find all occurrences based on a number of criteria
// - current criteria we use are just project name and resource uri
// - we can add more criteria later if we want i.e. occurrence Kind, severity and PageSize etc.
func (b *Backend) findOccurrencesForCriteria(ctx context.Context, projectPath string, resourceURIs []string) ([]*pb.Occurrence, error) {
	var uriFilters []string
	for _, url := range resourceURIs {
		uriFilters = append(uriFilters, fmt.Sprintf("resourceUrl=%q", url))
	}

	occurences, err := b.client.ListOccurrences(ctx,
		&pb.ListOccurrencesRequest{
			Parent: projectPath,
			Filter: strings.Join(uriFilters, " OR "),
		},
	)

	if err != nil {
		return nil, err
	}
	return occurences.GetOccurrences(), nil
}

// get resource uri based on the configured payload format that helps differentiate artifact type as well.
func (b *Backend) getResourceURI(tr *v1beta1.TaskRun, opts config.StorageOpts) (string, error) {
	switch opts.PayloadFormat {
	case formats.PayloadTypeSimpleSigning:
		return b.getOCIURI(tr, opts), nil
	case formats.PayloadTypeInTotoIte6:
		return b.getTaskRunURI(tr), nil
	default:
		return "", errors.New("Invalid payload format. Only in-toto and simplesigning are supported.")
	}
}

// get resource uri for a taskrun in the format of namespace-scoped resource uri
// `/apis/GROUP/VERSION/namespaces/NAMESPACE/RESOURCETYPE/NAME``
// see more details here https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-uris
func (b *Backend) getTaskRunURI(tr *v1beta1.TaskRun) string {
	return fmt.Sprintf("/apis/%s/namespaces/%s/%s/%s@%s",
		tr.GetGroupVersionKind().GroupVersion().String(),
		tr.GetNamespace(),
		tr.GetGroupVersionKind().Kind,
		tr.GetName(),
		string(tr.UID),
	)
}

// get resource uri for an oci image in the format of `IMAGE_URL@IMAGE_DIGEST`
func (b *Backend) getOCIURI(tr *v1beta1.TaskRun, opts config.StorageOpts) string {
	imgs := b.retrieveAllOCIURIs(tr)
	for _, img := range imgs {
		// get digest part of the image representation
		digest := strings.Split(img, "sha256:")[1]

		// for oci image, the key in StorageOpts will be the first 12 chars of digest
		// so we want to compare
		digestKey := digest[:12]
		if digestKey == opts.Key {
			return img
		}
	}
	return ""
}

// get the uri of all images for a specific taskrun in the format of `IMAGE_URL@IMAGE_DIGEST`
func (b *Backend) retrieveAllOCIURIs(tr *v1beta1.TaskRun) []string {
	result := []string{}
	images := artifacts.ExtractOCIImagesFromResults(tr, b.logger)

	for _, image := range images {
		ref := image.(name.Digest)
		result = append(result, ref.Name())
	}

	return result
}
