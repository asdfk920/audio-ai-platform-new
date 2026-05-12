package logic

import (
	"fmt"
	"io"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type OSSClient struct {
	client *oss.Client
}

func newOSSClient(endpoint, accessKey, secretKey string) (*OSSClient, error) {
	client, err := oss.New(endpoint, accessKey, secretKey)
	if err != nil {
		return nil, err
	}

	return &OSSClient{client: client}, nil
}

func (o *OSSClient) Bucket(name string) *OSSBucket {
	bucket, err := o.client.Bucket(name)
	if err != nil {
		return &OSSBucket{bucket: nil, err: fmt.Errorf("获取 Bucket 失败：%w", err)}
	}
	return &OSSBucket{bucket: bucket}
}

type OSSBucket struct {
	bucket *oss.Bucket
	err    error
}

func (b *OSSBucket) PutObject(key string, reader io.Reader) error {
	if b.err != nil {
		return b.err
	}
	return b.bucket.PutObject(key, reader)
}
