package object_storage

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	pathutil "path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	ae "github.com/piyushkumar96/app-error"
)

// IS3Client interface for S3 client operations - allows mocking in tests
type IS3Client interface {
	ListObjectsWithContext(ctx aws.Context, input *s3.ListObjectsInput, opts ...request.Option) (*s3.ListObjectsOutput, error)
	GetObjectWithContext(ctx aws.Context, input *s3.GetObjectInput, opts ...request.Option) (*s3.GetObjectOutput, error)
	DeleteObjectWithContext(ctx aws.Context, input *s3.DeleteObjectInput, opts ...request.Option) (*s3.DeleteObjectOutput, error)
	CopyObjectWithContext(ctx aws.Context, input *s3.CopyObjectInput, opts ...request.Option) (*s3.CopyObjectOutput, error)
}

// IS3Uploader interface for S3 upload operations - allows mocking in tests
type IS3Uploader interface {
	UploadWithContext(ctx aws.Context, input *s3manager.UploadInput, opts ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error)
}

// S3Backend implements IStorageBackend for Amazon S3
type S3Backend struct {
	Bucket     string
	Client     IS3Client
	Downloader *s3manager.Downloader
	Prefix     string
	Uploader   IS3Uploader
}

// NewS3Backend creates a new instance of S3Backend using default credentials
func NewS3Backend(bucket string, prefix string, region string, disableSSL bool) (*S3Backend, *ae.AppError) {
	s, err := session.NewSession()
	if err != nil {
		return nil, ae.GetAppErr(context.Background(), err, S3BackendClient, http.StatusInternalServerError)
	}
	service := s3.New(s, &aws.Config{
		Region:     aws.String(region),
		DisableSSL: aws.Bool(disableSSL),
	})
	return &S3Backend{
		Bucket:     bucket,
		Client:     service,
		Downloader: s3manager.NewDownloaderWithClient(service),
		Prefix:     cleanPrefix(prefix),
		Uploader:   s3manager.NewUploaderWithClient(service),
	}, nil
}

// NewS3BackendWithCredentials creates a new instance of S3Backend with explicit credentials
func NewS3BackendWithCredentials(bucket string, prefix string, region string, disableSSL bool, creds *credentials.Credentials) (*S3Backend, *ae.AppError) {
	s, err := session.NewSession()
	if err != nil {
		return nil, ae.GetAppErr(context.Background(), err, S3BackendClient, http.StatusInternalServerError)
	}
	service := s3.New(s, &aws.Config{
		Credentials: creds,
		Region:      aws.String(region),
		DisableSSL:  aws.Bool(disableSSL),
	})
	return &S3Backend{
		Bucket:     bucket,
		Client:     service,
		Downloader: s3manager.NewDownloaderWithClient(service),
		Prefix:     cleanPrefix(prefix),
		Uploader:   s3manager.NewUploaderWithClient(service),
	}, nil
}

// NewS3BackendWithEndpoint creates a new instance of S3Backend with custom endpoint (for S3-compatible services like MinIO)
func NewS3BackendWithEndpoint(bucket string, prefix string, region string, endpoint string, disableSSL bool, creds *credentials.Credentials) (*S3Backend, *ae.AppError) {
	s, err := session.NewSession()
	if err != nil {
		return nil, ae.GetAppErr(context.Background(), err, S3BackendClient, http.StatusInternalServerError)
	}
	service := s3.New(s, &aws.Config{
		Credentials:      creds,
		Region:           aws.String(region),
		Endpoint:         aws.String(endpoint),
		DisableSSL:       aws.Bool(disableSSL),
		S3ForcePathStyle: aws.Bool(true),
	})
	return &S3Backend{
		Bucket:     bucket,
		Client:     service,
		Downloader: s3manager.NewDownloaderWithClient(service),
		Prefix:     cleanPrefix(prefix),
		Uploader:   s3manager.NewUploaderWithClient(service),
	}, nil
}

