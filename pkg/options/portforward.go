package options

type PortForwardOptions struct {
	Port    string
	PodName string

	StopChan  <-chan struct{}
	ReadyChan chan struct{}
}
