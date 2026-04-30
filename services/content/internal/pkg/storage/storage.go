package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jacklau/audio-ai-platform/services/content/internal/config"
)

// Uploader 上传对象并返回 object key（不含 CDN 前缀）
type Uploader interface {
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	Delete(ctx context.Context, keys ...string) error
	PublicURL(objectKey string) string
}

func NewUploader(ctx context.Context, c config.Config) (Uploader, error) {
	switch strings.ToLower(strings.TrimSpace(c.Storage.Driver)) {
	case "s3", "minio":
		return newS3Uploader(ctx, c)
	default:
		return newLocalUploader(c)
	}
}

type localUploader struct {
	root    string
	cdnBase string
}

func newLocalUploader(c config.Config) (Uploader, error) {
	root := strings.TrimSpace(c.Local.Root)
	if root == "" {
		root = "./data/content-objects"
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir local storage: %w", err)
	}
	base := strings.TrimRight(strings.TrimSpace(c.Storage.CdnBaseUrl), "/")
	return &localUploader{root: root, cdnBase: base}, nil
}

func (l *localUploader) Put(_ context.Context, key string, body io.Reader, _ int64, _ string) error {
	path := filepath.Join(l.root, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, body)
	return err
}

func (l *localUploader) Delete(_ context.Context, keys ...string) error {
	for _, k := range keys {
		if k == "" {
			continue
		}
		_ = os.Remove(filepath.Join(l.root, filepath.FromSlash(k)))
	}
	return nil
}

func (l *localUploader) PublicURL(objectKey string) string {
	key := strings.TrimLeft(objectKey, "/")
	if l.cdnBase == "" {
		return "/" + key
	}
	return l.cdnBase + "/" + key
}

type s3Uploader struct {
	client *s3.Client
	bucket string
	cdn    string
}

func newS3Uploader(ctx context.Context, c config.Config) (Uploader, error) {
	if strings.TrimSpace(c.Storage.Bucket) == "" {
		return nil, fmt.Errorf("storage bucket required")
	}
	if strings.TrimSpace(c.Storage.AccessKey) == "" || strings.TrimSpace(c.Storage.SecretKey) == "" {
		return nil, fmt.Errorf("storage accessKey/secretKey required")
	}
	region := strings.TrimSpace(c.Storage.Region)
	if region == "" {
		region = "us-east-1"
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			c.Storage.AccessKey, c.Storage.SecretKey, "",
		)),
	)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if ep := strings.TrimSpace(c.Storage.Endpoint); ep != "" {
			o.BaseEndpoint = aws.String(ep)
		}
		o.UsePathStyle = c.Storage.UsePathStyle
	})
	base := strings.TrimRight(strings.TrimSpace(c.Storage.CdnBaseUrl), "/")
	return &s3Uploader{client: client, bucket: c.Storage.Bucket, cdn: base}, nil
}

func (s *s3Uploader) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	in := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   body,
	}
	if size > 0 {
		in.ContentLength = aws.Int64(size)
	}
	if contentType != "" {
		in.ContentType = aws.String(contentType)
	}
	_, err := s.client.PutObject(ctx, in)
	return err
}

func (s *s3Uploader) Delete(ctx context.Context, keys ...string) error {
	for _, k := range keys {
		if k == "" {
			continue
		}
		_, _ = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(k),
		})
	}
	return nil
}

func (s *s3Uploader) PublicURL(objectKey string) string {
	key := strings.TrimLeft(objectKey, "/")
	if s.cdn != "" {
		return s.cdn + "/" + key
	}
	return fmt.Sprintf("s3://%s/%s", s.bucket, key)
}
