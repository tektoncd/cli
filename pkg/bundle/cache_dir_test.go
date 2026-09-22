package bundle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func TestAddCacheFlagsDoesNotDeclareStaticDefault(t *testing.T) {
	var options CacheOptions
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	AddCacheFlags(flags, &options)

	flag := flags.Lookup("cache-dir")
	if flag == nil {
		t.Fatal("cache-dir flag was not registered")
	}
	if flag.DefValue != "" {
		t.Errorf("cache-dir default = %q, want no static default", flag.DefValue)
	}

	if err := flags.Set("cache-dir", "/custom/cache"); err != nil {
		t.Fatal(err)
	}
	if options.cacheDir != "/custom/cache" || !options.cacheDirSet {
		t.Errorf("cache-dir value = %q, cacheDirSet = %t; want /custom/cache and true", options.cacheDir, options.cacheDirSet)
	}
	if flag.DefValue != "" {
		t.Errorf("cache-dir default changed to %q after setting the flag", flag.DefValue)
	}
}

func TestCacheOptionsDirectory(t *testing.T) {
	tests := []struct {
		name         string
		xdgCacheHome string
		legacyEntry  string
		options      CacheOptions
		wantPath     string
		wantErr      string
	}{
		{
			name:         "legacy directory takes precedence",
			xdgCacheHome: "relative/xdg-cache",
			legacyEntry:  "directory",
			wantPath:     "legacy",
		},
		{
			name:         "uses XDG cache home",
			xdgCacheHome: "absolute",
			wantPath:     "xdg",
		},
		{
			name:     "falls back to XDG default",
			wantPath: "fallback",
		},
		{
			name:         "rejects relative XDG cache home",
			xdgCacheHome: "relative/xdg-cache",
			wantErr:      "XDG_CACHE_HOME",
		},
		{
			name:        "rejects non-directory legacy path",
			legacyEntry: "file",
			wantErr:     "is not a directory",
		},
		{
			name:         "explicit cache directory overrides XDG settings",
			xdgCacheHome: "relative/xdg-cache",
			options: CacheOptions{
				cacheDir: "~/custom-cache",
			},
			wantPath: "custom",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)

			xdgCacheHome := tc.xdgCacheHome
			switch xdgCacheHome {
			case "absolute":
				xdgCacheHome = filepath.Join(home, "xdg-cache")
			case "relative/xdg-cache":
			default:
				xdgCacheHome = ""
			}
			t.Setenv("XDG_CACHE_HOME", xdgCacheHome)

			legacyDir := filepath.Join(home, ".tekton")
			switch tc.legacyEntry {
			case "directory":
				if err := os.Mkdir(legacyDir, 0o700); err != nil {
					t.Fatal(err)
				}
			case "file":
				if err := os.WriteFile(legacyDir, nil, 0o600); err != nil {
					t.Fatal(err)
				}
			}

			got, err := tc.options.directory()
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("directory() error = %v, want an error containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			var want string
			switch tc.wantPath {
			case "legacy":
				want = filepath.Join(home, ".tekton", "bundles")
			case "xdg":
				want = filepath.Join(home, "xdg-cache", "tkn", "bundles")
			case "fallback":
				want = filepath.Join(home, ".cache", "tkn", "bundles")
			case "custom":
				want = filepath.Join(home, "custom-cache")
			}
			if got != want {
				t.Errorf("directory() = %q, want %q", got, want)
			}
		})
	}
}
