package s3

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3 struct {
	s3Client *s3.S3
	bucket   string
}

func New(s3Client *s3.S3, bucket string) *S3 {
	return &S3{s3Client: s3Client, bucket: bucket}
}

// WriteFile writes a file
func (s S3) WriteFile(ctx context.Context, path string, r io.ReadSeeker) error {
	_, err := s.s3Client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Body:   r,
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return err
	}
	return nil
}

// ReadFile reads a file
func (s S3) ReadFile(ctx context.Context, path string, w io.Writer) error {
	result, err := s.s3Client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return err
	}
	_, err = io.Copy(w, result.Body)
	if err != nil {
		return err
	}
	return nil
}

// DeleteFile deletes a file
func (s S3) DeleteFile(ctx context.Context, path string) error {
	_, err := s.s3Client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return err
	}
	return nil
}
