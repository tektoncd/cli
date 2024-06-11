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

	grafeasutil "github.com/grafeas/grafeas/go/utils/intoto"
	pb "github.com/grafeas/grafeas/proto/v1/grafeas_go_proto"
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/pkg/errors"
	"github.com/sigstore/cosign/v2/pkg/types"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/extract"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/status"
	"knative.dev/pkg/logging"
)

const (
	StorageBackendGrafeas     = "grafeas"
	projectPathFormat         = "projects/%s"
	notePathFormat            = "projects/%s/notes/%s"
	attestationNoteNameFormat = "%s-simplesigning"
	buildNoteNameFormat       = "%s-%s-intoto"
)

// Backend is a storage backend that stores signed payloads in the storage that
// is built on the top of grafeas i.e. container analysis.
type Backend struct {
	client pb.GrafeasClient
	cfg    config.Config
}

// NewStorageBackend returns a new Grafeas StorageBackend that stores signatures in a Grafeas server
func NewStorageBackend(ctx context.Context, cfg config.Config) (*Backend, error) {
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
		client: client,
		cfg:    cfg,
	}, nil
}

// StorePayload implements the storage.Backend interface.
func (b *Backend) StorePayload(ctx context.Context, obj objects.TektonObject, rawPayload []byte, signature string, opts config.StorageOpts) error {
	logger := logging.FromContext(ctx)
	// We only support simplesigning for OCI images, and in-toto for taskrun & pipelinerun.
	if _, ok := formats.IntotoAttestationSet[opts.PayloadFormat]; !ok && opts.PayloadFormat != formats.PayloadTypeSimpleSigning {
		return errors.New("Grafeas storage backend only supports simplesigning and intoto payload format.")
	}

	// Check if projectID is configured. If not, stop and return an error
	if b.cfg.Storage.Grafeas.ProjectID == "" {
		return errors.New("Project ID must be configured!")
	}

	// check if noteID is configured. If not, we give it a name as `tekton-<namespace>`
	if b.cfg.Storage.Grafeas.NoteID == "" {
		generatedNoteID := fmt.Sprintf("tekton-%s", obj.GetNamespace())
		b.cfg.Storage.Grafeas.NoteID = generatedNoteID
	}

	// step1: create note
	// If the note already exists, just move to the next step of creating occurrence.
	if _, err := b.createNote(ctx, obj, opts); err != nil && status.Code(err) != codes.AlreadyExists {
		return err
	}

	// step2: create occurrences
	occurrences, err := b.createOccurrence(ctx, obj, rawPayload, signature, opts)
	if err != nil {
		return err
	}

	occNames := []string{}
	for _, occ := range occurrences {
		occNames = append(occNames, occ.GetName())
	}

	if len(occNames) == 0 {
		logger.Infof("No occurrences created for payload of type %s for %s %s/%s", string(opts.PayloadFormat), obj.GetGVK(), obj.GetNamespace(), obj.GetName())
	} else {
		logger.Infof("Successfully created grafeas occurrences %v for %s %s/%s", occNames, obj.GetGVK(), obj.GetNamespace(), obj.GetName())
	}

	return nil
}

