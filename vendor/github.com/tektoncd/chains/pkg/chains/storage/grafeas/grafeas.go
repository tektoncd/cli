/*
Copyright 2022 The Tekton Authors
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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	grafeasutil "github.com/grafeas/grafeas/go/utils/intoto"
	pb "github.com/grafeas/grafeas/proto/v1/grafeas_go_proto"
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/pkg/errors"
	"github.com/sigstore/cosign/pkg/types"
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

const (
	StorageBackendGrafeas     = "grafeas"
	projectPath               = "projects/%s"
	notePath                  = "projects/%s/notes/%s"
	attestationNoteNameFormat = "%s-simplesigning"
	buildNoteNameFormat       = "%s-intoto"
)

// Backend is a storage backend that stores signed payloads in the storage that
// is built on the top of grafeas i.e. container analysis.
type Backend struct {
	logger *zap.SugaredLogger
	client pb.GrafeasClient
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
	client := pb.NewGrafeasClient(conn)

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
	if opts.PayloadFormat != formats.PayloadTypeInTotoIte6 && opts.PayloadFormat != formats.PayloadTypeSimpleSigning {
		return errors.New("Grafeas storage backend only supports simplesigning and intoto payload format.")
	}

	// Check if projectID is configured. If not, stop and return an error
	if b.cfg.Storage.Grafeas.ProjectID == "" {
		return errors.New("Project ID must be configured!")
	}

	// check if noteID is configured. If not, we give it a name as `tekton-<namespace>`
	if b.cfg.Storage.Grafeas.NoteID == "" {
		generatedNoteID := fmt.Sprintf("tekton-%s", tr.GetNamespace())
		b.cfg.Storage.Grafeas.NoteID = generatedNoteID
	}

	// step1: create note
	// If the note already exists, just move to the next step of creating occurrence.
	if _, err := b.createNote(ctx, tr, opts); err != nil && status.Code(err) != codes.AlreadyExists {
		return err
	}

	// step2: create occurrences
	occurrences, err := b.createOccurrence(ctx, tr, rawPayload, signature, opts)
	if err != nil {
		return err
	}

	names := []string{}
	for _, occ := range occurrences {
		names = append(names, occ.GetName())
	}

	b.logger.Infof("Successfully stored signature in the grafeas occurrences %s.", names)
	return nil
}

// Retrieve payloads from grafeas server and store it in a map
func (b *Backend) RetrievePayloads(ctx context.Context, tr *v1beta1.TaskRun, opts config.StorageOpts) (map[string]string, error) {
	// initialize an empty map for result
	result := make(map[string]string)

	// get all occurrences created using this backend
	occurrences, err := b.getAllOccurrences(ctx, tr, opts)
	if err != nil {
		return nil, err
	}

	for _, occ := range occurrences {
		// get payload identifier
		name := occ.GetResourceUri()

		// get "Payload" field from the occurrence
		payload := occ.GetEnvelope().GetPayload()

		result[name] = string(payload)
	}

	return result, nil
}

// Retrieve signatures from grafeas server and store it in a map
func (b *Backend) RetrieveSignatures(ctx context.Context, tr *v1beta1.TaskRun, opts config.StorageOpts) (map[string][]string, error) {
	// initialize an empty map for result
	result := make(map[string][]string)

	// get all occurrences created using this backend
	occurrences, err := b.getAllOccurrences(ctx, tr, opts)
	if err != nil {
		return nil, err
	}

	for _, occ := range occurrences {
		// get the Signature identifier
		name := occ.GetResourceUri()

		// get "Signatures" field from the occurrence
		signatures := occ.GetEnvelope().GetSignatures()
		// signature string
		sigStrings := []string{}
		for _, sig := range signatures {
			sigStrings = append(sigStrings, string(sig.GetSig()))
		}

		result[name] = sigStrings
	}

	return result, nil
}

func (b *Backend) Type() string {
	return StorageBackendGrafeas
}

// ----------------------------- Helper Functions ----------------------------
// createNote creates grafeas note that will be linked to grafeas occurrences
func (b *Backend) createNote(ctx context.Context, tr *v1beta1.TaskRun, opts config.StorageOpts) (*pb.Note, error) {
	noteID := b.cfg.Storage.Grafeas.NoteID

	// for oci image: AttestationNote
	if opts.PayloadFormat == formats.PayloadTypeSimpleSigning {
		return b.client.CreateNote(ctx,
			&pb.CreateNoteRequest{
				Parent: b.getProjectPath(),
				NoteId: fmt.Sprintf(attestationNoteNameFormat, noteID),
				Note: &pb.Note{
					ShortDescription: "OCI Artifact Attestation Note",
					Type: &pb.Note_Attestation{
						Attestation: &pb.AttestationNote{
							Hint: &pb.AttestationNote_Hint{
								HumanReadableName: b.cfg.Storage.Grafeas.NoteHint,
							},
						},
					},
				},
			},
		)
	}

	// for taskrun: BuildNote
	return b.client.CreateNote(ctx,
		&pb.CreateNoteRequest{
			Parent: b.getProjectPath(),
			NoteId: fmt.Sprintf(buildNoteNameFormat, noteID),
			Note: &pb.Note{
				ShortDescription: "Build Provenance Note",
				Type: &pb.Note_Build{
					Build: &pb.BuildNote{
						BuilderVersion: tr.GetGroupVersionKind().GroupVersion().String(),
					},
				},
			},
		},
	)
}

// createOccurrence creates grafeas occurrences in the grafeas server that stores the original payload and its signature
// for a single oci artifact
//   - its simplesigning payload and signature will be stored in ATTESTATION occurrence
//   - the identifier/ResourceUri is IMAGE_URL@IMAGE_DIGEST
//
// for a taskrun object
//   - its intoto payload and signature will be stored in a BUILD occurrences for each image artifact generated from the taskrun
//   - each BUILD occurrence will have the same data but differ in the ResourceUri field
//   - the identifier/ResourceUri is IMAGE_URL@IMAGE_DIGEST
func (b *Backend) createOccurrence(ctx context.Context, tr *v1beta1.TaskRun, payload []byte, signature string, opts config.StorageOpts) ([]*pb.Occurrence, error) {
	occs := []*pb.Occurrence{}

	// create Occurrence_Attestation for OCI
	if opts.PayloadFormat == formats.PayloadTypeSimpleSigning {
		uri := b.retrieveSingleOCIURI(tr, opts)
		occ, err := b.createAttestationOccurrence(ctx, tr, payload, signature, uri)
		if err != nil {
			return nil, err
		}
		occs = append(occs, occ)
		return occs, nil
	}

	// create Occurrence_Build for TaskRun
	allURIs := b.retrieveAllArtifactIdentifiers(tr)
	for _, uri := range allURIs {
		occ, err := b.createBuildOccurrence(ctx, tr, payload, signature, uri)
		if err != nil {
			return nil, err
		}
		occs = append(occs, occ)
	}
	return occs, nil
}

func (b *Backend) createAttestationOccurrence(ctx context.Context, tr *v1beta1.TaskRun, payload []byte, signature string, uri string) (*pb.Occurrence, error) {
	occurrenceDetails := &pb.Occurrence_Attestation{
		Attestation: &pb.AttestationOccurrence{
			SerializedPayload: payload,
			Signatures: []*pb.Signature{
				{
					Signature: []byte(signature),
					// TODO (#471): currently we only support storing KMS keyID, may add other keys' ids later i.e. k8s secret, fulcio
					PublicKeyId: b.cfg.Signers.KMS.KMSRef,
				},
			},
		},
	}
	envelope := &pb.Envelope{
		Payload:     payload,
		PayloadType: types.SimpleSigningMediaType,
		Signatures: []*pb.EnvelopeSignature{
			{
				Sig: []byte(signature),
				// TODO (#471): currently we only support storing KMS keyID, may add other keys' ids later i.e. k8s secret, fulcio
				Keyid: b.cfg.Signers.KMS.KMSRef,
			},
		},
	}

	return b.client.CreateOccurrence(ctx,
		&pb.CreateOccurrenceRequest{
			Parent: b.getProjectPath(),
			Occurrence: &pb.Occurrence{
				ResourceUri: uri,
				NoteName:    b.getAttestationNotePath(),
				Details:     occurrenceDetails,
				Envelope:    envelope,
			},
		},
	)
}

func (b *Backend) createBuildOccurrence(ctx context.Context, tr *v1beta1.TaskRun, payload []byte, signature string, uri string) (*pb.Occurrence, error) {
	in := intoto.ProvenanceStatement{}
	if err := json.Unmarshal(payload, &in); err != nil {
		return nil, err
	}

	pbf, err := grafeasutil.ToProto(&in)
	if err != nil {
		return nil, fmt.Errorf("Unable to convert to Grafeas proto: %w", err)
	}

	occurrenceDetails := &pb.Occurrence_Build{
		Build: &pb.BuildOccurrence{
			IntotoStatement: pbf,
		},
	}

	envelope := &pb.Envelope{
		Payload:     payload,
		PayloadType: types.IntotoPayloadType,
		Signatures: []*pb.EnvelopeSignature{
			{
				Sig:   []byte(signature),
				Keyid: b.cfg.Signers.KMS.KMSRef,
			},
		},
	}

	return b.client.CreateOccurrence(ctx,
		&pb.CreateOccurrenceRequest{
			Parent: b.getProjectPath(),
			Occurrence: &pb.Occurrence{
				ResourceUri: uri,
				NoteName:    b.getBuildNotePath(),
				Details:     occurrenceDetails,
				Envelope:    envelope,
			},
		},
	)
}

func (b *Backend) getProjectPath() string {
	projectID := b.cfg.Storage.Grafeas.ProjectID
	return fmt.Sprintf(projectPath, projectID)
}

func (b *Backend) getAttestationNotePath() string {
	projectID := b.cfg.Storage.Grafeas.ProjectID
	noteID := b.cfg.Storage.Grafeas.NoteID
	return fmt.Sprintf(notePath, projectID, fmt.Sprintf(attestationNoteNameFormat, noteID))
}

func (b *Backend) getBuildNotePath() string {
	projectID := b.cfg.Storage.Grafeas.ProjectID
	noteID := b.cfg.Storage.Grafeas.NoteID
	return fmt.Sprintf(notePath, projectID, fmt.Sprintf(buildNoteNameFormat, noteID))
}

// getAllOccurrences retrieves back all occurrences created for a taskrun
func (b *Backend) getAllOccurrences(ctx context.Context, tr *v1beta1.TaskRun, opts config.StorageOpts) ([]*pb.Occurrence, error) {
	// step 1: get all resource URIs created under the taskrun
	uriFilters := b.retrieveAllArtifactIdentifiers(tr)

	// step 2: find all occurrences by using ListOccurrences filters
	occs, err := b.findOccurrencesForCriteria(ctx, b.getProjectPath(), uriFilters)
	if err != nil {
		return nil, err
	}
	return occs, nil
}

// findOccurrencesForCriteria lookups a project's occurrences by the resource uri
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

// get resource uri for a single oci image in the format of `IMAGE_URL@IMAGE_DIGEST`
func (b *Backend) retrieveSingleOCIURI(tr *v1beta1.TaskRun, opts config.StorageOpts) string {
	imgs := b.retrieveAllArtifactIdentifiers(tr)
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

// retrieve the uri of all images generated from a specific taskrun in the format of `IMAGE_URL@IMAGE_DIGEST`
func (b *Backend) retrieveAllArtifactIdentifiers(tr *v1beta1.TaskRun) []string {
	result := []string{}
	// for image artifacts
	images := artifacts.ExtractOCIImagesFromResults(tr, b.logger)
	for _, image := range images {
		ref, ok := image.(name.Digest)
		if !ok {
			continue
		}
		result = append(result, ref.Name())
	}

	// for other signable artifacts
	artifacts := artifacts.ExtractSignableTargetFromResults(tr, b.logger)
	for _, a := range artifacts {
		result = append(result, a.FullRef())
	}
	return result
}
