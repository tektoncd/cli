package bundle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/types"
	"github.com/docker/docker/pkg/homedir"
	"github.com/google/go-containerregistry/pkg/authn"
)

type podmanKeychain struct {
	mu sync.Mutex
}

var PodmanKeyChain authn.Keychain = &podmanKeychain{}

func (pk *podmanKeychain) Resolve(target authn.Resource) (authn.Authenticator, error) {
	pk.mu.Lock()
	defer pk.mu.Unlock()

	authFile, err := os.Open(getPathToPodmanAuth())
	// Return error only when the auth file is there but somehow unable to read.
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return authn.Anonymous, nil
		}
		return nil, err
	}
	defer authFile.Close()

	cf, err := config.LoadFromReader(authFile)
	if err != nil {
		return nil, err
	}

	key := target.RegistryStr()
	cfg, err := cf.GetAuthConfig(key)
	if err != nil {
		return nil, err
	}

	empty := types.AuthConfig{}
	if cfg == empty {
		return authn.Anonymous, nil
	}
	return authn.FromConfig(authn.AuthConfig{
		Username:      cfg.Username,
		Password:      cfg.Password,
		Auth:          cfg.Auth,
		IdentityToken: cfg.IdentityToken,
		RegistryToken: cfg.RegistryToken,
	}), nil
}

func getPathToPodmanAuth() string {
	var (
		defaultPerUIDPathFormat = filepath.FromSlash("/run/containers/%d/auth.json")
		xdgRuntimeDirPath       = filepath.FromSlash("containers/auth.json")
		nonLinuxAuthFilePath    = filepath.FromSlash(".config/containers/auth.json")
	)

	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return filepath.Join(homedir.Get(), nonLinuxAuthFilePath)
	}

	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir != "" {
		return filepath.Join(runtimeDir, xdgRuntimeDirPath)
	}
	return fmt.Sprintf(defaultPerUIDPathFormat, os.Getuid())
}
