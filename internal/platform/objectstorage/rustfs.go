// Package objectstorage provides the application boundary for S3-compatible
// object storage. The media slice depends on this boundary instead of the AWS
// SDK directly.
package objectstorage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"kinstack/config"
	"kinstack/internal/pkg/assert"
	applog "kinstack/internal/pkg/log"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3 "github.com/aws/aws-sdk-go-v2/service/s3"
	fibers3 "github.com/gofiber/storage/s3"
)

type Client struct {
	storage   *fibers3.Storage
	bucket    string
	publicURL string
}

type Dependency struct {
	Config config.RustFSConfig
	Logger applog.Log
}

// New creates a RustFS client through Fiber's S3-compatible storage adapter.
func New(d Dependency) *Client {
	cfg := d.Config
	assert.Assert(cfg.Endpoint != "", "rustfs endpoint is required")
	assert.Assert(cfg.AccessKey != "", "rustfs access key is required")
	assert.Assert(cfg.SecretKey != "", "rustfs secret key is required")
	assert.Assert(cfg.Bucket != "", "rustfs bucket is required")
	assert.Assert(d.Logger != nil, "logger is required")

	endpoint := cfg.Endpoint
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		scheme := "http"
		if cfg.UseSSL {
			scheme = "https"
		}
		endpoint = scheme + "://" + endpoint
	}
	storage := fibers3.New(fibers3.Config{
		Bucket:   cfg.Bucket,
		Endpoint: endpoint,
		Region:   cfg.Region,
		Credentials: fibers3.Credentials{
			AccessKey:       cfg.AccessKey,
			SecretAccessKey: cfg.SecretKey,
		},
	})
	client := &Client{
		storage:   storage,
		bucket:    cfg.Bucket,
		publicURL: strings.TrimRight(cfg.PublicURL, "/"),
	}
	d.Logger.Infow("RustFS configured", "endpoint", endpoint, "bucket", cfg.Bucket)
	return client
}

// EnsureBucket creates the application bucket when it does not exist.
func (c *Client) EnsureBucket(ctx context.Context) error {
	assert.Assert(c != nil && c.storage != nil, "rustfs client must be initialized")
	_, err := c.storage.Conn().HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(c.bucket)})
	if err == nil {
		return nil
	}
	if _, err = c.storage.Conn().CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(c.bucket),
	}); err != nil {
		return fmt.Errorf("create rustfs bucket %q: %w", c.bucket, err)
	}
	return nil
}

// Put stores an object.
func (c *Client) Put(
	ctx context.Context,
	key string,
	body io.Reader,
	size int64,
	contentType string,
) error {
	assert.Assert(c != nil && c.storage != nil, "rustfs client must be initialized")
	assert.Assert(key != "", "object key is required")
	assert.Assert(body != nil, "object body is required")
	assert.Assert(size >= 0, "object size must not be negative")

	_, err := c.storage.Conn().PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(c.bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("put rustfs object %q: %w", key, err)
	}
	return nil
}

// Get opens an object for streaming delivery.
func (c *Client) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	assert.Assert(c != nil && c.storage != nil, "rustfs client must be initialized")
	assert.Assert(key != "", "object key is required")

	output, err := c.storage.Conn().GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("get rustfs object %q: %w", key, err)
	}
	return output.Body, nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	assert.Assert(c != nil && c.storage != nil, "rustfs client must be initialized")
	assert.Assert(key != "", "object key is required")

	if _, err := c.storage.Conn().DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}); err != nil {
		return fmt.Errorf("delete rustfs object %q: %w", key, err)
	}
	return nil
}

// PublicURL returns the browser URL for an object delivered from the bucket.
func (c *Client) PublicURL(key string) string {
	assert.Assert(c != nil, "rustfs client must be initialized")
	return c.publicURL + "/" + url.PathEscape(c.bucket) + "/" + strings.TrimLeft(key, "/")
}

// PresignedGetURL creates a time-limited download URL for a private object.
func (c *Client) PresignedGetURL(
	ctx context.Context,
	key string,
	lifetime time.Duration,
) (*url.URL, error) {
	assert.Assert(c != nil && c.storage != nil, "rustfs client must be initialized")
	assert.Assert(key != "", "object key is required")
	assert.Assert(lifetime > 0, "presigned URL lifetime must be positive")

	presigner := s3.NewPresignClient(c.storage.Conn())
	result, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, func(options *s3.PresignOptions) {
		options.Expires = lifetime
	})
	if err != nil {
		return nil, fmt.Errorf("presign rustfs object %q: %w", key, err)
	}
	parsed, err := url.Parse(result.URL)
	if err != nil {
		return nil, fmt.Errorf("parse presigned RustFS URL: %w", err)
	}
	return parsed, nil
}
