package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func GeneratePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignedClient := s3.NewPresignClient(s3Client)
	presignedObj, err := presignedClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}, s3.WithPresignExpires(expireTime))

	if err != nil {
		return "", err
	}

	return presignedObj.URL, nil
}
