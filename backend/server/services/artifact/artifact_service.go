package artifact

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/h2non/filetype"
	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/util"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/store"
)

type ArtifactService struct {
	db                *store.DB
	artifactStore     store.ArtifactStore
	ownershipStore    store.OwnershipStore
	blobStore         services.BlobStore
	resourceLinkStore store.ResourceLinkStore
	logger.Log
}

func NewArtifactService(
	db *store.DB,
	artifactStore store.ArtifactStore,
	ownershipStore store.OwnershipStore,
	blobStore services.BlobStore,
	resourceLinkStore store.ResourceLinkStore,
	logFactory logger.LogFactory) *ArtifactService {

	return &ArtifactService{
		db:                db,
		artifactStore:     artifactStore,
		ownershipStore:    ownershipStore,
		blobStore:         blobStore,
		resourceLinkStore: resourceLinkStore,
		Log:               logFactory("ArtifactService"),
	}
}

// Read an existing artifact, looking it up by ID.
func (s *ArtifactService) Read(ctx context.Context, txOrNil *store.Tx, id models.ArtifactID) (*models.Artifact, error) {
	return s.artifactStore.Read(ctx, txOrNil, id)
}

// Create a new artifact with its contents provided by reader. It is the caller's responsibility to close reader.
// Optionally specify expectedMD5 to verify the file contents matches the expected MD5.
// If storeData is true then the artifact data obtained from the reader will be stored in the blob store.
func (s *ArtifactService) Create(
	ctx context.Context,
	jobID models.JobID,
	groupName models.ResourceName,
	relativePath string,
	expectedMD5 string,
	reader io.Reader,
	storeData bool,
) (*models.Artifact, error) {

	name, err := s.makeArtifactName(relativePath)
	if err != nil {
		return nil, fmt.Errorf("error creating artifact name: %w", err)
	}
	artifactData := models.NewArtifactData(models.NewTime(time.Now().UTC()), name, jobID, groupName, relativePath)
	artifact, _, err := s.findOrCreateArtifact(ctx, nil, artifactData) // NOTE: explicitly passing nil here. We don't want to hold a txn while we upload the data.

	if err != nil {
		return nil, fmt.Errorf("error creating artifact file: %w", err)
	}
	md5Hash := md5.New()
	countingReader := util.NewCountingReader(reader)
	hashingReader := newHashingReader(md5Hash, countingReader)
	key := s.makeArtifactKey(artifact.ID)

	if storeData {
		err = s.blobStore.PutBlob(ctx, key, hashingReader)
		if err != nil {
			return nil, fmt.Errorf("error writing artifact data to blob store: %w", err)
		}
	} else {
		// Read and discard the data, in order to get the count and hash
		_, err := io.Copy(io.Discard, hashingReader)
		if err != nil {
			return nil, fmt.Errorf("error reading artifact data: %w", err)
		}
	}

	calculatedMD5 := hex.EncodeToString(md5Hash.Sum(nil))
	if expectedMD5 != "" && strings.ToLower(expectedMD5) != calculatedMD5 {
		// TODO Delete blob
		// TODO Delete artifact
		return nil, fmt.Errorf("error MD5 mismatch. Expected %q, calculated %q", expectedMD5, calculatedMD5)
	}
	artifact.Sealed = true
	artifact.Size = countingReader.Count()
	artifact.Hash = calculatedMD5
	artifact.HashType = models.HashTypeMD5
	// artifact.Mime = // TODO sniff sniff
	return artifact, s.artifactStore.Update(ctx, nil, artifact)
}

// Search all artifacts. If searcher is set, the results will be limited to artifacts the searcher is authorized to
// see (via the read:artifact permission). Use cursor to page through results, if any.
func (s *ArtifactService) Search(ctx context.Context, txOrNil *store.Tx, searcher models.IdentityID, search models.ArtifactSearch) ([]*models.Artifact, *models.Cursor, error) {
	err := search.Validate()
	if err != nil {
		return nil, nil, errors.Wrap(err, "error validating search")
	}
	return s.artifactStore.Search(ctx, txOrNil, searcher, search)
}

// findOrCreateArtifact creates an artifact if no artifact with the same unique values exist,
// otherwise it reads and returns the existing artifact.
func (s *ArtifactService) findOrCreateArtifact(ctx context.Context, txOrNil *store.Tx, artifactData *models.ArtifactData) (artifact *models.Artifact, created bool, err error) {
	err = s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		artifact, created, err = s.artifactStore.FindOrCreate(ctx, tx, artifactData)
		if err != nil {
			return fmt.Errorf("error finding or creating artifact: %w", err)
		}
		if created {
			ownership := models.NewOwnership(models.NewTime(time.Now()), artifact.JobID.ResourceID, artifact.GetID())
			err = s.ownershipStore.Create(ctx, tx, ownership)
			if err != nil {
				return errors.Wrap(err, "error creating ownership")
			}
			_, _, err = s.resourceLinkStore.Upsert(ctx, tx, artifact)
			if err != nil {
				return fmt.Errorf("error upserting resource link: %w", err)
			}
			s.Infof("Created artifact %q", artifact.ID)
		}
		return nil
	})
	return artifact, created, err
}

// GetArtifactData returns a reader to the data of an artifact.
// It is the callers responsibility to close reader.
func (s *ArtifactService) GetArtifactData(ctx context.Context, artifactID models.ArtifactID) (io.ReadCloser, error) {
	key := s.makeArtifactKey(artifactID)
	return s.blobStore.GetBlob(ctx, key)
}

func (s *ArtifactService) makeArtifactKey(artifactID models.ArtifactID) string {
	return fmt.Sprintf("artifacts/%s", artifactID)
}

// getFileMimeType returns the mime type of the file or an error if it could not be determined.
// If error is nil the reader is guaranteed to be repositioned to offset 0. The caller is
// responsible for closing the reader.
// TODO adapt this to work from Create
func (s *ArtifactService) getFileMimeType(file io.ReadSeeker) (string, error) {
	headerRead := 0
	header := make([]byte, 261) // magic number from https://github.com/h2non/filetype
	for headerRead < len(header) {
		n, err := file.Read(header[headerRead:])
		headerRead += n
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", errors.Wrap(err, "error reading artifact file header")
		}
	}
	_, err := file.Seek(0, 0)
	if err != nil {
		return "", errors.Wrap(err, "error seeking to beginning of artifact file")
	}
	kind, err := filetype.Match(header[:headerRead])
	if err != nil {
		return "", errors.Wrap(err, "error determining artifact file mime type")
	}
	return kind.MIME.Type, nil
}

// makeArtifactName generates a deterministic name for an artifact based on the artifact's filepath.
func (s *ArtifactService) makeArtifactName(artifactRelativePath string) (models.ResourceName, error) {
	hash := sha256.New()
	_, err := hash.Write([]byte(artifactRelativePath))
	if err != nil {
		return "", fmt.Errorf("error hashing artifact path: %w", err)
	}
	return models.ResourceName(hex.EncodeToString(hash.Sum(nil)[:18])), nil
}
