# Generic Object Storage

A unified Go library for performing CRUD operations on cloud object storage services. Supports **Google Cloud Storage (GCS)** and **Amazon S3** with a consistent interface.

## Features

- **Unified Interface**: Single `IStorageBackend` interface works with both GCS and S3
- **Full CRUD Operations**: Get, Put, Delete, Copy, and List objects
- **Context Support**: All operations accept context for cancellation and timeouts
- **Structured Errors**: Consistent error handling with detailed error codes
- **Prefix Support**: Organize objects with path prefixes
- **Testable**: Interfaces designed for easy mocking in tests

## Installation

```bash
go get github.com/piyushkumar96/generic-object-storage
```

## Quick Start

### Google Cloud Storage

```go
package main

import (
    "context"
    "log"

    storage "github.com/piyushkumar96/generic-object-storage"
)

func main() {
    ctx := context.Background()

    // Create GCS backend
    // Uses Application Default Credentials (set GOOGLE_APPLICATION_CREDENTIALS)
    backend, err := storage.NewGoogleCSBackend(ctx, "my-bucket", "optional/prefix")
    if err != nil {
        log.Fatal(err)
    }

    // Upload an object
    if err := backend.PutObject(ctx, "path/to/file.txt", []byte("Hello, World!")); err != nil {
        log.Fatal(err)
    }

    // Get an object
    obj, err := backend.GetObject(ctx, "path/to/file.txt")
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Content: %s", string(obj.Content))

    // List objects
    objects, err := backend.GetObjects(ctx, "path/")
    if err != nil {
        log.Fatal(err)
    }
    for _, o := range objects {
        log.Printf("Found: %s", o.Path)
    }

    // Copy an object
    if err := backend.CopyObject(ctx, "path/to/file.txt", "path/to/copy.txt"); err != nil {
        log.Fatal(err)
    }

    // Delete an object
    if err := backend.DeleteObject(ctx, "path/to/file.txt"); err != nil {
        log.Fatal(err)
    }
}
```

### Amazon S3

```go
package main

import (
    "context"
    "log"

    storage "github.com/piyushkumar96/generic-object-storage"
    "github.com/aws/aws-sdk-go/aws/credentials"
)

func main() {
    ctx := context.Background()

    // Option 1: Use default credentials (env vars, ~/.aws/credentials, IAM role)
    backend, err := storage.NewS3Backend("my-bucket", "optional/prefix", "us-east-1", false)
    if err != nil {
        log.Fatal(err)
    }

    // Option 2: Use explicit credentials
    creds := credentials.NewStaticCredentials("ACCESS_KEY", "SECRET_KEY", "")
    backend, err = storage.NewS3BackendWithCredentials("my-bucket", "prefix", "us-east-1", false, creds)
    if err != nil {
        log.Fatal(err)
    }

    // Option 3: Use custom endpoint (MinIO, LocalStack, etc.)
    creds := credentials.NewStaticCredentials("minioadmin", "minioadmin", "")
    backend, err = storage.NewS3BackendWithEndpoint(
        "my-bucket",
        "prefix",
        "us-east-1",
        "http://localhost:9000",
        true, // disableSSL
        creds,
    )
    if err != nil {
        log.Fatal(err)
    }

    // All operations are identical to GCS
    if err := backend.PutObject(ctx, "test.txt", []byte("Hello from S3!")); err != nil {
        log.Fatal(err)
    }
}
```

## API Reference

### Interface

```go
type IStorageBackend interface {
    GetObject(ctx context.Context, path string) (Object, *ae.AppError)
    GetObjects(ctx context.Context, prefix string) ([]Object, *ae.AppError)
    PutObject(ctx context.Context, path string, content []byte) *ae.AppError
    DeleteObject(ctx context.Context, path string) *ae.AppError
    CopyObject(ctx context.Context, srcPath, dstPath string) *ae.AppError
}
```

### Object Structure

```go
type Object struct {
    Meta         Metadata
    Path         string
    Content      []byte
    LastModified time.Time
}

type Metadata struct {
    Name    string
    Version string
}
```

