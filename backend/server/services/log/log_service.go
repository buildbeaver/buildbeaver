package log

import (
	"context"
	"fmt"
	"io"

	"github.com/benbjohnson/clock"
	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/store"
)

type LogServiceConfig struct {
	WriterConfig WriterConfig
}

type LogService struct {
	log            logger.Log
	logFactory     logger.LogFactory
	clk            clock.Clock
	db             *store.DB
	config         LogServiceConfig
	blobStore      services.BlobStore
	logStore       store.LogStore
	ownershipStore store.OwnershipStore
}

func NewLogService(
	logFactory logger.LogFactory,
	clk clock.Clock,
	db *store.DB,
	config LogServiceConfig,
	blobStore services.BlobStore,
	logContainerStore store.LogStore,
	ownershipStore store.OwnershipStore) *LogService {

	return &LogService{
		log:            logFactory("LogService"),
		logFactory:     logFactory,
		clk:            clk,
		db:             db,
		config:         config,
		blobStore:      blobStore,
		logStore:       logContainerStore,
		ownershipStore: ownershipStore,
	}
}

// Create a new log descriptor.
// Returns store.ErrAlreadyExists if a log descriptor with matching unique properties already exists.
func (l *LogService) Create(ctx context.Context, txOrNil *store.Tx, log *models.LogDescriptor) (*models.LogDescriptor, error) {
	err := log.Validate()
	if err != nil {
		return nil, fmt.Errorf("error validating descriptor: %w", err)
	}
	err = l.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		err := l.logStore.Create(ctx, tx, log)
		if err != nil {
			return fmt.Errorf("error creating descriptor: %w", err)
		}
		ownership := models.NewOwnership(log.CreatedAt, log.ResourceID, log.GetID())
		err = l.ownershipStore.Create(ctx, tx, ownership)
		if err != nil {
			return errors.Wrap(err, "error creating ownership")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return log, nil
}

// Read an existing log descriptor, looking it up by ID.
// Returns models.ErrNotFound if the log descriptor does not exist.
func (l *LogService) Read(ctx context.Context, txOrNil *store.Tx, id models.LogDescriptorID) (*models.LogDescriptor, error) {
	return l.logStore.Read(ctx, txOrNil, id)
}

// Search all log descriptors. If searcher is set, the results will be limited to log descriptors the searcher
// is authorized to see (via the read:build permission). Use cursor to page through results, if any.
func (l *LogService) Search(ctx context.Context, txOrNil *store.Tx, searcher models.IdentityID, search models.LogDescriptorSearch) ([]*models.LogDescriptor, *models.Cursor, error) {
	return l.logStore.Search(ctx, txOrNil, searcher, search)
}

// WriteData pipes data from reader and writes it to the log descriptor's data.
func (l *LogService) WriteData(ctx context.Context, logDescriptorID models.LogDescriptorID, reader io.Reader) error {
	descriptor, err := l.logStore.Read(ctx, nil, logDescriptorID)
	if err != nil {
		return fmt.Errorf("error reading log descriptor: %w", err)
	}
	if descriptor.Sealed {
		return gerror.NewErrLogClosed()
	}
	writer := newWriter(l.logFactory, l.clk, l.config.WriterConfig, l.blobStore, descriptor)
	writer.Start()
	defer writer.Stop()
	return writer.drain(ctx, reader)
}

// ReadData opens a read stream to a log descriptor's data.
func (l *LogService) ReadData(ctx context.Context, logID models.LogDescriptorID, search *models.LogSearch) (io.ReadCloser, error) {
	var logs []*models.LogDescriptor
	if search.Expand != nil && *search.Expand {
		pagination := models.NewPagination(models.DefaultPaginationLimit, nil)
		all, cursor, err := l.Search(ctx, nil, models.NoIdentity, models.LogDescriptorSearch{Pagination: pagination, ParentLogID: &logID})
		if err != nil {
			return nil, fmt.Errorf("error searching log descriptors")
		}
		if cursor != nil && cursor.Next != nil {
			// If the number of logs for a single build exceeds the number of results in a single page then
			// we'll hit this. Not likely for some time...
			l.log.Warnf("Truncating log descriptor search results - this shouldn't be happening")
		}
		logs = all
	} else {
		log, err := l.Read(ctx, nil, logID)
		if err != nil {
			return nil, fmt.Errorf("error reading log descriptor")
		}
		logs = []*models.LogDescriptor{log}
	}

	plaintext := false
	if search.Plaintext != nil {
		plaintext = *search.Plaintext
	}
	reader := newReader(ctx, l.logFactory, l.blobStore, &query{
		descriptors: logs,
		startSeqNo:  search.StartSeqNo,
		plaintext:   plaintext,
	})
	return reader, nil
}

// Seal a log descriptor and its data, making it immutable going forward.
func (l *LogService) Seal(ctx context.Context, txOrNil *store.Tx, id models.LogDescriptorID) error {
	descriptor, err := l.logStore.Read(ctx, txOrNil, id)
	if err != nil {
		return fmt.Errorf("error reading log descriptor: %w", err)
	}
	if descriptor.Sealed {
		return fmt.Errorf("error descriptor is already sealed")
	}
	// TODO size calculation should probably be out of band in future as it can tie up a transaction
	//  for an extended period of time, and it shouldn't be in the critical path of finalizing a build/job/step.
	limit := 1000
	blobs, cursor, err := l.blobStore.ListBlobs(ctx, fmt.Sprintf(logChunkKeyBaseFormat, descriptor.ResourceID, descriptor.ID), "", models.NewPagination(limit, nil))
	if err != nil {
		return fmt.Errorf("error listing initial log parts: %w", err)
	}
	all := blobs
	for cursor != nil && cursor.Next != nil {
		blobs, cursor, err = l.blobStore.ListBlobs(ctx, fmt.Sprintf(logChunkKeyBaseFormat, descriptor.ResourceID, descriptor.ID), "", models.NewPagination(limit, cursor.Next))
		if err != nil {
			return fmt.Errorf("error listing log parts page: %w", err)
		}
		all = append(all, blobs...)
	}
	for _, blob := range all {
		descriptor.SizeBytes += blob.SizeBytes
	}
	descriptor.Sealed = true
	descriptor.UpdatedAt = models.NewTime(l.clk.Now())
	err = l.logStore.Update(ctx, txOrNil, descriptor)
	if err != nil {
		return fmt.Errorf("error updating log descriptor: %w", err)
	}
	return nil
}
