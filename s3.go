package main
import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"errors"
)


var S3Client *s3.Client = nil
var S3Bucket string = ""

type S3Config struct {
	Region string
	Host string
	AccessKeyID string
	SecretAccessKey string
}

func ConnectToS3(config S3Config, bucket string) (*s3.Client, error) {
	if S3Client != nil {
		return S3Client, nil
	}

	cfg := aws.Config{
		Region:      config.Region,
		Credentials: credentials.NewStaticCredentialsProvider(config.AccessKeyID, config.SecretAccessKey, ""),
		BaseEndpoint: aws.String(config.Host),
	}
	S3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	S3Bucket = bucket
	return S3Client, nil
}

func CloseS3Connection() {
	S3Client = nil
}

func UploadFileToS3(s3Object *s3.PutObjectInput) error {
	if S3Client == nil {
		return errors.New("S3 client not connected")
	}
	_, err := S3Client.PutObject(context.TODO(), s3Object)
	if err != nil {
		return err
	}
	return nil
}

func GetS3Bucket() string {
	return S3Bucket
}