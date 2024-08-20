/*
Copyright 2021 The Tekton Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package docdb

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"gocloud.dev/docstore"
	_ "gocloud.dev/docstore/awsdynamodb"
	_ "gocloud.dev/docstore/gcpfirestore"
	"gocloud.dev/docstore/mongodocstore"
	_ "gocloud.dev/docstore/mongodocstore"
	"knative.dev/pkg/logging"
)

const (
	StorageTypeDocDB = "docdb"
)

// ErrNothingToWatch is an error that's returned when the backend doesn't have anything to "watch"
var ErrNothingToWatch = fmt.Errorf("backend has nothing to watch")

// Backend is a storage backend that stores signed payloads in the TaskRun metadata as an annotation.
// It is stored as base64 encoded JSON.
type Backend struct {
	coll *docstore.Collection
}

type SignedDocument struct {
	Signed    []byte
	Signature string
	Cert      string
	Chain     string
	Object    interface{}
	Name      string
}

// NewStorageBackend returns a new Tekton StorageBackend that stores signatures on a TaskRun
func NewStorageBackend(ctx context.Context, cfg config.Config) (*Backend, error) {
	docdbURL := cfg.Storage.DocDB.URL

	u, err := url.Parse(docdbURL)
	if err != nil {
		return nil, err
	}

	if u.Scheme == mongodocstore.Scheme {
		// MONGO_SERVER_URL can be passed in as an environment variable or via config fields
		if err := populateMongoServerURL(ctx, cfg); err != nil {
			return nil, err
		}
	}

	coll, err := docstore.OpenCollection(ctx, docdbURL)
	if err != nil {
		return nil, err
	}

	return &Backend{
		coll: coll,
	}, nil
}

// WatchBackend returns a channel that receives a new Backend each time it needs to be updated
func WatchBackend(ctx context.Context, cfg config.Config, watcherStop chan bool) (chan *Backend, error) {
	logger := logging.FromContext(ctx)
	docDBURL := cfg.Storage.DocDB.URL

	u, err := url.Parse(docDBURL)
	if err != nil {
		return nil, err
	}

	// Set up the watcher only for mongo backends
	if u.Scheme != mongodocstore.Scheme {
		return nil, ErrNothingToWatch
	}

	// Set up watcher only when `storage.docdb.mongo-server-url-dir` is set
	if cfg.Storage.DocDB.MongoServerURLDir == "" {
		return nil, ErrNothingToWatch
	}

	logger.Infof("setting up fsnotify watcher for directory: %s", cfg.Storage.DocDB.MongoServerURLDir)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	pathsToWatch := []string{
		// mongo-server-url-dir/MONGO_SERVER_URL is where the MONGO_SERVER_URL environment
		// variable is expected to be mounted, either manually or via a Kubernetes secret, etc.
		filepath.Join(cfg.Storage.DocDB.MongoServerURLDir, "MONGO_SERVER_URL"),
		// When a Kubernetes secret is mounted on a path, the `data` in that secret is mounted
		// under path/..data that is then `symlink`ed to the key of the data. In this instance,
		// the mounted path is going to look like:
		// file 1 - ..2024_05_03_11_23_23.1253599725
		// file 2 - ..data -> ..2024_05_03_11_23_23.1253599725
		// file 3 - MONGO_SERVER_URL -> ..data/MONGO_SERVER_URL
		// So each time the secret is updated, the file `MONGO_SERVER_URL` is not updated,
		// instead the underlying symlink at `..data` is updated and that's what we want to
		// capture via the fsnotify event watcher
		filepath.Join(cfg.Storage.DocDB.MongoServerURLDir, "..data"),
	}

	backendChan := make(chan *Backend)
	// Start listening for events.
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				logger.Infof("received event: %s, path: %s", event.Op.String(), event.Name)
				// Only respond to create/write/remove events in the directory
				if !(event.Has(fsnotify.Create) || event.Has(fsnotify.Write) || event.Has(fsnotify.Remove)) {
					continue
				}

				if !slices.Contains(pathsToWatch, event.Name) {
					continue
				}

				updatedEnv, err := getMongoServerURLFromDir(cfg.Storage.DocDB.MongoServerURLDir)
				if err != nil {
					logger.Error(err)
					backendChan <- nil
				}
				if updatedEnv != os.Getenv("MONGO_SERVER_URL") {
					logger.Infof("directory %s has been updated, reconfiguring backend...", cfg.Storage.DocDB.MongoServerURLDir)

					// Now that MONGO_SERVER_URL has been updated, we should update docdb backend again
					newDocDBBackend, err := NewStorageBackend(ctx, cfg)
					if err != nil {
						logger.Error(err)
						backendChan <- nil
					} else {
						// Storing the backend in the signer so everyone has access to the up-to-date backend
						backendChan <- newDocDBBackend
					}
				} else {
					logger.Infof("MONGO_SERVER_URL has not changed in path: %s, backend will not be reconfigured", cfg.Storage.DocDB.MongoServerURLDir)
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.Error(err)

			case <-watcherStop:
				logger.Info("stopping fsnotify context...")
				return
			}
		}
	}()

	// Add a path.
	err = watcher.Add(cfg.Storage.DocDB.MongoServerURLDir)
	if err != nil {
		return nil, err
	}
	return backendChan, nil
}

// StorePayload implements the Payloader interface.
func (b *Backend) StorePayload(ctx context.Context, _ objects.TektonObject, rawPayload []byte, signature string, opts config.StorageOpts) error {
	var obj interface{}
	if err := json.Unmarshal(rawPayload, &obj); err != nil {
		return err
	}

	entry := SignedDocument{
		Signed:    rawPayload,
		Signature: base64.StdEncoding.EncodeToString([]byte(signature)),
		Object:    obj,
		Name:      opts.ShortKey,
		Cert:      opts.Cert,
		Chain:     opts.Chain,
	}

	if err := b.coll.Put(ctx, &entry); err != nil {
		return err
	}

	return nil
}

func (b *Backend) Type() string {
	return StorageTypeDocDB
}

func (b *Backend) RetrieveSignatures(ctx context.Context, _ objects.TektonObject, opts config.StorageOpts) (map[string][]string, error) {
	// Retrieve the document.
	documents, err := b.retrieveDocuments(ctx, opts)
	if err != nil {
		return nil, err
	}

	m := make(map[string][]string)
	for _, d := range documents {
		// Extract and decode the signature.
		sig, err := base64.StdEncoding.DecodeString(d.Signature)
		if err != nil {
			return nil, err
		}
		m[d.Name] = []string{string(sig)}
	}
	return m, nil
}

func (b *Backend) RetrievePayloads(ctx context.Context, _ objects.TektonObject, opts config.StorageOpts) (map[string]string, error) {
	documents, err := b.retrieveDocuments(ctx, opts)
	if err != nil {
		return nil, err
	}

	m := make(map[string]string)
	for _, d := range documents {
		m[d.Name] = string(d.Signed)
	}

	return m, nil
}

func (b *Backend) retrieveDocuments(ctx context.Context, opts config.StorageOpts) ([]SignedDocument, error) {
	d := SignedDocument{Name: opts.ShortKey}
	if err := b.coll.Get(ctx, &d); err != nil {
		return []SignedDocument{}, err
	}
	return []SignedDocument{d}, nil
}

func populateMongoServerURL(ctx context.Context, cfg config.Config) error {
	// First preference is given to the key `storage.docdb.mongo-server-url-dir`.
	// If that doesn't exist, then we move on to `storage.docdb.mongo-server-url`.
	// If that doesn't exist, then we check if `MONGO_SERVER_URL` env var is set.
	logger := logging.FromContext(ctx)
	mongoEnv := "MONGO_SERVER_URL"

	if cfg.Storage.DocDB.MongoServerURLDir != "" {
		logger.Infof("setting %s from storage.docdb.mongo-server-url-dir: %s", mongoEnv, cfg.Storage.DocDB.MongoServerURLDir)
		if err := setMongoServerURLFromDir(cfg.Storage.DocDB.MongoServerURLDir); err != nil {
			return err
		}
	} else if cfg.Storage.DocDB.MongoServerURL != "" {
		logger.Infof("setting %s from storage.docdb.mongo-server-url", mongoEnv)
		if err := os.Setenv(mongoEnv, cfg.Storage.DocDB.MongoServerURL); err != nil {
			return err
		}
	}

	if _, envExists := os.LookupEnv(mongoEnv); !envExists {
		return fmt.Errorf("mongo docstore configured but %s environment variable not set, "+
			"supply one of storage.docdb.mongo-server-url-dir, storage.docdb.mongo-server-url or set %s", mongoEnv, mongoEnv)
	}

	return nil
}

func setMongoServerURLFromDir(dir string) error {
	fileDataNormalized, err := getMongoServerURLFromDir(dir)
	if err != nil {
		return err
	}

	if err = os.Setenv("MONGO_SERVER_URL", fileDataNormalized); err != nil {
		return err
	}

	return nil
}

func getMongoServerURLFromDir(dir string) (string, error) {
	mongoEnv := "MONGO_SERVER_URL"

	stat, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// If directory does not exist, then create it. This is needed for
			// the fsnotify watcher.
			// fsnotify does not receive events if the path that it's watching
			// is created later.
			if err := os.MkdirAll(dir, 0755); err != nil {
				return "", err
			}
			return "", nil
		}
		return "", err
	}
	// If the path exists but is not a directory, then throw an error
	if !stat.IsDir() {
		return "", fmt.Errorf("path specified at storage.docdb.mongo-server-url-dir: %s is not a directory", dir)
	}

	filePath := filepath.Join(dir, mongoEnv)
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	// A trailing newline is fairly common in mounted files, let's remove it.
	fileDataNormalized := strings.TrimSuffix(string(fileData), "\n")

	return fileDataNormalized, nil
}
