package plugins

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/env"
	"gotest.tools/v3/fs"
)

func TestFindPlugin(t *testing.T) {
	nd := fs.NewDir(t, "TestFindPlugins")
	defer nd.Remove()
	// nolint: gosec
	err := ioutil.WriteFile(nd.Join("tkn-test"), []byte("test"), 0o700)
	assert.NilError(t, err)
	defer env.Patch(t, pluginDirEnv, nd.Path())()
	path, err := FindPlugin("test")
	assert.NilError(t, err)
	assert.Equal(t, path, nd.Join("tkn-test"))
}

func TestFindPluginInPath(t *testing.T) {
	nd := fs.NewDir(t, "TestFindPluginsInPath")
	defer nd.Remove()
	// nolint: gosec
	err := ioutil.WriteFile(nd.Join("tkn-testp"), []byte("testp"), 0o700)
	assert.NilError(t, err)
	defer env.Patch(t, "PATH", nd.Path())()
	path, err := FindPlugin("testp")
	assert.NilError(t, err)
	assert.Equal(t, path, nd.Join("tkn-testp"))
}

func TestGetAllTknPluginFromPathPlugindir(t *testing.T) {
	nd := fs.NewDir(t, "TestGetAllTknPluginFromPluginPath")
	defer nd.Remove()
	// nolint: gosec
	err := ioutil.WriteFile(nd.Join("tkn-fromplugindir"), []byte("test"), 0o700)
	assert.NilError(t, err)

	defer env.Patch(t, "PATH", "")()
	defer env.Patch(t, pluginDirEnv, nd.Path())()

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
	err := ioutil.WriteFile(nd.Join("tkn-test"), []byte("testp"), 0o700)
	assert.NilError(t, err)

	nd2 := fs.NewDir(t, "TestGetAllTknPluginFromPaths2")
	defer nd2.Remove()
	// nolint: gosec
	err = ioutil.WriteFile(nd.Join("tkn-test"), []byte("testp"), 0o700)
	assert.NilError(t, err)

	defer env.Patch(t, "PATH", fmt.Sprintf("%s:%s", nd.Path(), nd2.Path()))()

	// Reset the TKN_PLUGINS_DIR so that during local test
	// existing plugins are not considered using tests
	pluginHome := os.Getenv("TKN_PLUGINS_DIR")
	os.Setenv("TKN_PLUGINS_DIR", "/non/existing/path")

	plugins := GetAllTknPluginFromPaths()
	assert.NilError(t, err)
	assert.Equal(t, len(plugins), 1)

	os.Setenv("TKN_PLUGINS_DIR", pluginHome)
}
