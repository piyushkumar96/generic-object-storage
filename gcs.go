package object_storage

import (
	"cloud.google.com/go/storage"
	ae "github.com/piyushkumar96/app-error"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"io"
	"net/http"
	pathutil "path"
)

// IGCSClient this interface is added to make Client ins GCS BucketHandle mock compatible for tests
type IGCSClient interface {
	Objects(ctx context.Context, q *storage.Query) *storage.ObjectIterator
	Object(name string) *storage.ObjectHandle
}

// GoogleCSBackend is a storage backend for Google Cloud Storage
type GoogleCSBackend struct {
	Prefix string
	Client IGCSClient
}

// NewGoogleCSBackend creates a new instance of GoogleCSBackend
func NewGoogleCSBackend(ctx context.Context, bucket string, prefix string) (*GoogleCSBackend, *ae.AppError) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, ae.GetAppErr(ctx, err, GoogleCSBackendClient, http.StatusInternalServerError)
	}
	bucketHandle := client.Bucket(bucket)
	prefix = cleanPrefix(prefix)
	b := &GoogleCSBackend{
		Prefix: prefix,
		Client: bucketHandle,
	}
	return b, nil
}

// GetObject retrieves an object from Google Cloud Storage bucket, at prefix
func (b GoogleCSBackend) GetObject(ctx context.Context, path string) (Object, *ae.AppError) {
	var object Object
	object.Path = path
	objectHandle := b.Client.Object(pathutil.Join(b.Prefix, path))
	attrs, err := objectHandle.Attrs(ctx)
	if err != nil {
		appErr := ae.GetAppErr(ctx, err, GCSGetObject, http.StatusInternalServerError)
		if err.Error() == storage.ErrObjectNotExist.Error() {
			appErr = appErr.SetHTTPCode(http.StatusNotFound)
		}
		return object, appErr
	}
	object.LastModified = attrs.Updated
	rc, err := objectHandle.NewReader(ctx)
	if err != nil {
		return object, ae.GetAppErr(ctx, err, GCSGetObject, http.StatusInternalServerError)
	}
	content, err := io.ReadAll(rc)
	if err != nil {
		return object, ae.GetAppErr(ctx, errors.Wrap(err, "failed to read from reader stream"), GCSGetObject, http.StatusInternalServerError)
	}
	err = rc.Close()
	if err != nil {
		return object, ae.GetAppErr(ctx, err, GCSGetObject, http.StatusInternalServerError)
	}
	object.Content = content
	return object, nil
}

// GetObjects lists all objects in Google Cloud Storage bucket, at prefix
func (b GoogleCSBackend) GetObjects(ctx context.Context, prefix string) ([]Object, *ae.AppError) {
	var objects []Object
	prefix = pathutil.Join(b.Prefix, prefix)
	listQuery := &storage.Query{
		Prefix: prefix,
	}
	it := b.Client.Objects(ctx, listQuery)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			appErr := ae.GetAppErr(ctx, err, GCSGetObjects, http.StatusInternalServerError)
			if err.Error() == storage.ErrObjectNotExist.Error() {
				appErr = appErr.SetHTTPCode(http.StatusNotFound)
			}
			return objects, appErr
		}
		path := removePrefixFromObjectPath(prefix, attrs.Name)
		object := Object{
			Path:         path,
			Content:      []byte{},
			LastModified: attrs.Updated,
		}
		objects = append(objects, object)
	}
	return objects, nil
}

// PutObject uploads an object to Google Cloud Storage bucket, at prefix
func (b GoogleCSBackend) PutObject(ctx context.Context, path string, content []byte) *ae.AppError {
	wc := b.Client.Object(pathutil.Join(b.Prefix, path)).NewWriter(ctx)
	_, err := wc.Write(content)
	if err != nil {
		appErr := ae.GetAppErr(ctx, err, GCSPutObject, http.StatusInternalServerError)
		if err.Error() == storage.ErrObjectNotExist.Error() {
			appErr = appErr.SetHTTPCode(http.StatusNotFound)
		}
		return appErr
	}
	err = wc.Close()
	if err != nil {
		return ae.GetAppErr(ctx, err, GCSPutObject, http.StatusInternalServerError)
	}
	return nil
}

// DeleteObject removes an object from Google Cloud Storage bucket, at prefix
func (b GoogleCSBackend) DeleteObject(ctx context.Context, path string) *ae.AppError {
	err := b.Client.Object(pathutil.Join(b.Prefix, path)).Delete(ctx)
	if err != nil {
		appErr := ae.GetAppErr(ctx, err, GCSDeleteObject, http.StatusInternalServerError)
		if err.Error() == storage.ErrObjectNotExist.Error() {
			appErr = appErr.SetHTTPCode(http.StatusNotFound)
		}
		return appErr
	}
	return nil
}

// CopyObject copy an object from Google Cloud Storage bucket one path to another
func (b GoogleCSBackend) CopyObject(ctx context.Context, srcPath, dstPath string) *ae.AppError {
	src := b.Client.Object(srcPath)
	dst := b.Client.Object(dstPath)
	if _, err := dst.CopierFrom(src).Run(ctx); err != nil {
		appErr := ae.GetAppErr(ctx, err, GCSCopyObject, http.StatusInternalServerError)
		if err.Error() == storage.ErrObjectNotExist.Error() {
			appErr = appErr.SetHTTPCode(http.StatusNotFound)
		}
		return appErr
	}
	return nil
}
