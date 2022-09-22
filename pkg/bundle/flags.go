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
	var opts []remoteimg.Option

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

	// Use local keychain if no auth is provided. It's not allowed to use both.
	if len(opts) == 0 {
		keychains := authn.NewMultiKeychain(authn.DefaultKeychain, PodmanKeyChain)
		opts = []remoteimg.Option{remoteimg.WithAuthFromKeychain(keychains)}
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

// PullOptions configure how an image is cached once it is fetched from the remote.
type CacheOptions struct {
	cacheDir string
	noCache  bool
}

// AddCacheFlags will define a set of flags to control how Tekton Bundle caching is done.
func AddCacheFlags(flags *pflag.FlagSet, opts *CacheOptions) {
	flags.StringVar(&opts.cacheDir, "cache-dir", "~/.tekton/bundles", "A directory to cache Tekton bundles in.")
	flags.BoolVar(&opts.noCache, "no-cache", false, "If set to true, pulls a Tekton bundle from the remote even its exact digest is available in the cache.")
}
