package bundle

import (
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	remoteimg "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/spf13/pflag"
)

// RemoteOptions is a set of flags that are used configure the connection options to a registry.
type RemoteOptions struct {
	bearerToken   string
	basicUsername string
	basicPassword string

	skipTLS bool
}

// ToOptions outputs a list of `remoteimg.Option`s that can be passed into various fetch/write calls to a remote
// registry.
func (r *RemoteOptions) ToOptions() []remoteimg.Option {
	opts := []remoteimg.Option{remoteimg.WithAuthFromKeychain(authn.DefaultKeychain)}

	// Set the auth chain based on the flags.
	if r.bearerToken != "" {
		opts = append(opts, remoteimg.WithAuth(&authn.Bearer{Token: r.bearerToken}))
	}
	if r.basicUsername != "" && r.basicPassword != "" {
		opts = append(opts, remoteimg.WithAuth(&authn.Basic{
			Username: r.basicUsername,
			Password: r.basicPassword,
		}))
	}

	transport := http.DefaultTransport.(*http.Transport)
	if r.skipTLS {
		transport.TLSClientConfig.InsecureSkipVerify = r.skipTLS
	}
	// TODO: consider adding CA overrides for self-signed or private registries.
	opts = append(opts, remoteimg.WithTransport(transport))
	return opts
}

// AddRemoteFlags will define a common set of flags that can be used to change how images are pushed/fetched from remote
// image repositories.
func AddRemoteFlags(flags *pflag.FlagSet, opts *RemoteOptions) {
	flags.StringVar(&opts.bearerToken, "remote-bearer", "", "A Bearer token to authenticate against the repository")
	flags.StringVar(&opts.basicUsername, "remote-username", "", "A username to pass to the registry for basic auth. Must be used with --remote-password")
	flags.StringVar(&opts.basicPassword, "remote-password", "", "A password to pass to the registry for basic auth. Must be used with --remote-username")

	// TLS related flags.
	flags.BoolVar(&opts.skipTLS, "remote-skip-tls", false, "If set to true, skips TLS check when connecting to the registry")
}
