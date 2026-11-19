package anchor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"github.com/sopandgo/sopandgo/backend/internal/backup"
)

// ErrS3NotConfigured means no S3 bucket is saved under Settings → Backup.
var ErrS3NotConfigured = errors.New("anchoring needs an S3 bucket: configure one under Settings → Backup")

// maxReceiptObject bounds what we read back from S3 per object.
const maxReceiptObject = 2 << 20

// ObjectStore is where receipts are copied off the server.
type ObjectStore interface {
	Put(ctx context.Context, key string, body []byte, contentType string) error
	Get(ctx context.Context, key string) ([]byte, error)
	List(ctx context.Context, prefix string) ([]string, error)
	// ObjectLockEnabled reports whether the bucket has S3 Object Lock on.
	ObjectLockEnabled(ctx context.Context) (bool, error)
	// Prefix is the key prefix receipts for this instance are written under.
	Prefix(instanceID string) string
}

// OpenObjectStore returns the store for the bucket saved under Settings →
// Backup. Anchoring uses the bucket and credentials even when scheduled
// backups are switched off.
type OpenObjectStore func(ctx context.Context) (ObjectStore, error)

// S3Opener builds an OpenObjectStore from the backup S3 settings.
func S3Opener(settings *backup.S3SettingsStore) OpenObjectStore {
	return func(ctx context.Context) (ObjectStore, error) {
		cfg, err := settings.LoadConfig()
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(cfg.Bucket) == "" {
			return nil, ErrS3NotConfigured
		}
		client, err := backup.NewS3Client(ctx, cfg)
		if err != nil {
			return nil, err
		}
		return &s3Store{client: client, bucket: cfg.Bucket, root: strings.Trim(cfg.RootPrefix, "/")}, nil
	}
}

type s3Store struct {
	client *s3.Client
	bucket string
	root   string
}

func (s *s3Store) Prefix(instanceID string) string {
	p := "anchors/" + instanceID + "/"
	if s.root != "" {
		p = s.root + "/" + p
	}
	return p
}

func (s *s3Store) Put(ctx context.Context, key string, body []byte, contentType string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("upload %s: %w", key, err)
	}
	return nil
}

func (s *s3Store) Get(ctx context.Context, key string) ([]byte, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", key, err)
	}
	defer out.Body.Close()
	return io.ReadAll(io.LimitReader(out.Body, maxReceiptObject))
}

func (s *s3Store) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	p := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{Bucket: aws.String(s.bucket), Prefix: aws.String(prefix)})
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list %s: %w", prefix, err)
		}
		for _, o := range page.Contents {
			keys = append(keys, aws.ToString(o.Key))
		}
	}
	return keys, nil
}

func (s *s3Store) ObjectLockEnabled(ctx context.Context) (bool, error) {
	out, err := s.client.GetObjectLockConfiguration(ctx, &s3.GetObjectLockConfigurationInput{Bucket: aws.String(s.bucket)})
	if err != nil {
		// AWS answers ObjectLockConfigurationNotFoundError for a bucket without
		// Object Lock; S3-compatible stores answer InvalidRequest, NotImplemented
		// or AccessDenied instead. Any answer from the store means "not known to
		// be locked", which is a warning, not a failure. Network errors still fail.
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			return false, nil
		}
		return false, err
	}
	return out.ObjectLockConfiguration != nil &&
		out.ObjectLockConfiguration.ObjectLockEnabled == types.ObjectLockEnabledEnabled, nil
}
