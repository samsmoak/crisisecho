package repository

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type UploadRepository interface {
	GeneratePresignedURL(fileType, fileName, folder string) (string, string, string, error)
	GenerateRetrievalURL(filePath string, expiry time.Duration) (string, error)
	VerifyFileExists(filePath string) (bool, error)
	UploadFile(file io.Reader, fileName, fileType, folder string) (string, error)
}

type uploadRepository struct {
	s3Client *s3.S3
	uploader *s3manager.Uploader
	bucket   string
	region   string
}

func NewUploadRepository() UploadRepository {
	region := os.Getenv("AWS_S3_REGION")
	bucket := os.Getenv("AWS_S3_BUCKET")

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			os.Getenv("AWS_ACCESS_KEY_ID"),
			os.Getenv("AWS_SECRET_ACCESS_KEY"),
			"",
		),
	})
	if err != nil {
		panic("crisisecho: failed to create AWS session: " + err.Error())
	}

	return &uploadRepository{
		s3Client: s3.New(sess),
		uploader: s3manager.NewUploader(sess),
		bucket:   bucket,
		region:   region,
	}
}

func (r *uploadRepository) GeneratePresignedURL(fileType, fileName, folder string) (string, string, string, error) {
	s3Key := fmt.Sprintf("%s/%s/%d_%s", folder, fileType, time.Now().Unix(), fileName)

	req, _ := r.s3Client.PutObjectRequest(&s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(s3Key),
		ContentType: aws.String(fileType),
	})

	url, err := req.Presign(15 * time.Minute)
	if err != nil {
		return "", "", "", fmt.Errorf("UploadRepository.GeneratePresignedURL: %w", err)
	}

	publicURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", r.bucket, r.region, s3Key)
	return url, s3Key, publicURL, nil
}

func (r *uploadRepository) GenerateRetrievalURL(filePath string, expiry time.Duration) (string, error) {
	req, _ := r.s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(filePath),
	})

	url, err := req.Presign(expiry)
	if err != nil {
		return "", fmt.Errorf("UploadRepository.GenerateRetrievalURL: %w", err)
	}

	return url, nil
}

func (r *uploadRepository) VerifyFileExists(filePath string) (bool, error) {
	_, err := r.s3Client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(filePath),
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NotFound" {
			return false, nil
		}
		return false, fmt.Errorf("UploadRepository.VerifyFileExists: %w", err)
	}
	return true, nil
}

func (r *uploadRepository) UploadFile(file io.Reader, fileName, fileType, folder string) (string, error) {
	s3Key := fmt.Sprintf("%s/%s/%d_%s", folder, fileType, time.Now().Unix(), fileName)

	_, err := r.uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(s3Key),
		Body:        file,
		ContentType: aws.String(fileType),
	})
	if err != nil {
		return "", fmt.Errorf("UploadRepository.UploadFile: %w", err)
	}

	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", r.bucket, r.region, s3Key), nil
}
