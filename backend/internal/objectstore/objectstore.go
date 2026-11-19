// Package objectstore is the port both background jobs use to reach the S3
// bucket saved under Settings → Backup: automatic backups stream archives
// and prune old ones, audit anchoring copies receipts. S3 is the production
// adapter; Memory is for tests in any package.
package objectstore

import (
	"context"
	"io"
	"time"
)

// MaxGetSize bounds what Get reads back per object.
const MaxGetSize = 2 << 20

// Object is one listed object.
type Object struct {
	Key          string
	LastModified time.Time
}

// Store is one bucket, opened with one set of credentials.
type Store interface {
	// Put uploads body under key. A body that is also an io.Seeker (a file,
	// a bytes.Reader) is streamed without being buffered.
	Put(ctx context.Context, key string, body io.Reader, contentType string) error
	// Get reads at most MaxGetSize bytes of the object.
	Get(ctx context.Context, key string) ([]byte, error)
	// List returns every object whose key starts with prefix, sorted by key.
	List(ctx context.Context, prefix string) ([]Object, error)
	Delete(ctx context.Context, key string) error
	// ObjectLockEnabled reports whether the bucket has S3 Object Lock on.
	ObjectLockEnabled(ctx context.Context) (bool, error)
	// Root is the key prefix the admin saved, without leading or trailing
	// slashes. Each job builds its own keys below it.
	Root() string
}

// Opener opens the store from the settings saved now. Each job run opens it
// once and uses it throughout, so one run uses one bucket.
type Opener func(ctx context.Context) (Store, error)

// Join builds a key prefix from the root and further segments, ending in a
// slash. An empty root adds nothing.
func Join(root string, segments ...string) string {
	p := ""
	if root != "" {
		p = root + "/"
	}
	for _, s := range segments {
		p += s + "/"
	}
	return p
}
