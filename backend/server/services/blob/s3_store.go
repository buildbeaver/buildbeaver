package blob

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
)

const (
	AWSS3BlobStoreType BlobStoreType = "AWS_S3"
	LocalBlobStoreType BlobStoreType = "LOCAL"
)

type BlobStoreType string

func (s BlobStoreType) String() string {
	return string(s)
}

func BlobStoreTypes() []string {
	return []string{AWSS3BlobStoreType.String(), LocalBlobStoreType.String()}
}

type S3BlobStoreConfig struct {
	BucketName      string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
}

type S3BlobStore struct {
	s3       *s3.S3
	uploader *s3manager.Uploader
	config   S3BlobStoreConfig
	log      logger.Log
}

func NewS3BlobStore(config S3BlobStoreConfig, logFactory logger.LogFactory) (*S3BlobStore, error) {
	if config.BucketName == "" {
		return nil, fmt.Errorf("error bucket name must be configured")
	}
	log := logFactory("AWSS3BlobStore")
	cfg := &aws.Config{}
	log.Infof("Using bucket: %s", config.BucketName)
	if config.Region != "" {
		log.Infof("Using region: %s", config.Region)
		cfg = cfg.WithRegion(config.Region)
	} else {
		log.Info("Using default region")
	}
	if config.AccessKeyID != "" && config.SecretAccessKey != "" {
		log.Infof("Using static credentials: %s", config.AccessKeyID)
		cfg = cfg.WithCredentials(credentials.NewStaticCredentials(config.AccessKeyID, config.SecretAccessKey, ""))
	} else {
		log.Infof("Using default credentials")
	}
	sess, err := session.NewSession(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating AWS session: %w", err)
	}
	uploader := s3manager.NewUploader(sess)
	return &S3BlobStore{
		s3:       s3.New(sess),
		uploader: uploader,
		config:   config,
		log:      log,
	}, nil
}

// PutBlob writes all data in the source reader to a blob identified by key.
// The caller is responsible for closing the reader.
func (s *S3BlobStore) PutBlob(ctx context.Context, key string, source io.Reader) error {
	input := &s3manager.UploadInput{
		Body:                 source,
		Bucket:               aws.String(s.config.BucketName),
		ContentMD5:           nil, // TODO
		ContentType:          aws.String("application/octet-stream"),
		Key:                  aws.String(key),
		ServerSideEncryption: aws.String("AES256"),
	}
	// NOTE For future selves: This will use multipart uploads if it needs to. If the upload fails it
	// will attempt to clean up the parts. This cleanup can fail for a variety of reasons, so we may
	// find we accumulate some dead parts over time and will need to have a background process remove them.
	out, err := s.uploader.UploadWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf("error putting blob %s: %s", key, err)
	}
	s.log.WithField("bucket", s.config.BucketName).
		WithField("key", key).
		WithField("upload_id", out.UploadID).
		Infof("Uploaded object")
	return nil
}

// GetBlob returns a reader positioned at the beginning of the blob identified by key.
// The caller is responsible for closing the reader.
func (s *S3BlobStore) GetBlob(ctx context.Context, key string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(key),
	}
	output, err := s.s3.GetObjectWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error getting blob %s: %s", key, err)
	}
	s.log.WithField("bucket", s.config.BucketName).
		WithField("key", key).
		Infof("Read object")
	return output.Body, nil
}

// GetBlobRange returns a reader positioned at the specified offset of the blob identified
// by key, which will read up to length bytes. The caller is responsible for closing the reader.
func (s *S3BlobStore) GetBlobRange(ctx context.Context, key string, offset, length int64) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(key),
		Range:  aws.String(fmt.Sprintf("%d-%d", offset, offset+length-1)),
	}
	output, err := s.s3.GetObjectWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error getting blob range %s: %s", key, err)
	}
	s.log.WithField("bucket", s.config.BucketName).
		WithField("key", key).
		Infof("Read object range")
	return output.Body, nil
}

// DeleteBlob deletes a blob. Returns nil if the blob does not exist.
func (s *S3BlobStore) DeleteBlob(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(key),
	}
	_, err := s.s3.DeleteObjectWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf("error deleting blob %s: %s", key, err)
	}
	s.log.WithField("bucket", s.config.BucketName).
		WithField("key", key).
		Infof("Deleted object")
	return nil
}

// ListBlobs lists blobs matching prefix, starting at marker. Use cursor to page through results, if any.
func (s *S3BlobStore) ListBlobs(ctx context.Context, prefix string, marker string, pagination models.Pagination) ([]*models.BlobDescriptor, *models.Cursor, error) {
	if strings.HasPrefix(prefix, "/") {
		return nil, nil, fmt.Errorf("error blob keys cannot begin with /")
	}
	if pagination.Cursor != nil && pagination.Cursor.Direction != models.CursorDirectionNext {
		return nil, nil, fmt.Errorf("error only next markers are supported")
	}
	if pagination.Cursor != nil {
		marker = pagination.Cursor.Marker
	}
	input := &s3.ListObjectsInput{
		Bucket:  aws.String(s.config.BucketName),
		Marker:  aws.String(marker),
		MaxKeys: aws.Int64(int64(pagination.Limit)),
		Prefix:  aws.String(prefix),
	}
	output, err := s.s3.ListObjectsWithContext(ctx, input)
	if err != nil {
		return nil, nil, fmt.Errorf("error listing blobs prefix=%s marker=%s: %w", prefix, marker, err)
	}
	s.log.
		WithField("bucket", s.config.BucketName).
		WithField("marker", marker).
		WithField("prefix", prefix).
		WithField("results", len(output.Contents)).
		Infof("Listed objects")
	var results []*models.BlobDescriptor
	for _, obj := range output.Contents {
		results = append(results, &models.BlobDescriptor{Key: *obj.Key, SizeBytes: *obj.Size})
	}
	var cursor *models.Cursor
	if *output.IsTruncated {
		cursor = &models.Cursor{
			Prev: nil,
			Next: &models.DirectionalCursor{
				Direction: models.CursorDirectionNext,
				Marker:    results[len(results)-1].Key,
			},
		}
	}
	return results, cursor, nil
}
