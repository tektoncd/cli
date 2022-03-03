package bundle

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"gotest.tools/assert"
)

var testRegistry, _ = name.NewRegistry("test.io", name.WeakValidation)

func setupConfigDir(t *testing.T) string {
	tmpdir, err := ioutil.TempDir("", "keychain")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		os.Setenv("HOME", tmpdir)
	} else {
		os.Setenv("XDG_RUNTIME_DIR", tmpdir)
	}
	return tmpdir
}

func setUpConfigFile(t *testing.T, content string) string {
	tmpdir := setupConfigDir(t)

	authFile := getPathToPodmanAuth()
	authFileDir := filepath.Dir(authFile)

	if err := os.MkdirAll(authFileDir, 0700); err != nil {
		t.Fatalf("unable to mkdir %q: %v", authFileDir, err)
	}

	if err := ioutil.WriteFile(authFile, []byte(content), 0600); err != nil {
		t.Fatalf("write %q: %v", authFile, err)
	}
	return tmpdir
}

func encode(user, pass string) string {
	delimited := fmt.Sprintf("%s:%s", user, pass)
	return base64.StdEncoding.EncodeToString([]byte(delimited))
}

// If auth.json doesn't exist, return Anonymous
func TestNoConfig(t *testing.T) {
	cd := setupConfigDir(t)
	defer os.RemoveAll(cd)

	auth, err := PodmanKeyChain.Resolve(testRegistry)
	if err != nil {
		t.Fatalf("Resolve() = %v", err)
	}

	if auth != authn.Anonymous {
		t.Errorf("expected Anonymous, got %v", auth)
	}
}

func TestNoAuthContent(t *testing.T) {
	cd := setUpConfigFile(t, `{"auths": {}}`)
	defer os.RemoveAll(cd)
	auth, err := PodmanKeyChain.Resolve(testRegistry)

	if err != nil {
		t.Fatalf("Resolve() = %v", err)
	}

	if auth != authn.Anonymous {
		t.Errorf("expected Anonymous, got %v", auth)
	}

}

func TestGetPathToPodmanAuth(t *testing.T) {
	switch runtime.GOOS {
	case "windows", "darwin":
		authPath := filepath.Join(os.Getenv("HOME"), ".config", "containers", "auth.json")
		assert.Equal(t, getPathToPodmanAuth(), authPath)
	default:
		os.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
		assert.Equal(t, getPathToPodmanAuth(), "/run/user/1000/containers/auth.json")
		os.Unsetenv("XDG_RUNTIME_DIR")
		assert.Equal(t, getPathToPodmanAuth(), fmt.Sprintf("/run/containers/%d/auth.json", os.Getuid()))
	}
}

func TestPodmanKeychain(t *testing.T) {
	tests := []struct {
		content string
		wantErr bool
		target  name.Registry
		cfg     *authn.AuthConfig
	}{
		{
			target:  testRegistry,
			content: `}{`,
			wantErr: true,
		},
		{
			target:  testRegistry,
			content: `{"credsStore":"#definitely-does-not-exist"}`,
			wantErr: true,
		},
		{
			target:  testRegistry,
			content: fmt.Sprintf(`{"auths": {"test.io": {"auth": %q}}}`, encode("foo", "bar")),
			cfg: &authn.AuthConfig{
				Username: "foo",
				Password: "bar",
			},
		},
	}

	for _, test := range tests {
		cd := setUpConfigFile(t, test.content)
		defer os.RemoveAll(cd)

		auth, err := PodmanKeyChain.Resolve(test.target)
		if test.wantErr {
			if err == nil {
				t.Fatal("wanted err, got nil")
			} else if err != nil {
				continue
			}
		}
		if err != nil {
			t.Fatalf("wanted nil, got err: %v", err)
		}
		cfg, err := auth.Authorization()
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(cfg, test.cfg) {
			t.Errorf("got %+v, want %+v", cfg, test.cfg)
		}
	}

}
