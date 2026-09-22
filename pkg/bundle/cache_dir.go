package bundle

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func (o *CacheOptions) directory() (string, error) {
	if o.cacheDirSet || o.cacheDir != "" {
		if o.cacheDir == "" {
			return "", fmt.Errorf("--cache-dir cannot be empty")
		}
		dir, err := expandCacheDir(o.cacheDir)
		if err != nil {
			return "", fmt.Errorf("failed to expand cache directory %q: %w", o.cacheDir, err)
		}
		return dir, nil
	}

	return defaultCacheDir()
}

func expandCacheDir(path string) (string, error) {
	if len(path) == 0 || path[0] != '~' {
		return path, nil
	}
	if len(path) > 1 && path[1] != '/' && path[1] != '\\' {
		return "", errors.New("cannot expand user-specific home dir")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, path[1:]), nil
}

func defaultCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory for bundle cache: %w", err)
	}
	if !filepath.IsAbs(homeDir) {
		return "", fmt.Errorf("home directory %q is not an absolute path", homeDir)
	}

	tektonDir := filepath.Join(homeDir, ".tekton")
	info, err := os.Stat(tektonDir)
	if err == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("Tekton directory %q is not a directory", tektonDir)
		}
		return filepath.Join(tektonDir, "bundles"), nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("could not inspect Tekton directory %q: %w", tektonDir, err)
	}

	if xdgCacheHome := os.Getenv("XDG_CACHE_HOME"); xdgCacheHome != "" {
		if !filepath.IsAbs(xdgCacheHome) {
			return "", fmt.Errorf("XDG_CACHE_HOME %q is not an absolute path", xdgCacheHome)
		}
		return filepath.Join(xdgCacheHome, "tkn", "bundles"), nil
	}

	return filepath.Join(homeDir, ".cache", "tkn", "bundles"), nil
}
