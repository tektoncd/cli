package chains

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/storage"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/flags"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Namespace   string = "tekton-chains"
	ConfigMap   string = "chains-config"
	x509Keypair string = "k8s://tekton-chains/signing-secrets"
)

func Command(p cli.Params) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "chains",
		Aliases: []string{},
		Short:   "Manage Tekton Chains",
		Annotations: map[string]string{
			"commandType": "main",
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return flags.InitParams(p, cmd)
		},
	}

	flags.AddTektonOptions(cmd)
	_ = cmd.PersistentFlags().MarkHidden("context")
	_ = cmd.PersistentFlags().MarkHidden("kubeconfig")
	_ = cmd.PersistentFlags().MarkHidden("namespace")
	cmd.AddCommand(
		payloadCommand(p),
		signatureCommand(p),
	)
	return cmd
}

func InitializeBackends(cs *cli.Clients, namespace string, tr *v1beta1.TaskRun, sugaredLogger *zap.SugaredLogger) (map[string]storage.Backend, error) {
	// Prepare the Kubernetes client.
	kc := cs.Kube

	// Prepare the Pipelineset client.
	ps := cs.Tekton

	// Retrieve the Chains configuration.
	ctx := context.Background()
	chainsConfig, err := cs.Kube.CoreV1().ConfigMaps(Namespace).Get(ctx, ConfigMap, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error retrieving tekton chains configmap: %s", err)
	}
	cfg, err := config.NewConfigFromConfigMap(chainsConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating tekton chains configuration: %s", err)
	}

	// Initialize the backend.
	backends, err := storage.InitializeBackends(ps, kc, sugaredLogger, tr, *cfg)
	if err != nil {
		return nil, fmt.Errorf("error initializing backends: %s", err)
	}

	// Return the configured backend.
	return backends, nil
}

func GetTaskRunBackends(p cli.Params, tr *v1beta1.TaskRun) (map[string]storage.Backend, config.StorageOpts, error) {
	// Create the Tekton clients.
	cs, err := p.Clients()
	if err != nil {
		return nil, config.StorageOpts{}, fmt.Errorf("failed to retrieve clients")
	}

	// Prepare the logger.
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, config.StorageOpts{}, err
	}
	// flushes buffer, if any
	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Println(err)
		}
	}()
	sugaredLogger := logger.Sugar()

	// Get the storage backend.
	backends, err := InitializeBackends(cs, p.Namespace(), tr, sugaredLogger)
	if err != nil {
		return nil, config.StorageOpts{}, fmt.Errorf("failed to retrieve the backend storage: %v", err)
	}

	// Initialize the storage options.
	opts := config.StorageOpts{
		Key: fmt.Sprintf("taskrun-%s", tr.UID),
	}

	return backends, opts, nil
}

// ConfigMapToContext returns a context initialized with the Chains ConfigMap.
func ConfigMapToContext(cs *cli.Clients) (context.Context, error) {
	ctx := context.Background()
	chainsConfig, err := cs.Kube.CoreV1().ConfigMaps(Namespace).Get(ctx, ConfigMap, metav1.GetOptions{})
	if err != nil {
		return context.TODO(), fmt.Errorf("error retrieving tekton chains configmap: %s", err)
	}
	cfg, err := config.NewConfigFromConfigMap(chainsConfig)
	if err != nil {
		return context.TODO(), fmt.Errorf("error creating tekton chains configuration: %s", err)
	}
	return config.ToContext(ctx, cfg), nil
}

// AllFormattersString returns an array populated with the string representations
// of each supported format.
func AllFormattersString() []string {
	allFormattersStrings := []string{}
	for _, format := range formats.AllFormatters {
		allFormattersStrings = append(allFormattersStrings, string(format))
	}
	return allFormattersStrings
}
