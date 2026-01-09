package media

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Service struct {
	Client        *s3.Client
	Presigner     *s3.PresignClient
	Bucket        string
	IsConfigured  bool
}

func NewS3Service() (*S3Service, error) {
	accountId := os.Getenv("R2_ACCOUNT_ID")
	accessKeyId := os.Getenv("R2_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("R2_SECRET_ACCESS_KEY")
	bucketName := os.Getenv("R2_BUCKET_NAME")

	if accountId == "" || accessKeyId == "" || accessKeySecret == "" || bucketName == "" {
		log.Printf("❌ R2 Config Missing: AccountID=%v, KeyID=%v, Secret=%v, Bucket=%v", 
            accountId != "", accessKeyId != "", accessKeySecret != "", bucketName != "")
		return &S3Service{IsConfigured: false}, nil
	}
    log.Println("✅ R2/S3 Configured Successfully")

	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: "https://" + accountId + ".eu.r2.cloudflarestorage.com",
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyId, accessKeySecret, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg)
	presigner := s3.NewPresignClient(client)

	return &S3Service{
		Client:       client,
		Presigner:    presigner,
		Bucket:       bucketName,
		IsConfigured: true,
	}, nil
}

func (s *S3Service) GeneratePresignedPutURL(key string, contentType string, expiry time.Duration) (string, error) {
	if !s.IsConfigured {
		return "", nil // Or error
	}

	req, err := s.Presigner.PresignPutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(s.Bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(expiry))

	if err != nil {
		return "", err
	}

	return req.URL, nil
}

func (s *S3Service) GeneratePresignedGetURL(key string, expiry time.Duration) (string, error) {
	if !s.IsConfigured {
		return "", nil
	}

	req, err := s.Presigner.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(60*time.Minute))

	if err != nil {
		return "", err
	}

	return req.URL, nil
}
