package logic

import (
	"bytes"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type S3Uploader struct {
	uploader *s3manager.Uploader
	bucket   string
	region   string
}

func newAWSSession(region, endpoint, accessKey, secretKey string) (*session.Session, error) {
	config := &aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	}

	if endpoint != "" {
		config.Endpoint = aws.String(endpoint)
		config.S3ForcePathStyle = aws.Bool(true)
	}

	sess, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("创建 AWS Session 失败：%w", err)
	}

	return sess, nil
}

func newS3Uploader(sess *session.Session, bucket, region string) *S3Uploader {
	return &S3Uploader{
		uploader: s3manager.NewUploader(sess),
		bucket:   bucket,
		region:   region,
	}
}

func (u *S3Uploader) Upload(file io.Reader, key string) (string, error) {
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		return "", fmt.Errorf("读取文件失败：%w", err)
	}

	result, err := u.uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(buf.Bytes()),
	})
	if err != nil {
		return "", fmt.Errorf("上传到 S3 失败：%w", err)
	}

	url := result.Location

	if url == "" {
		url = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", u.bucket, u.region, key)
	}

	return url, nil
}
