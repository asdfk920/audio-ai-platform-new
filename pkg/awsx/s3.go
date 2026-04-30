package awsx

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var (
	S3Client *s3.Client
	cfg      aws.Config
)

type Config struct {
	Region   string
	Endpoint string // 用于 LocalStack
}

// Init 初始化 AWS 客户端
func Init(ctx context.Context, awsCfg Config) error {
	var err error

	if awsCfg.Endpoint != "" {
		// LocalStack 或自定义端点
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(awsCfg.Region),
			config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
						URL:           awsCfg.Endpoint,
						SigningRegion: awsCfg.Region,
					}, nil
				})),
		)
	} else {
		// 真实 AWS
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(awsCfg.Region),
		)
	}

	if err != nil {
		return err
	}

	S3Client = s3.NewFromConfig(cfg)
	return nil
}

// GetPresignedUploadURL 获取预签名上传 URL
func GetPresignedUploadURL(ctx context.Context, bucket, key string, expiration time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(S3Client)

	request, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiration))

	if err != nil {
		return "", err
	}

	return request.URL, nil
}

// GetPresignedDownloadURL 获取预签名下载 URL
func GetPresignedDownloadURL(ctx context.Context, bucket, key string, expiration time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(S3Client)

	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiration))

	if err != nil {
		return "", err
	}

	return request.URL, nil
}
