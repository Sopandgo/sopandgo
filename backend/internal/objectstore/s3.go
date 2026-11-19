package objectstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// S3Config is the connection part of the S3 settings.
type S3Config struct {
	Bucket       string
	Region       string
	RootPrefix   string
	Endpoint     string // optional, e.g. MinIO
	UsePathStyle bool
	StaticKey    string // empty = AWS default credential chain
	StaticSecret string
}

// S3 is the production adapter. Building it does not contact the bucket.
type S3 struct {
	client *s3.Client
	bucket string
	root   string
}

// NewS3 builds a client for cfg's bucket.
func NewS3(ctx context.Context, cfg S3Config) (*S3, error) {
	opts := []func(*config.LoadOptions) error{}
	if cfg.Region != "" {
		opts = append(opts, config.WithRegion(cfg.Region))
	}
	if cfg.StaticKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.StaticKey, cfg.StaticSecret, ""),
		))
	}
	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}

	s3Opts := []func(*s3.Options){}
	if cfg.Endpoint != "" {
		ep := cfg.Endpoint
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(ep)
			o.UsePathStyle = cfg.UsePathStyle
		})
	} else if cfg.UsePathStyle {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}
	return &S3{
		client: s3.NewFromConfig(awsCfg, s3Opts...),
		bucket: cfg.Bucket,
		root:   strings.Trim(cfg.RootPrefix, "/"),
	}, nil
}

func (s *S3) Root() string { return s.root }

func (s *S3) Put(ctx context.Context, key string, body io.Reader, contentType string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("upload %s: %w", key, err)
	}
	return nil
}

func (s *S3) Get(ctx context.Context, key string) ([]byte, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", key, err)
	}
	defer out.Body.Close()
	return io.ReadAll(io.LimitReader(out.Body, MaxGetSize))
}

func (s *S3) List(ctx context.Context, prefix string) ([]Object, error) {
	var objs []Object
	p := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{Bucket: aws.String(s.bucket), Prefix: aws.String(prefix)})
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list %s: %w", prefix, err)
		}
		for _, o := range page.Contents {
			if o.Key == nil {
				continue
			}
			objs = append(objs, Object{Key: *o.Key, LastModified: aws.ToTime(o.LastModified)})
		}
	}
	sort.Slice(objs, func(i, j int) bool { return objs[i].Key < objs[j].Key })
	return objs, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)}); err != nil {
		return fmt.Errorf("delete %s: %w", key, err)
	}
	return nil
}

func (s *S3) ObjectLockEnabled(ctx context.Context) (bool, error) {
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
