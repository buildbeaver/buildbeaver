package blob

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/util"
	"github.com/buildbeaver/buildbeaver/server/services"
)

func TestLocalStore(t *testing.T) {
	t.Run("ListBlobs/Local", testListBlobs(NewLocalBlobStore(LocalBlobStoreDirectory(t.TempDir()))))
}

func TestS3BlobStoreIntegration(t *testing.T) {
	t.Skip("Skipping S3 blob store integration test")

	if testing.Short() {
		t.Skip("Skipping S3 blob store integration test")
	}

	logRegistry, err := logger.NewLogRegistry("")
	assert.Nil(t, err)
	logFactory := logger.MakeLogrusLogFactoryStdOut(logRegistry)
	s3, err := NewS3BlobStore(S3BlobStoreConfig{
		BucketName: "buildbeaver-integration-test",
		Region:     "us-west-2",
	}, logFactory)
	assert.Nil(t, err)
	t.Run("ListBlobs/S3", testListBlobs(s3))
}

func testListBlobs(store services.BlobStore) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		skipIfAWSCredentialsNotFound(t, ctx, store)

		keys := []string{
			makeTestKey("foo/1"),
			makeTestKey("foo/2"),
			makeTestKey("foo/3"),
			makeTestKey("foo/bar/1"),
			makeTestKey("foo/bar/2"),
			makeTestKey("foo/bar/3"),
		}

		for _, key := range keys {
			err := store.PutBlob(ctx, key, bytes.NewBuffer([]byte{1}))
			require.Nil(t, err)
		}

		blobs, cursor, err := store.ListBlobs(ctx, makeTestKey("foo/"), "", models.Pagination{Limit: 2})
		require.Nil(t, err)
		require.Len(t, blobs, 2)
		require.NotNil(t, cursor)

		blobs, cursor, err = store.ListBlobs(ctx, makeTestKey("foo/"), "", models.Pagination{Limit: 2, Cursor: cursor.Next})
		require.Nil(t, err)
		require.Len(t, blobs, 2)
		require.NotNil(t, cursor)

		blobs, cursor, err = store.ListBlobs(ctx, makeTestKey("foo/"), "", models.Pagination{Limit: 2, Cursor: cursor.Next})
		require.Nil(t, err)
		require.Len(t, blobs, 2)
		require.Nil(t, cursor)

		for _, key := range keys {
			err := store.DeleteBlob(ctx, key)
			require.Nil(t, err)
		}
	}
}

var (
	keyPrefix string
	once      sync.Once
)

func makeTestKey(key string) string {
	once.Do(func() {
		timestamp := strconv.FormatInt(time.Now().UTC().Unix(), 10)
		keyPrefix = fmt.Sprintf("%s-%s/", timestamp, util.RandAlphaString(10))
	})
	return fmt.Sprintf("%s%s", keyPrefix, key)
}

func skipIfAWSCredentialsNotFound(t *testing.T, ctx context.Context, store services.BlobStore) {
	pingKey := makeTestKey("ping")
	err := store.PutBlob(ctx, pingKey, bytes.NewBuffer([]byte{1}))
	if err != nil && (strings.Contains(err.Error(), "EnvAccessKeyNotFound") ||
		strings.Contains(err.Error(), "SharedCredsLoad") ||
		strings.Contains(err.Error(), "NoCredentialProviders") ||
		strings.Contains(err.Error(), "InvalidAccessKeyId")) {
		t.Skip("Skipping S3 test as no AWS credentials found")
	}
	err = store.DeleteBlob(ctx, pingKey)
	require.Nil(t, err)
}
