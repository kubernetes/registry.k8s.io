package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
)

const (
	// gcsMetadataTimeout is the timeout when querying GCS object metadata (attributes)
	gcsMetadataTimeout = 5 * time.Second
	// gcsDataTimeout is the timeout when reading the full GCS object
	// Note that we only normally do this on smaller files - hash files
	gcsDataTimeout = 10 * time.Second
)

type fileCache struct {
	mutex sync.RWMutex
	// cache contains the data for all cached files
	// Though it is not bounded, we only intend to store the contents of hash files so we should be OK
	// (and the cloud run container will spin down after an idle period)
	cache map[GCSKey]*fileInfo
}

type fileInfo struct {
	mutex sync.Mutex
	body  []byte
	attrs *fileAttrs
}

type fileAttrs struct {
	Size int64
}

type GCSKey struct {
	Bucket    string
	ObjectKey string
}

func (k GCSKey) String() string {
	return urlJoin("gs://"+k.Bucket, k.ObjectKey)
}

func (k *GCSKey) Join(path string) GCSKey {
	joined := urlJoin(k.ObjectKey, path)
	joined = strings.TrimPrefix(joined, "/")
	return GCSKey{
		Bucket:    k.Bucket,
		ObjectKey: joined,
	}
}

// func (f *fileInfo) getStat(ctx context.Context, storageClient storage.Client, k GCSKey) (*fileAttrs, error) {
// 	f.mutex.Lock()
// 	defer f.mutex.Unlock()

// 	if f.attrs != nil {
// 		return f.attrs, nil
// 	}

// 	ctx, cancel := context.WithTimeout(ctx, gcsMetadataTimeout)
// 	defer cancel()

// 	attrs, err := storageClient.Bucket(k.Bucket).Object(k.ObjectKey).Attrs(ctx)
// 	if err != nil {
// 		// TODO: Cache negative lookups?
// 		return nil, fmt.Errorf("error reading metadata for %q: %w", k, err)
// 	}
// 	// We avoid storing the full attributes, they are quite large
// 	f.attrs = &fileAttrs{
// 		Size: attrs.Size,
// 	}
// 	return f.attrs, nil
// }

func (f *fileInfo) getContent(ctx context.Context, storageClient *storage.Client, k GCSKey) ([]byte, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.body != nil {
		return f.body, nil
	}

	ctx, cancel := context.WithTimeout(ctx, gcsDataTimeout)
	defer cancel()

	r, err := storageClient.Bucket(k.Bucket).Object(k.ObjectKey).NewReader(ctx)
	if err != nil {
		// TODO: Cache negative lookups?
		return nil, fmt.Errorf("error opening reader for %q: %w", k, err)
	}
	defer r.Close()

	// if f.attrs == nil {
	// 	f.attrs = &fileAttrs{
	// 		Size: r.Attrs.Size,
	// 	}
	// }

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error reading data for %q: %w", k, err)
	}
	f.body = data
	return data, nil
}

type contentCache struct {
	storageClient *storage.Client
	fileCache
}

func newContentCache(storageClient *storage.Client) *contentCache {
	return &contentCache{
		storageClient: storageClient,
		fileCache: fileCache{
			cache: make(map[GCSKey]*fileInfo),
		},
	}
}

// func (c *fileCache) getFileInfo(k GCSKey, create bool) *fileInfo {
// 	if !create {
// 		c.mutex.RLock()
// 		defer c.mutex.RUnlock()
// 		content, exists := c.cache[k]
// 		if !exists {
// 			return nil
// 		}
// 		return content
// 	} else {
// 		c.mutex.Lock()
// 		defer c.mutex.Unlock()
// 		content, exists := c.cache[k]
// 		if !exists {
// 			content = &fileInfo{}
// 			c.cache[k] = content
// 		}
// 		return content
// 	}
// }

func (c *fileCache) getFileInfo(k GCSKey) *fileInfo {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	content, exists := c.cache[k]
	if !exists {
		content = &fileInfo{}
		c.cache[k] = content
	}
	return content
}

func (c *contentCache) GetContents(ctx context.Context, k GCSKey) ([]byte, error) {
	fileInfo := c.fileCache.getFileInfo(k)

	data, err := fileInfo.getContent(ctx, c.storageClient, k)
	if err != nil {
		// Map to a cloud agnostic error
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	return data, nil
}
