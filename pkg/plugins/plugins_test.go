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
