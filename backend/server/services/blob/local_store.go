package blob

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/models"
	util2 "github.com/buildbeaver/buildbeaver/common/util"
)

type LocalBlobStoreDirectory string

func (l LocalBlobStoreDirectory) String() string {
	return string(l)
}

func (f LocalBlobStoreDirectory) Set(value string) error {
	f = LocalBlobStoreDirectory(value)
	return nil
}

type blobStoreFile struct {
	os.FileInfo
	// RelPath is a path to the file relative to the root of the blob store.
	// This path is unescaped and is guaranteed to use forward slashes.
	RelPath string
}

type LocalBlobStore struct {
	path string
}

func NewLocalBlobStore(path LocalBlobStoreDirectory) *LocalBlobStore {
	return &LocalBlobStore{
		path: string(path),
	}
}

// PutBlob writes all data in the source reader to a blob identified by key.
// The caller is responsible for closing the reader.
func (s *LocalBlobStore) PutBlob(ctx context.Context, key string, source io.Reader) error {
	if strings.HasPrefix(key, "/") {
		return fmt.Errorf("error blob keys cannot begin with /")
	}
	blobPath := s.makeBlobPath(key)
	err := os.MkdirAll(filepath.Dir(blobPath), 0700)
	if err != nil {
		return errors.Wrap(err, "error making blob directory")
	}
	blobFile, err := os.Create(blobPath)
	if err != nil {
		return errors.Wrapf(err, "Error opening blob %s for writing", blobPath)
	}
	defer blobFile.Close()
	_, err = io.Copy(blobFile, source)
	if err != nil {
		return errors.Wrapf(err, "Error writing data to blob %s", blobPath)
	}
	err = blobFile.Sync()
	if err != nil {
		return errors.Wrapf(err, "Error syncing blob %s", blobPath)
	}
	return nil
}

// GetBlob returns a reader positioned at the beginning of the blob identified by key.
// The caller is responsible for closing the reader.
func (s *LocalBlobStore) GetBlob(ctx context.Context, key string) (io.ReadCloser, error) {
	if strings.HasPrefix(key, "/") {
		return nil, fmt.Errorf("error blob keys cannot begin with /")
	}
	blobPath := s.makeBlobPath(key)
	blobFile, err := os.Open(blobPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, gerror.NewErrNotFound("Not Found").Wrap(err).IDetail("key", key)
		}
		return nil, errors.Wrapf(err, "Error opening blob %s for reading", blobPath)
	}
	return blobFile, nil
}

// GetBlobRange returns a reader positioned at the specified offset of the blob identified
// by key, which will read up to length bytes. The caller is responsible for closing the reader.
func (s *LocalBlobStore) GetBlobRange(ctx context.Context, key string, offset, length int64) (io.ReadCloser, error) {
	if strings.HasPrefix(key, "/") {
		return nil, fmt.Errorf("error blob keys cannot begin with /")
	}
	blobPath := s.makeBlobPath(key)
	blobFile, err := os.Open(blobPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, gerror.NewErrNotFound("Not Found").Wrap(err).IDetail("key", key)
		}
		return nil, errors.Wrapf(err, "Error opening blob %s for reading", blobPath)
	}
	if offset > 0 {
		_, err = blobFile.Seek(offset, 0)
		if err != nil {
			return nil, errors.Wrapf(err, "Unable to seek blob %s to offset %v", blobPath, offset)
		}
	}
	if length > 0 {
		return NewLimitReaderCloser(blobFile, length), nil
	}
	return blobFile, nil
}

// DeleteBlob deletes a blob. Returns nil if the blob does not exist.
func (s *LocalBlobStore) DeleteBlob(ctx context.Context, key string) error {
	if strings.HasPrefix(key, "/") {
		return fmt.Errorf("error blob keys cannot begin with /")
	}
	blobPath := s.makeBlobPath(key)
	err := os.Remove(blobPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("error deleting blob %s: %w", blobPath, err)
	}
	return nil
}

// ListBlobs lists blobs matching prefix. Use cursor to page through results, if any.
func (s *LocalBlobStore) ListBlobs(ctx context.Context, prefix string, marker string, pagination models.Pagination) ([]*models.BlobDescriptor, *models.Cursor, error) {
	// NOTE: This (and the rest of LocalBlobStore) provides a very naive mapping from objects to files
	// and is really only expected to be used in bb or tests (e.g. log volume). For the production server
	// we'll use an s3-based implementation.

	// NOTE: All inputs/outputs of the blob store use forward slash separators to be s3-compatible.
	// Internally however we're dealing with a filesystem that might use forward slashes (Unix systems)
	// or backslashes (Windows), so we need to convert to/from these two path styles as appropriate.

	if strings.HasPrefix(prefix, "/") {
		return nil, nil, fmt.Errorf("error blob keys cannot begin with /")
	}
	if pagination.Cursor != nil && pagination.Cursor.Direction != models.CursorDirectionNext {
		return nil, nil, fmt.Errorf("error only next markers are supported")
	}

	rootPath := s.makeBlobPath(filepath.Dir(prefix))

	_, err := os.Stat(rootPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("error stating root path: %w", err)
	}

	var listing []blobStoreFile
	err = filepath.Walk(rootPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(string(s.path), path)
			if err != nil {
				return fmt.Errorf("error getting relative path: %w", err)
			}
			unescaped, err := util2.UnescapeFileName(rel)
			if err != nil {
				return fmt.Errorf("error escaping path: %w", err)
			}
			listing = append(listing, blobStoreFile{FileInfo: info, RelPath: filepath.ToSlash(unescaped)})
			return nil
		})
	if err != nil {
		return nil, nil, fmt.Errorf("error during walk: %w", err)
	}

	var results []*models.BlobDescriptor
	for _, candidate := range listing {
		if !strings.HasPrefix(candidate.RelPath, prefix) {
			continue
		}
		if pagination.Cursor != nil && pagination.Cursor.Marker >= candidate.RelPath {
			continue
		}
		if pagination.Cursor == nil && marker != "" && marker >= candidate.RelPath {
			continue
		}
		results = append(results, &models.BlobDescriptor{Key: candidate.RelPath, SizeBytes: candidate.Size()})
		if len(results) >= pagination.Limit+1 { // read one more, so we can determine if a cursor should be returned
			break
		}
	}

	var cursor *models.Cursor
	if len(results) > pagination.Limit {
		results = results[:pagination.Limit]
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

// makeBlobPath makes a path to a blob on the local filesystem.
func (s *LocalBlobStore) makeBlobPath(key string) string {
	return filepath.Join(string(s.path), util2.EscapeFileName(key))
}
