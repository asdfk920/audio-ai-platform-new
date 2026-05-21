package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jacklau/audio-ai-platform/services/content/internal/config"

	"github.com/zeromicro/go-zero/core/logx"
)

// Uploader 上传对象并返回 object key（不含 CDN 前缀）
type Uploader interface {
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	Delete(ctx context.Context, keys ...string) error
	PublicURL(objectKey string) string
}

func NewUploader(ctx context.Context, c config.Config) (Uploader, error) {
	d := strings.ToLower(strings.TrimSpace(c.Storage.Driver))
	switch d {
	case "s3", "minio":
		u, err := newS3Uploader(ctx, c)
		if err == nil {
			logx.Infof("[storage] initialized driver=s3 bucket=%s", strings.TrimSpace(c.Storage.Bucket))
		}
		return u, err
	case "oss", "aliyun", "aliyunoss":
		u, err := newOSSUploader(c)
		if err == nil {
			logx.Infof("[storage] initialized driver=oss bucket=%s endpoint=%s", strings.TrimSpace(c.Storage.Bucket), normalizeOSSHost(c.Storage.Endpoint))
		}
		return u, err
	default:
		u, err := newLocalUploader(c)
		if err == nil {
			logx.Infof("[storage] initialized driver=local root=%s cdnBase=%s", strings.TrimSpace(c.Local.Root), strings.TrimSpace(c.Storage.CdnBaseUrl))
		}
		return u, err
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
		logx.Errorf("[storage:local] mkdir failed path=%s: %v", path, err)
		return err
	}
	logx.Infof("[storage:local] write path=%s", path)
	f, err := os.Create(path)
	if err != nil {
		logx.Errorf("[storage:local] create failed: %v", err)
		return err
	}
	defer f.Close()
	n, err := io.Copy(f, body)
	if err != nil {
		logx.Errorf("[storage:local] copy failed written=%d: %v", n, err)
		return fmt.Errorf("本地写入失败: %w", err)
	}
	logx.Infof("[storage:local] write ok bytes=%d path=%s", n, path)
	return nil
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
	logx.Infof("[storage:s3] PutObject bucket=%s key=%s size=%d", s.bucket, strings.TrimLeft(key, "/"), size)
	_, err := s.client.PutObject(ctx, in)
	if err != nil {
		logx.Errorf("[storage:s3] PutObject failed bucket=%s key=%s: %v", s.bucket, key, err)
		return fmt.Errorf("S3 上传失败（bucket=%s key=%s）: %w", s.bucket, key, err)
	}
	logx.Infof("[storage:s3] PutObject ok bucket=%s key=%s", s.bucket, key)
	return nil
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

type ossUploader struct {
	client       *oss.Client
	bucket       string
	cdn          string
	endpointHost string
}

func normalizeOSSHost(ep string) string {
	ep = strings.TrimSpace(ep)
	ep = strings.TrimPrefix(ep, "https://")
	ep = strings.TrimPrefix(ep, "http://")
	return strings.Trim(ep, "/")
}

func newOSSUploader(c config.Config) (Uploader, error) {
	ep := strings.TrimSpace(c.Storage.Endpoint)
	if ep == "" || strings.TrimSpace(c.Storage.Bucket) == "" ||
		strings.TrimSpace(c.Storage.AccessKey) == "" || strings.TrimSpace(c.Storage.SecretKey) == "" {
		return nil, fmt.Errorf("oss: Endpoint、Bucket、AccessKey、SecretKey 均不能为空")
	}
	host := normalizeOSSHost(ep)
	client, err := oss.New(host, strings.TrimSpace(c.Storage.AccessKey), strings.TrimSpace(c.Storage.SecretKey))
	if err != nil {
		logx.Errorf("[storage:oss] New client failed endpoint=%s: %v", host, err)
		return nil, fmt.Errorf("oss client: %w", err)
	}
	base := strings.TrimRight(strings.TrimSpace(c.Storage.CdnBaseUrl), "/")
	return &ossUploader{
		client:       client,
		bucket:       strings.TrimSpace(c.Storage.Bucket),
		cdn:          base,
		endpointHost: host,
	}, nil
}

func (o *ossUploader) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	key = strings.TrimLeft(key, "/")
	bucket, err := o.client.Bucket(o.bucket)
	if err != nil {
		logx.Errorf("[storage:oss] Bucket(%s) resolve failed: %v", o.bucket, err)
		return fmt.Errorf("oss 绑定 Bucket %q: %w", o.bucket, err)
	}
	var opts []oss.Option
	if contentType != "" {
		opts = append(opts, oss.ContentType(contentType))
	}
	if size > 0 {
		opts = append(opts, oss.ContentLength(size))
	}
	logx.Infof("[storage:oss] PutObject start bucket=%s key=%s size=%d content-type=%s", o.bucket, key, size, contentType)
	err = bucket.PutObject(key, body, opts...)
	if err != nil {
		logx.Errorf("[storage:oss] PutObject failed bucket=%s key=%s size=%d: %v（常见原因：Bucket 不存在、AK/SK 错误、签名不匹配、endpoint 不可用、RAM 未授权 PutObject）", o.bucket, key, size, err)
		return fmt.Errorf("OSS 上传失败（bucket=%s key=%s）: %w", o.bucket, key, err)
	}
	logx.Infof("[storage:oss] PutObject ok bucket=%s key=%s", o.bucket, key)
	return nil
}

func (o *ossUploader) Delete(ctx context.Context, keys ...string) error {
	bucket, err := o.client.Bucket(o.bucket)
	if err != nil {
		return err
	}
	for _, k := range keys {
		k = strings.TrimLeft(strings.TrimSpace(k), "/")
		if k == "" {
			continue
		}
		_ = bucket.DeleteObject(k)
	}
	return nil
}

func (o *ossUploader) PublicURL(objectKey string) string {
	key := strings.TrimLeft(objectKey, "/")
	if o.cdn != "" {
		return o.cdn + "/" + key
	}
	if o.endpointHost != "" && o.bucket != "" {
		return fmt.Sprintf("https://%s.%s/%s", o.bucket, o.endpointHost, key)
	}
	return fmt.Sprintf("oss://%s/%s", o.bucket, key)
}