// Retrieve payloads from grafeas server and store it in a map
func (b *Backend) RetrievePayloads(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (map[string]string, error) {
	// initialize an empty map for result
	result := make(map[string]string)

	// get all occurrences created using this backend
	occurrences, err := b.getAllOccurrences(ctx, obj, opts)
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
func (b *Backend) RetrieveSignatures(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (map[string][]string, error) {
	// initialize an empty map for result
	result := make(map[string][]string)

	// get all occurrences created using this backend
	occurrences, err := b.getAllOccurrences(ctx, obj, opts)
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
func (b *Backend) createNote(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (*pb.Note, error) {
	notePrefix := b.cfg.Storage.Grafeas.NoteID

	// for oci image: AttestationNote
	if opts.PayloadFormat == formats.PayloadTypeSimpleSigning {
		return b.client.CreateNote(ctx,
			&pb.CreateNoteRequest{
				Parent: b.getProjectPath(),
				NoteId: fmt.Sprintf(attestationNoteNameFormat, b.cfg.Storage.Grafeas.NoteID),
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

	return b.createBuildNote(ctx, fmt.Sprintf(buildNoteNameFormat, notePrefix, obj.GetKindName()), obj)
}

func (b *Backend) createBuildNote(ctx context.Context, noteid string, obj objects.TektonObject) (*pb.Note, error) {
	return b.client.CreateNote(ctx,
		&pb.CreateNoteRequest{
			Parent: b.getProjectPath(),
			NoteId: noteid,
			Note: &pb.Note{
				ShortDescription: "Build Provenance Note for TaskRun",
				Type: &pb.Note_Build{
					Build: &pb.BuildNote{
						BuilderVersion: obj.GetGVK(),
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
// for a taskrun/pipelinerun object
//   - its intoto payload and signature will be stored in a BUILD occurrences for each image artifact generated from the taskrun/pipelinerun
//   - each BUILD occurrence will have the same data but differ in the ResourceUri field
//   - the identifier/ResourceUri is IMAGE_URL@IMAGE_DIGEST
func (b *Backend) createOccurrence(ctx context.Context, obj objects.TektonObject, payload []byte, signature string, opts config.StorageOpts) ([]*pb.Occurrence, error) {
	occs := []*pb.Occurrence{}

	// create Occurrence_Attestation for OCI
	if opts.PayloadFormat == formats.PayloadTypeSimpleSigning {
		occ, err := b.createAttestationOccurrence(ctx, payload, signature, opts.FullKey)
		if err != nil {
			return nil, err
		}
		occs = append(occs, occ)
		return occs, nil
	}

	// create Occurrence_Build for TaskRun
	allURIs := b.getAllArtifactURIs(ctx, opts.PayloadFormat, obj)
	for _, uri := range allURIs {
		occ, err := b.createBuildOccurrence(ctx, obj, payload, signature, uri)
		if err != nil {
			return nil, err
		}
		occs = append(occs, occ)
	}
	return occs, nil
}

func (b *Backend) getAllArtifactURIs(ctx context.Context, payloadFormat config.PayloadType, obj objects.TektonObject) []string {
	logger := logging.FromContext(ctx)
	payloader, err := formats.GetPayloader(payloadFormat, b.cfg)
	if err != nil {
		logger.Infof("couldn't get payloader for %v format, will use extract.RetrieveAllArtifactURIs method instead", payloadFormat)
		return extract.RetrieveAllArtifactURIs(ctx, obj, b.cfg.Artifacts.PipelineRuns.DeepInspectionEnabled)
	}

	if uris, err := payloader.RetrieveAllArtifactURIs(ctx, obj); err == nil {
		return uris
	}

	logger.Infof("couldn't get URIs from payloader %v, will use extract.RetrieveAllArtifactURIs method instead", payloadFormat)
	return extract.RetrieveAllArtifactURIs(ctx, obj, b.cfg.Artifacts.PipelineRuns.DeepInspectionEnabled)
}

func (b *Backend) createAttestationOccurrence(ctx context.Context, payload []byte, signature string, uri string) (*pb.Occurrence, error) {
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

func (b *Backend) createBuildOccurrence(ctx context.Context, obj objects.TektonObject, payload []byte, signature string, uri string) (*pb.Occurrence, error) {
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
				NoteName:    b.getBuildNotePath(obj),
				Details:     occurrenceDetails,
				Envelope:    envelope,
			},
		},
	)
}

func (b *Backend) getProjectPath() string {
	projectID := b.cfg.Storage.Grafeas.ProjectID
	return fmt.Sprintf(projectPathFormat, projectID)
}

func (b *Backend) getAttestationNotePath() string {
	projectID := b.cfg.Storage.Grafeas.ProjectID
	noteID := b.cfg.Storage.Grafeas.NoteID
	return fmt.Sprintf(notePathFormat, projectID, fmt.Sprintf(attestationNoteNameFormat, noteID))
}

func (b *Backend) getBuildNotePath(obj objects.TektonObject) string {
	projectID := b.cfg.Storage.Grafeas.ProjectID
	noteID := b.cfg.Storage.Grafeas.NoteID
	return fmt.Sprintf(notePathFormat, projectID, fmt.Sprintf(buildNoteNameFormat, noteID, obj.GetKindName()))
}

// getAllOccurrences retrieves back all occurrences created for a taskrun
func (b *Backend) getAllOccurrences(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) ([]*pb.Occurrence, error) {
	result := []*pb.Occurrence{}
	// step 1: get all resource URIs created under the taskrun
	uriFilters := b.getAllArtifactURIs(ctx, opts.PayloadFormat, obj)

	// step 2: find all build occurrences
	if _, ok := formats.IntotoAttestationSet[opts.PayloadFormat]; ok {
		occs, err := b.findOccurrencesForCriteria(ctx, b.getBuildNotePath(obj), uriFilters)
		if err != nil {
			return nil, err
		}
		result = append(result, occs...)
	}

	// step 3: find all attestation occurrences
	if opts.PayloadFormat == formats.PayloadTypeSimpleSigning {
		occs, err := b.findOccurrencesForCriteria(ctx, b.getAttestationNotePath(), uriFilters)
		if err != nil {
			return nil, err
		}
		result = append(result, occs...)
	}

	return result, nil
}

// findOccurrencesForCriteria lookups a project's occurrences by the resource uri
func (b *Backend) findOccurrencesForCriteria(ctx context.Context, noteName string, resourceURIs []string) ([]*pb.Occurrence, error) {

	var uriFilters []string
	for _, url := range resourceURIs {
		uriFilters = append(uriFilters, fmt.Sprintf("resourceUrl=%q", url))
	}

	occurences, err := b.client.ListNoteOccurrences(ctx,
		&pb.ListNoteOccurrencesRequest{
			Name:   noteName,
			Filter: strings.Join(uriFilters, " OR "),
		},
	)

	if err != nil {
		return nil, err
	}
	return occurences.GetOccurrences(), nil
}
