package eventlistener

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/eventlistener"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/triggers/pkg/reconciler/eventlistener/resources"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

var defaultPort = strconv.Itoa(resources.DefaultPort)

func portforwardCommand(p cli.Params) *cobra.Command {
	opts := &options.PortForwardOptions{}

	eg := `Forward port '9001' from EventListener 'foo' in namespace 'bar' to localhost port '8001':

	tkn eventlistener port-forward foo -n bar -p 8001:9001

or

	tkn el pf foo -n bar -p 8001:9001
`

	c := &cobra.Command{
		Use:          "port-forward",
		Aliases:      []string{"pf"},
		Short:        "Port forwards an EventListener in a namespace",
		Example:      eg,
		SilenceUsage: true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		Args:              cobra.ExactValidArgs(1),
		ValidArgsFunction: formatted.ParentCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			cs, err := p.Clients()
			if err != nil {
				return fmt.Errorf("failed to create tekton client")
			}

			pods, err := eventlistener.Pods(cs, args[0], metav1.GetOptions{}, p.Namespace())
			if err != nil {
				return err
			}

			if len(pods) == 0 {
				return fmt.Errorf("no pods available for EventListener %s", args[0])
			}

			// Do what kubectl does and use the first pod if there's multiple
			opts.PodName = pods[0].Name

			done := make(chan struct{}, 1)
			opts.ReadyChan = make(chan struct{})
			opts.StopChan = done

			signals := make(chan os.Signal, 1)
			signal.Notify(signals, os.Interrupt)
			defer signal.Stop(signals)

			go func() {
				<-signals
				close(done)
			}()

			s := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			return forwardPorts(p, s, opts)
		},
	}

	c.Flags().StringVarP(&opts.Port, "port", "p", defaultPort, "port to forward, format is [LOCAL_PORT:]REMOTE_PORT")
	return c
}

func forwardPorts(p cli.Params, s *cli.Stream, opts *options.PortForwardOptions) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	req := cs.Kube.CoreV1().RESTClient().Post().
		Namespace(p.Namespace()).
		Resource("pods").
		Name(opts.PodName).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(cs.Config)
	if err != nil {
		return err
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())
	fw, err := portforward.New(dialer, []string{opts.Port}, opts.StopChan, opts.ReadyChan, s.Out, s.Err)
	if err != nil {
		return err
	}
	return fw.ForwardPorts()
}
