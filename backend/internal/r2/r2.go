// Package r2 is a thin client for the Cloudflare R2 media bucket over its
// S3-compatible API.
package r2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// Cache-Control values. Media files are content-addressed (the hash is in the
// name), so they never change and can be cached for a year. The manifest is
// the one file that does change.
const (
	CacheImmutable = "public, max-age=31536000, immutable"
	CacheManifest  = "public, max-age=60"
)

// Config is the bucket and the credentials for it.
type Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	PublicURL       string // e.g. https://pub-….r2.dev, no trailing slash
}

// ConfigFromEnv reads R2_* variables and names any that are missing.
func ConfigFromEnv() (Config, error) {
	c := Config{
		AccountID:       os.Getenv("R2_ACCOUNT_ID"),
		AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
		Bucket:          os.Getenv("R2_BUCKET"),
		PublicURL:       strings.TrimRight(os.Getenv("R2_PUBLIC_URL"), "/"),
	}
	var missing []string
	for name, v := range map[string]string{
		"R2_ACCOUNT_ID": c.AccountID, "R2_ACCESS_KEY_ID": c.AccessKeyID,
		"R2_SECRET_ACCESS_KEY": c.SecretAccessKey, "R2_BUCKET": c.Bucket,
	} {
		if v == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return c, fmt.Errorf("missing %s (set them in .env at the repo root)", strings.Join(missing, ", "))
	}
	return c, nil
}

// Bucket talks to one R2 bucket.
type Bucket struct {
	s3     *s3.Client
	name   string
	public string
}

// New returns a client for the configured bucket.
func New(c Config) *Bucket {
	client := s3.New(s3.Options{
		Region:       "auto",
		BaseEndpoint: aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", c.AccountID)),
		Credentials:  credentials.NewStaticCredentialsProvider(c.AccessKeyID, c.SecretAccessKey, ""),
	})
	return &Bucket{s3: client, name: c.Bucket, public: c.PublicURL}
}

// PublicURL returns the public address of an object, or "" if no public URL
// is configured.
func (b *Bucket) PublicURL(key string) string {
	if b.public == "" {
		return ""
	}
	return b.public + "/" + key
}

// Exists reports whether an object is already in the bucket.
func (b *Bucket) Exists(ctx context.Context, key string) (bool, error) {
	_, err := b.s3.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &b.name, Key: &key})
	if err == nil {
		return true, nil
	}
	if isNotFound(err) {
		return false, nil
	}
	return false, fmt.Errorf("checking %s: %w", key, err)
}

// PutFile uploads a local file.
func (b *Bucket) PutFile(ctx context.Context, key, path, contentType, cacheControl string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	_, err = b.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        &b.name,
		Key:           &key,
		Body:          f,
		ContentLength: aws.Int64(st.Size()),
		ContentType:   &contentType,
		CacheControl:  &cacheControl,
	})
	if err != nil {
		return fmt.Errorf("uploading %s: %w", key, err)
	}
	return nil
}

// GetJSON decodes an object into v. It returns found=false if the object
// doesn't exist.
func (b *Bucket) GetJSON(ctx context.Context, key string, v any) (found bool, err error) {
	out, err := b.s3.GetObject(ctx, &s3.GetObjectInput{Bucket: &b.name, Key: &key})
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("reading %s: %w", key, err)
	}
	defer out.Body.Close()
	if err := json.NewDecoder(out.Body).Decode(v); err != nil {
		return true, fmt.Errorf("decoding %s: %w", key, err)
	}
	return true, nil
}

// PutJSON uploads v as indented JSON.
func (b *Bucket) PutJSON(ctx context.Context, key string, v any, cacheControl string) error {
	body, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = b.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        &b.name,
		Key:           &key,
		Body:          bytes.NewReader(body),
		ContentLength: aws.Int64(int64(len(body))),
		ContentType:   aws.String("application/json"),
		CacheControl:  &cacheControl,
	})
	if err != nil {
		return fmt.Errorf("uploading %s: %w", key, err)
	}
	return nil
}

// Object is one listed object.
type Object struct {
	Key  string
	Size int64
}

// List returns every object whose key starts with prefix.
func (b *Bucket) List(ctx context.Context, prefix string) ([]Object, error) {
	var objs []Object
	p := s3.NewListObjectsV2Paginator(b.s3, &s3.ListObjectsV2Input{Bucket: &b.name, Prefix: &prefix})
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing %s: %w", prefix, err)
		}
		for _, o := range page.Contents {
			objs = append(objs, Object{Key: aws.ToString(o.Key), Size: aws.ToInt64(o.Size)})
		}
	}
	return objs, nil
}

// Delete removes an object.
func (b *Bucket) Delete(ctx context.Context, key string) error {
	if _, err := b.s3.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &b.name, Key: &key}); err != nil {
		return fmt.Errorf("deleting %s: %w", key, err)
	}
	return nil
}

// SetCORS lets the given origins read the bucket from the browser. The site
// needs this because WebGL can only draw a cross-origin video or image into a
// texture when the response carries CORS headers, and fetch() of the
// manifest needs them too.
func (b *Bucket) SetCORS(ctx context.Context, origins []string) error {
	_, err := b.s3.PutBucketCors(ctx, &s3.PutBucketCorsInput{
		Bucket: &b.name,
		CORSConfiguration: &types.CORSConfiguration{CORSRules: []types.CORSRule{{
			AllowedOrigins: origins,
			AllowedMethods: []string{"GET", "HEAD"},
			AllowedHeaders: []string{"Range"},
			ExposeHeaders:  []string{"Content-Length", "Content-Range", "Accept-Ranges", "ETag"},
			MaxAgeSeconds:  aws.Int32(86400),
		}}},
	})
	if err != nil {
		return fmt.Errorf("setting CORS: %w", err)
	}
	return nil
}

func isNotFound(err error) bool {
	var nsk *types.NoSuchKey
	var nf *types.NotFound
	if errors.As(err, &nsk) || errors.As(err, &nf) {
		return true
	}
	var api smithy.APIError
	return errors.As(err, &api) && (api.ErrorCode() == "NotFound" || api.ErrorCode() == "NoSuchKey")
}