// GetObject retrieves an object from Amazon S3 bucket
func (b *S3Backend) GetObject(ctx context.Context, path string) (Object, *ae.AppError) {
	var object Object
	object.Path = path

	s3Input := &s3.GetObjectInput{
		Bucket: aws.String(b.Bucket),
		Key:    aws.String(pathutil.Join(b.Prefix, path)),
	}

	s3Result, err := b.Client.GetObjectWithContext(ctx, s3Input)
	if err != nil {
		appErr := ae.GetAppErr(ctx, err, S3GetObject, http.StatusInternalServerError)
		if isS3NotFoundError(err) {
			appErr = appErr.SetHTTPCode(http.StatusNotFound)
		}
		return object, appErr
	}
	defer s3Result.Body.Close()

	content, err := io.ReadAll(s3Result.Body)
	if err != nil {
		return object, ae.GetAppErr(ctx, err, S3GetObject, http.StatusInternalServerError)
	}

	object.Content = content
	if s3Result.LastModified != nil {
		object.LastModified = *s3Result.LastModified
	}
	return object, nil
}

// GetObjects lists all objects in Amazon S3 bucket at the given prefix
func (b *S3Backend) GetObjects(ctx context.Context, prefix string) ([]Object, *ae.AppError) {
	var objects []Object
	fullPrefix := pathutil.Join(b.Prefix, prefix)

	s3Input := &s3.ListObjectsInput{
		Bucket: aws.String(b.Bucket),
		Prefix: aws.String(fullPrefix),
	}

	for {
		s3Result, err := b.Client.ListObjectsWithContext(ctx, s3Input)
		if err != nil {
			appErr := ae.GetAppErr(ctx, err, S3GetObjects, http.StatusInternalServerError)
			if isS3NotFoundError(err) {
				appErr = appErr.SetHTTPCode(http.StatusNotFound)
			}
			return objects, appErr
		}

		for _, obj := range s3Result.Contents {
			path := removePrefixFromObjectPath(fullPrefix, *obj.Key)
			object := Object{
				Path:         path,
				Content:      []byte{},
				LastModified: *obj.LastModified,
			}
			objects = append(objects, object)
		}

		if s3Result.IsTruncated == nil || !*s3Result.IsTruncated {
			break
		}
		s3Input.Marker = s3Result.Contents[len(s3Result.Contents)-1].Key
	}

	return objects, nil
}

// PutObject uploads an object to Amazon S3 bucket
func (b *S3Backend) PutObject(ctx context.Context, path string, content []byte) *ae.AppError {
	s3Input := &s3manager.UploadInput{
		Bucket: aws.String(b.Bucket),
		Key:    aws.String(pathutil.Join(b.Prefix, path)),
		Body:   bytes.NewBuffer(content),
	}

	_, err := b.Uploader.UploadWithContext(ctx, s3Input)
	if err != nil {
		return ae.GetAppErr(ctx, err, S3PutObject, http.StatusInternalServerError)
	}
	return nil
}

// DeleteObject removes an object from Amazon S3 bucket
func (b *S3Backend) DeleteObject(ctx context.Context, path string) *ae.AppError {
	s3Input := &s3.DeleteObjectInput{
		Bucket: aws.String(b.Bucket),
		Key:    aws.String(pathutil.Join(b.Prefix, path)),
	}

	_, err := b.Client.DeleteObjectWithContext(ctx, s3Input)
	if err != nil {
		appErr := ae.GetAppErr(ctx, err, S3DeleteObject, http.StatusInternalServerError)
		if isS3NotFoundError(err) {
			appErr = appErr.SetHTTPCode(http.StatusNotFound)
		}
		return appErr
	}
	return nil
}

// CopyObject copies an object within Amazon S3 bucket
func (b *S3Backend) CopyObject(ctx context.Context, srcPath, dstPath string) *ae.AppError {
	copySource := pathutil.Join(b.Bucket, b.Prefix, srcPath)
	copyObjectInput := &s3.CopyObjectInput{
		Bucket:     aws.String(b.Bucket),
		CopySource: aws.String(url.PathEscape(copySource)),
		Key:        aws.String(pathutil.Join(b.Prefix, dstPath)),
	}

	_, err := b.Client.CopyObjectWithContext(ctx, copyObjectInput)
	if err != nil {
		appErr := ae.GetAppErr(ctx, err, S3CopyObject, http.StatusInternalServerError)
		if isS3NotFoundError(err) {
			appErr = appErr.SetHTTPCode(http.StatusNotFound)
		}
		return appErr
	}
	return nil
}

// isS3NotFoundError checks if the error is an S3 not found error
func isS3NotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "NoSuchKey") || contains(errStr, "NotFound") || contains(errStr, "404")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