### Constructor Functions

#### Google Cloud Storage

```go
// NewGoogleCSBackend creates a GCS backend using Application Default Credentials
func NewGoogleCSBackend(ctx context.Context, bucket string, prefix string) (*GoogleCSBackend, *ae.AppError)
```

#### Amazon S3

```go
// NewS3Backend creates an S3 backend using default credential chain
func NewS3Backend(bucket string, prefix string, region string, disableSSL bool) (*S3Backend, *ae.AppError)

// NewS3BackendWithCredentials creates an S3 backend with explicit credentials
func NewS3BackendWithCredentials(bucket string, prefix string, region string, disableSSL bool, creds *credentials.Credentials) (*S3Backend, *ae.AppError)

// NewS3BackendWithEndpoint creates an S3 backend with custom endpoint (for S3-compatible services)
func NewS3BackendWithEndpoint(bucket string, prefix string, region string, endpoint string, disableSSL bool, creds *credentials.Credentials) (*S3Backend, *ae.AppError)
```

## Error Handling

The library uses structured errors with error codes for easy identification:

### GCS Error Codes
| Code | Description |
|------|-------------|
| `ERR_OS_GCS_1000` | Failed to initialize GCS client |
| `ERR_OS_GCS_1001` | Error getting objects from GCS |
| `ERR_OS_GCS_1002` | Error getting single object from GCS |
| `ERR_OS_GCS_1003` | Error putting object to GCS |
| `ERR_OS_GCS_1004` | Error deleting object from GCS |
| `ERR_OS_GCS_1005` | Error copying object in GCS |

### S3 Error Codes
| Code | Description |
|------|-------------|
| `ERR_OS_S3_2000` | Failed to initialize S3 client |
| `ERR_OS_S3_2001` | Error getting objects from S3 |
| `ERR_OS_S3_2002` | Error getting single object from S3 |
| `ERR_OS_S3_2003` | Error putting object to S3 |
| `ERR_OS_S3_2004` | Error deleting object from S3 |
| `ERR_OS_S3_2005` | Error copying object in S3 |

## Authentication

### Google Cloud Storage

GCS uses [Application Default Credentials (ADC)](https://cloud.google.com/docs/authentication/application-default-credentials):

1. **Service Account Key** (recommended for production):
   ```bash
   export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account.json"
   ```

2. **User Credentials** (for development):
   ```bash
   gcloud auth application-default login
   ```

3. **Workload Identity** (for GKE)

### Amazon S3

S3 supports multiple authentication methods via the [AWS SDK credential chain](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html):

1. **Environment Variables**:
   ```bash
   export AWS_ACCESS_KEY_ID="your-access-key"
   export AWS_SECRET_ACCESS_KEY="your-secret-key"
   export AWS_REGION="us-east-1"
   ```

2. **Shared Credentials File** (`~/.aws/credentials`)

3. **IAM Role** (for EC2, ECS, Lambda)

4. **Explicit Credentials** (using `NewS3BackendWithCredentials`)

## Testing with Mocks

The library includes mock implementations for testing:

```go
import (
    "testing"
    "context"

    storage "github.com/piyushkumar96/generic-object-storage"
    "github.com/piyushkumar96/generic-object-storage/mocks"
)

func TestMyFunction(t *testing.T) {
    mockBackend := mocks.NewMockIStorageBackend(t)
    
    // Set up expectations
    mockBackend.On("GetObject", mock.Anything, "test.txt").Return(
        storage.Object{Content: []byte("test data")},
        nil,
    )
    
    // Use mockBackend in your tests
    obj, err := mockBackend.GetObject(context.Background(), "test.txt")
    // ... assertions
}
```

## Examples

See the [examples](./examples/) directory for complete working examples:

```bash
# Run GCS example
STORAGE_TYPE=gcs GCS_BUCKET=my-bucket go run examples/example.go

# Run S3 example  
STORAGE_TYPE=s3 S3_BUCKET=my-bucket AWS_REGION=us-east-1 go run examples/example.go
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
