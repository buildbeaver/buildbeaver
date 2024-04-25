package log

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"strings"
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/util"
)

func TestLogWriter(t *testing.T) {
	clk := clock.New()
	logRegistry, err := logger.NewLogRegistry("")
	assert.Nil(t, err)
	logFactory := logger.MakeLogrusLogFactoryStdOut(logRegistry)

	resourceID := models.NewJobID().ResourceID
	descriptor := models.NewLogDescriptor(models.NewTime(clk.Now()), models.LogDescriptorID{}, resourceID)

	entries := []*models.LogEntry{
		models.NewLogEntryLine(
			1,
			models.NewTime(clk.Now()),
			"Executing job foo...",
			1,
			nil),
		models.NewLogEntryBlock(
			2,
			models.NewTime(clk.Now()),
			"Pulling Docker image...",
			"block-foo",
			nil),
		models.NewLogEntryLine(
			3,
			models.NewTime(clk.Now()),
			"I am test text",
			1,
			models.OptionalResourceName("block-foo")),
	}
	buf, err := json.Marshal(entries)
	assert.Nil(t, err)

	t.Run("Successful write", testSuccess(logFactory, clk, descriptor, buf))
	t.Run("Error handling", testErrorHandling(logFactory, clk, descriptor, buf))
}

func testSuccess(
	logFactory logger.LogFactory,
	clk clock.Clock,
	descriptor *models.LogDescriptor,
	logEntryBuf []byte,
) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		blobStore := newTestBlobStore()
		blobStore.returnError = false

		logWriter := newWriter(logFactory, clk, DefaultWriterConfig, blobStore, descriptor)
		logWriter.Start()
		defer logWriter.Stop()

		err := logWriter.drain(ctx, bytes.NewReader(logEntryBuf))
		assert.Nil(t, err)
		// Drain performs a flush before returning

		logReader := newReader(ctx, logFactory, blobStore, &query{
			descriptors: []*models.LogDescriptor{descriptor},
			startSeqNo:  nil,
		})
		bytes, err := ioutil.ReadAll(logReader)
		assert.Nil(t, err)
		// should be at least one log entry in the array written to the blob store
		assert.True(t, len(bytes) > 2)
	}
}

func testErrorHandling(
	logFactory logger.LogFactory,
	clk clock.Clock,
	descriptor *models.LogDescriptor,
	logEntryBuf []byte,
) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		blobStore := newTestBlobStore()
		blobStore.returnError = true

		logWriter := newWriter(logFactory, clk, DefaultWriterConfig, blobStore, descriptor)
		logWriter.Start()
		defer logWriter.Stop()

		err := logWriter.drain(ctx, bytes.NewReader(logEntryBuf))
		// Drain performs a flush before returning, so error will be returned
		assert.Error(t, err)

		logReader := newReader(ctx, logFactory, blobStore, &query{
			descriptors: []*models.LogDescriptor{descriptor},
			startSeqNo:  nil,
		})
		bytes, err := ioutil.ReadAll(logReader)
		assert.Nil(t, err)
		// should be no log entries in the array written to the blob store, i.e. '[]'
		assert.True(t, len(bytes) == 2)
	}
}

type blob struct {
	models.BlobDescriptor
	data []byte
}

type testBlobStore struct {
	blobs       map[string]*blob
	returnError bool
}

func newTestBlobStore() *testBlobStore {
	return &testBlobStore{blobs: map[string]*blob{}}
}

func (s *testBlobStore) PutBlob(ctx context.Context, key string, source io.Reader) error {
	data, err := ioutil.ReadAll(source)
	if err != nil {
		return fmt.Errorf("error reading data data: %w", err)
	}
	if s.returnError {
		return fmt.Errorf("error for testing purposes")
	}
	s.blobs[key] = &blob{
		BlobDescriptor: models.BlobDescriptor{
			Key:       key,
			SizeBytes: int64(len(data)),
		},
		data: data,
	}
	return nil
}

func (s *testBlobStore) GetBlob(ctx context.Context, key string) (io.ReadCloser, error) {
	blob, ok := s.blobs[key]
	if !ok {
		return nil, gerror.NewErrNotFound(fmt.Sprintf("error %q does not exist", key))
	}
	return util.NewFakeCloser(bytes.NewReader(blob.data)), nil
}

func (s *testBlobStore) GetBlobRange(ctx context.Context, key string, offset, length int64) (io.ReadCloser, error) {
	blob, ok := s.blobs[key]
	if !ok {
		return nil, gerror.NewErrNotFound(fmt.Sprintf("error %q does not exist", key))
	}
	return util.NewFakeCloser(bytes.NewReader(blob.data[offset:length])), nil
}

func (s *testBlobStore) DeleteBlob(ctx context.Context, key string) error {
	delete(s.blobs, key)
	return nil
}

func (s *testBlobStore) ListBlobs(ctx context.Context, prefix string, marker string, pagination models.Pagination) ([]*models.BlobDescriptor, *models.Cursor, error) {
	var (
		keys   []string
		blobs  []*models.BlobDescriptor
		cursor *models.Cursor
	)
	for k, _ := range s.blobs {
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})
	for _, k := range keys {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		if pagination.Cursor != nil && pagination.Cursor.Marker >= k {
			continue
		}
		if pagination.Cursor == nil && marker != "" && marker >= k {
			continue
		}
		blobs = append(blobs, &s.blobs[k].BlobDescriptor)
		if len(blobs) == pagination.Limit {
			break
		}
	}
	if len(blobs) > 0 {
		cursor = &models.Cursor{
			Prev: nil,
			Next: &models.DirectionalCursor{
				Direction: models.CursorDirectionNext,
				Marker:    blobs[len(blobs)-1].Key,
			},
		}
	}
	return blobs, cursor, nil
}

func (s *testBlobStore) reset() {
	s.blobs = map[string]*blob{}
}

func (s *testBlobStore) delete(key string) {
	delete(s.blobs, key)
}
