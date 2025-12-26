package main

import (
	"context"
	"fmt"
	"log"
	"os"

	storage "github.com/piyushkumar96/generic-object-storage"

	"github.com/aws/aws-sdk-go/aws/credentials"
)

func main() {
	ctx := context.Background()

	// Choose which example to run based on environment
	storageType := os.Getenv("STORAGE_TYPE")
	switch storageType {
	case "s3":
		runS3Example(ctx)
	case "gcs":
		runGCSExample(ctx)
	default:
		fmt.Println("Set STORAGE_TYPE environment variable to 's3' or 'gcs'")
		fmt.Println("Example: STORAGE_TYPE=gcs go run example.go")
	}
}

// runGCSExample demonstrates Google Cloud Storage operations
func runGCSExample(ctx context.Context) {
	bucketName := os.Getenv("GCS_BUCKET")
	if bucketName == "" {
		log.Fatal("GCS_BUCKET environment variable is required")
	}

	objectPrefix := os.Getenv("GCS_PREFIX") // optional, can be empty

	// Create GCS backend client
	// Note: Uses Application Default Credentials (ADC)
	// Set GOOGLE_APPLICATION_CREDENTIALS env var to your service account key file
	backend, appErr := storage.NewGoogleCSBackend(ctx, bucketName, objectPrefix)
	if appErr != nil {
		log.Fatalf("Failed to create GCS backend: %v", appErr)
	}

	demonstrateOperations(ctx, backend, "GCS")
}

// runS3Example demonstrates Amazon S3 operations
func runS3Example(ctx context.Context) {
	bucketName := os.Getenv("S3_BUCKET")
	if bucketName == "" {
		log.Fatal("S3_BUCKET environment variable is required")
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	objectPrefix := os.Getenv("S3_PREFIX") // optional, can be empty

	var backend *storage.S3Backend
	var appErr error

	// Check if custom credentials are provided
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	if accessKey != "" && secretKey != "" {
		// Use explicit credentials
		creds := credentials.NewStaticCredentials(accessKey, secretKey, "")
		backend, appErr = storage.NewS3BackendWithCredentials(bucketName, objectPrefix, region, false, creds)
	} else {
		// Use default credentials chain (env vars, shared config, IAM role, etc.)
		backend, appErr = storage.NewS3Backend(bucketName, objectPrefix, region, false)
	}

	if appErr != nil {
		log.Fatalf("Failed to create S3 backend: %v", appErr)
	}

	demonstrateOperations(ctx, backend, "S3")
}

// demonstrateOperations shows common storage operations
func demonstrateOperations(ctx context.Context, backend storage.IStorageBackend, providerName string) {
	fmt.Printf("\n=== %s Storage Operations ===\n\n", providerName)

	testPath := "test/hello.txt"
	testContent := []byte("Hello, World! This is a test file.")

	// 1. Put Object
	fmt.Printf("1. Uploading object to '%s'...\n", testPath)
	if err := backend.PutObject(ctx, testPath, testContent); err != nil {
		log.Fatalf("Failed to put object: %v", err)
	}
	fmt.Println("   ✓ Object uploaded successfully")

	// 2. Get Object
	fmt.Printf("\n2. Retrieving object from '%s'...\n", testPath)
	obj, err := backend.GetObject(ctx, testPath)
	if err != nil {
		log.Fatalf("Failed to get object: %v", err)
	}
	fmt.Printf("   ✓ Content: %s\n", string(obj.Content))
	fmt.Printf("   ✓ Last Modified: %v\n", obj.LastModified)

	// 3. List Objects
	fmt.Println("\n3. Listing objects in 'test/' prefix...")
	objects, err := backend.GetObjects(ctx, "test/")
	if err != nil {
		log.Fatalf("Failed to list objects: %v", err)
	}
	fmt.Printf("   ✓ Found %d object(s):\n", len(objects))
	for _, o := range objects {
		fmt.Printf("     - %s (modified: %v)\n", o.Path, o.LastModified)
	}

	// 4. Copy Object
	copyPath := "test/hello-copy.txt"
	fmt.Printf("\n4. Copying object to '%s'...\n", copyPath)
	if err := backend.CopyObject(ctx, testPath, copyPath); err != nil {
		log.Fatalf("Failed to copy object: %v", err)
	}
	fmt.Println("   ✓ Object copied successfully")

	// 5. Delete Objects
	fmt.Println("\n5. Cleaning up - deleting test objects...")
	if err := backend.DeleteObject(ctx, testPath); err != nil {
		log.Fatalf("Failed to delete object: %v", err)
	}
	fmt.Printf("   ✓ Deleted '%s'\n", testPath)

	if err := backend.DeleteObject(ctx, copyPath); err != nil {
		log.Fatalf("Failed to delete copied object: %v", err)
	}
	fmt.Printf("   ✓ Deleted '%s'\n", copyPath)

	fmt.Printf("\n=== %s Operations Complete ===\n", providerName)
}
