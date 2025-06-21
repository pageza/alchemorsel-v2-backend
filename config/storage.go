package config

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Config holds S3 client and bucket info
type S3Config struct {
	Client     *s3.Client
	BucketName string
}

// NewS3Config initializes the S3 client using environment variables
func NewS3Config(ctx context.Context) (*S3Config, error) {
	bucket := os.Getenv("S3_BUCKET_NAME")
	if bucket == "" {
		bucket = "alchemorsel-profile-pictures" // default bucket name
	}

	// Load AWS config from environment or shared config
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(os.Getenv("AWS_REGION")),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg)

	return &S3Config{
		Client:     client,
		BucketName: bucket,
	}, nil
}

// SetupBucketPolicy applies a bucket policy to allow public read access
func (s *S3Config) SetupBucketPolicy(ctx context.Context) error {
	policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "PublicReadGetObject",
				"Effect": "Allow",
				"Principal": "*",
				"Action": "s3:GetObject",
				"Resource": "arn:aws:s3:::` + s.BucketName + `/*"
			}
		]
	}`
	_, err := s.Client.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
		Bucket: aws.String(s.BucketName),
		Policy: aws.String(policy),
	})
	return err
}

// GeneratePresignedURL generates a presigned URL for the given object key with the specified expiration time
func (s *S3Config) GeneratePresignedURL(ctx context.Context, objectKey string, expiration time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s.Client)
	presignedURL, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(objectKey),
	}, s3.WithPresignExpires(expiration))
	if err != nil {
		return "", err
	}
	return presignedURL.URL, nil
}
