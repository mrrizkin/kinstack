package config

import (
	"fmt"
	"os"
	"strings"

	"kinstack/internal/pkg/assert"
)

// RustFSConfig configures the S3-compatible object storage backend used by the
// media slice. When Enabled is false the settings are ignored so a base
// checkout can run without object storage.
type RustFSConfig struct {
	Enabled   bool
	Endpoint  string
	PublicURL string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	UseSSL    bool
}

// Load reads RustFS settings from the environment.
func (c *RustFSConfig) Load() {
	c.Enabled = assert.OK(envBool("MEDIA_ENABLED", false))
	c.Endpoint = env("RUSTFS_ENDPOINT", "http://localhost:9000")
	c.PublicURL = env("RUSTFS_PUBLIC_URL", "http://localhost:9000")
	c.AccessKey = os.Getenv("RUSTFS_ACCESS_KEY")
	c.SecretKey = os.Getenv("RUSTFS_SECRET_KEY")
	c.Bucket = env("RUSTFS_BUCKET", "kinstack")
	c.Region = env("RUSTFS_REGION", "us-east-1")
	c.UseSSL = assert.OK(envBool("RUSTFS_USE_SSL", false))
}

// Validate checks the RustFS settings only when the media slice is enabled. An
// enabled slice with incomplete settings is a developer setup error and must
// fail at startup rather than degrade at runtime.
func (c RustFSConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if err := validateURL("RUSTFS_ENDPOINT", c.Endpoint, false); err != nil {
		return err
	}
	if err := validateURL("RUSTFS_PUBLIC_URL", c.PublicURL, false); err != nil {
		return err
	}
	if c.AccessKey == "" || c.SecretKey == "" || c.Bucket == "" || c.Region == "" {
		return fmt.Errorf("rustfs endpoint, credentials, bucket, and region are required")
	}
	if isProduction() {
		if placeholderSecret(c.AccessKey) || placeholderSecret(c.SecretKey) {
			return fmt.Errorf("rustfs credentials must be non-placeholder production secrets")
		}
		if !c.UseSSL || !strings.HasPrefix(strings.ToLower(c.Endpoint), "https://") {
			return fmt.Errorf(
				"RUSTFS_ENDPOINT must use HTTPS and RUSTFS_USE_SSL=true in production",
			)
		}
	}
	return nil
}
