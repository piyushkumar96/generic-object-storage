package object_storage

import (
	"context"
	"time"

	ae "github.com/piyushkumar96/app-error"
)

// Object represents a storage object with its metadata and content
type Object struct {
	Meta         Metadata
	Path         string
	Content      []byte
	LastModified time.Time
}

// Metadata contains additional information about the object
type Metadata struct {
	Name    string
	Version string
}

// IStorageBackend defines the interface for storage backend implementations
// Both S3Backend and GoogleCSBackend implement this interface
type IStorageBackend interface {
	// GetObject retrieves a single object from the storage bucket
	GetObject(ctx context.Context, path string) (Object, *ae.AppError)
	// GetObjects lists all objects at the given prefix
	GetObjects(ctx context.Context, prefix string) ([]Object, *ae.AppError)
	// PutObject uploads an object to the storage bucket
	PutObject(ctx context.Context, path string, content []byte) *ae.AppError
	// DeleteObject removes an object from the storage bucket
	DeleteObject(ctx context.Context, path string) *ae.AppError
	// CopyObject copies an object from source path to destination path
	CopyObject(ctx context.Context, srcPath, dstPath string) *ae.AppError
}
