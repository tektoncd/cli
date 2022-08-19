package plugins

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
)

const (
	pluginDirEnv = "TKN_PLUGINS_DIR"
	pluginDir    = "~/.config/tkn/plugins"
	tknPrefix    = "tkn-"
)

func getPluginDir() (string, error) {
	dir := os.Getenv(pluginDirEnv)
	// if TKN_PLUGINS_DIR is set, follow it
	if dir != "" {
		return dir, nil
	}
	// Respect XDG_CONFIG_HOME if set
	if xdgHome := os.Getenv("XDG_CONFIG_HOME"); xdgHome != "" {
		return filepath.Join(xdgHome, "tkn", "plugins"), nil
	}
	// Fallback to default pluginDir (~/.config/tkn/plugins)
	return homedir.Expand(pluginDir)
}

// Find a binary in plugin homedir directory or user paths.
func FindPlugin(pluginame string) (string, error) {
	cmd := tknPrefix + pluginame
	dir, _ := getPluginDir()
	path := filepath.Join(dir, cmd)
	_, err := os.Stat(path)
	if err == nil {
		// Found in dir
		return path, nil
	}

	path, err = exec.LookPath(cmd)
	if err == nil {
		return path, nil
	}

	return "", fmt.Errorf("cannot find plugin in path or %s: %s", pluginDir, cmd)
}

func GetAllTknPluginFromPaths() []string {
	pluginlist := []string{}
	paths := filepath.SplitList(os.Getenv("PATH"))
	if dir, err := getPluginDir(); err == nil {
		paths = append(paths, dir)
	}
	// go over all paths in the PATH environment
	// and add them to the completion command
	for _, path := range paths {
		// list all files in path
		files, err := os.ReadDir(path)
		if err != nil {
			continue
		}
		// add all files that start with tkn-
		for _, file := range files {
			if strings.HasPrefix(file.Name(), tknPrefix) {
				basep := strings.TrimLeft(file.Name(), tknPrefix)
				if contains(pluginlist, basep) {
					continue
				}
				fpath := filepath.Join(path, file.Name())
				info, err := os.Stat(fpath)
				if err != nil {
					continue
				}
				if info.Mode()&0o111 != 0 {
					pluginlist = append(pluginlist, basep)
				}
			}
		}
	}
	return pluginlist
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
