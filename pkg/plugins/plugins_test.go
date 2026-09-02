package plugins

import (
	"fmt"
	"os"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

func TestFindPlugin(t *testing.T) {
	nd := fs.NewDir(t, "TestFindPlugins")
	defer nd.Remove()
	// nolint: gosec
	err := os.WriteFile(nd.Join("tkn-test"), []byte("test"), 0o700)
	assert.NilError(t, err)
	t.Setenv(pluginDirEnv, nd.Path())
	path, err := FindPlugin("test")
	assert.NilError(t, err)
	assert.Equal(t, path, nd.Join("tkn-test"))
}

func TestFindPluginInPath(t *testing.T) {
	nd := fs.NewDir(t, "TestFindPluginsInPath")
	defer nd.Remove()
	// nolint: gosec
	err := os.WriteFile(nd.Join("tkn-testp"), []byte("testp"), 0o700)
	assert.NilError(t, err)
	t.Setenv("PATH", nd.Path())
	path, err := FindPlugin("testp")
	assert.NilError(t, err)
	assert.Equal(t, path, nd.Join("tkn-testp"))
}

func TestGetAllTknPluginFromPathPlugindir(t *testing.T) {
	nd := fs.NewDir(t, "TestGetAllTknPluginFromPluginPath")
	defer nd.Remove()
	// nolint: gosec
	err := os.WriteFile(nd.Join("tkn-fromplugindir"), []byte("test"), 0o700)
	assert.NilError(t, err)

	t.Setenv("PATH", "")
	t.Setenv(pluginDirEnv, nd.Path())

	paths := GetAllTknPluginFromPaths()
	assert.NilError(t, err)
	assert.Equal(t, len(paths), 1)
	assert.Equal(t, paths[0], "fromplugindir")
}

func TestGetPluginDirRelativeTKNPluginsDir(t *testing.T) {
	t.Setenv(pluginDirEnv, "relative/path")
	_, err := getPluginDir()
	assert.ErrorContains(t, err, "not an absolute path")
}

func TestGetPluginDirRelativeXDGConfigHome(t *testing.T) {
	t.Setenv(pluginDirEnv, "")
	t.Setenv("XDG_CONFIG_HOME", "relative/xdg")
	_, err := getPluginDir()
	assert.ErrorContains(t, err, "not an absolute path")
}

func TestFindPluginDoesNotFallBackToCwd(t *testing.T) {
	nd := fs.NewDir(t, "TestFindPluginCwd")
	defer nd.Remove()
	err := os.WriteFile(nd.Join("tkn-evil"), []byte("evil"), 0o700)
	assert.NilError(t, err)

	// Change into the directory that contains the malicious binary.
	orig, err := os.Getwd()
	assert.NilError(t, err)
	defer os.Chdir(orig) //nolint:errcheck
	assert.NilError(t, os.Chdir(nd.Path()))

	// Force getPluginDir() to fail via a relative TKN_PLUGINS_DIR.
	// Keep nd off PATH so LookPath cannot find the binary either.
	t.Setenv(pluginDirEnv, "relative/bad/path")
	t.Setenv("PATH", "")

	// The binary is only reachable via cwd — FindPlugin must not find it.
	_, err = FindPlugin("evil")
	assert.ErrorContains(t, err, "cannot find plugin")
}

// as well tested differently in root_test.go
func TestGetAllTknPluginFromPaths(t *testing.T) {
	nd := fs.NewDir(t, "TestGetAllTknPluginFromPaths1")
	defer nd.Remove()
	// nolint: gosec
	err := os.WriteFile(nd.Join("tkn-test"), []byte("testp"), 0o700)
	assert.NilError(t, err)

	nd2 := fs.NewDir(t, "TestGetAllTknPluginFromPaths2")
	defer nd2.Remove()
	// nolint: gosec
	err = os.WriteFile(nd.Join("tkn-test"), []byte("testp"), 0o700)
	assert.NilError(t, err)

	t.Setenv("PATH", fmt.Sprintf("%s:%s", nd.Path(), nd2.Path()))
	t.Setenv("TKN_PLUGINS_DIR", "/non/existing/path")
	plugins := GetAllTknPluginFromPaths()
	assert.NilError(t, err)
	assert.Equal(t, len(plugins), 1)
}
