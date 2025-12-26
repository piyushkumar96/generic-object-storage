package object_storage

import ae "github.com/piyushkumar96/app-error"

// GCS (Google Cloud Storage) error definitions
var (
	GoogleCSBackendClient = ae.GetCustomErr("ERR_OS_GCS_1000",
		"failed to initialise the gcs client", false)
	GCSGetObjects = ae.GetCustomErr("ERR_OS_GCS_1001",
		"error while getting objects from gcs bucket", false)
	GCSGetObject = ae.GetCustomErr("ERR_OS_GCS_1002",
		"error while getting object from gcs bucket", false)
	GCSPutObject = ae.GetCustomErr("ERR_OS_GCS_1003",
		"error while putting object to gcs bucket", false)
	GCSDeleteObject = ae.GetCustomErr("ERR_OS_GCS_1004",
		"error while deleting object from gcs bucket", false)
	GCSCopyObject = ae.GetCustomErr("ERR_OS_GCS_1005",
		"error while copying object in gcs bucket", false)
)

// S3 (Amazon S3) error definitions
var (
	S3BackendClient = ae.GetCustomErr("ERR_OS_S3_2000",
		"failed to initialise the s3 client", false)
	S3GetObjects = ae.GetCustomErr("ERR_OS_S3_2001",
		"error while getting objects from s3 bucket", false)
	S3GetObject = ae.GetCustomErr("ERR_OS_S3_2002",
		"error while getting object from s3 bucket", false)
	S3PutObject = ae.GetCustomErr("ERR_OS_S3_2003",
		"error while putting object to s3 bucket", false)
	S3DeleteObject = ae.GetCustomErr("ERR_OS_S3_2004",
		"error while deleting object from s3 bucket", false)
	S3CopyObject = ae.GetCustomErr("ERR_OS_S3_2005",
		"error while copying object in s3 bucket", false)
)
