package runner

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v2"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/runner/logging"
)

type ArtifactManager struct {
	local            bool
	hostWorkspaceDir string
	apiClient        APIClient
}

func NewArtifactManager(local bool,
	hostWorkspaceDir string,
	apiClient APIClient) *ArtifactManager {
	return &ArtifactManager{
		local:            local,
		hostWorkspaceDir: hostWorkspaceDir,
		apiClient:        apiClient,
	}
}

// UploadArtifacts uploads all artifacts produced by the job.
// Any errors encountered are wrapped in ErrArtifactUploadFailed error codes
func (b *ArtifactManager) UploadArtifacts(ctx *JobBuildContext, globalEnvVarsByName map[string]string) error {
	if ctx.IsJobIndirected() {
		return nil
	}
	if len(ctx.Job().Job.ArtifactDefinitions) == 0 {
		return nil
	}
	uploadLogger := ctx.LogPipeline().StructuredLogger().Wrap("artifact_upload", "Uploading artifacts...")
	var results *multierror.Error
	for _, artifactDefinition := range ctx.Job().Job.ArtifactDefinitions {
		for _, rawPath := range artifactDefinition.Paths {
			absolutePath := filepath.Join(
				b.hostWorkspaceDir,
				os.Expand(rawPath, func(key string) string {
					val, ok := globalEnvVarsByName[key]
					if ok {
						return val
					} else {
						return key
					}
				}))
			paths, err := doublestar.Glob(absolutePath) // TODO we should walk this ourselves worked on the streamed results
			if err != nil {
				results = multierror.Append(results, gerror.NewErrArtifactUploadFailed(fmt.Sprintf("error executing glob %q", rawPath), err))
				continue
			}
			for _, path := range paths {
				err := b.uploadArtifact(ctx, uploadLogger, artifactDefinition.GroupName, path)
				if err != nil {
					results = multierror.Append(results, gerror.NewErrArtifactUploadFailed("Failed uploading artifact", err))
				}
			}
		}
	}
	return results.ErrorOrNil()
}

// DownloadArtifacts downloads all artifacts that the step depends on to the workspace.
func (b *ArtifactManager) DownloadArtifacts(ctx *JobBuildContext) error {
	if b.local {
		// For local builds, artifacts will already be on the local machine's filesystem
		return nil
	}
	var downloadLogger *logging.StructuredLogger
	for _, jobDependency := range ctx.Job().Job.Depends {
		for _, dependency := range jobDependency.ArtifactDependencies {
			search := models.NewArtifactSearch()
			search.Workflow = &dependency.Workflow
			search.JobName = &dependency.JobName
			search.GroupName = &dependency.GroupName
			paginator, err := b.apiClient.SearchArtifacts(ctx.Ctx(), ctx.Job().Job.BuildID, search)
			if err != nil {
				return errors.Wrap(err, "error searching artifacts")
			}
			for paginator.HasNext() {
				artifacts, err := paginator.Next(ctx.Ctx())
				if err != nil {
					return errors.Wrap(err, "error getting next set of artifact search results")
				}
				for _, artifact := range artifacts {
					if downloadLogger == nil {
						// Only log when we have at least one artifact to download...
						downloadLogger = ctx.LogPipeline().StructuredLogger().Wrap("artifact_download", "Downloading artifacts...")
					}
					err := b.downloadArtifact(ctx, downloadLogger, artifact)
					if err != nil {
						return errors.Wrap(err, "error downloading artifact")
					}
				}
			}
		}
	}
	return nil
}

// downloadArtifact downloads a single artifact to the workspace.
func (b *ArtifactManager) downloadArtifact(ctx *JobBuildContext, downloadLogger *logging.StructuredLogger, artifact *models.Artifact) error {
	absolutePath := filepath.Join(b.hostWorkspaceDir, artifact.Path)
	exists, err := b.checkAndVerifyArtifact(artifact)
	if err != nil {
		// TODO A file exists at artifact path but it isn't the file we expect - what do we do?
		return err
	}
	if exists {
		downloadLogger.WriteLinef("Artifact already exists in workspace: %s", artifact.Path)
	} else {
		downloadLogger.WriteLinef("Downloading artifact (%d bytes) to: %s", artifact.Size, artifact.Path)
		reader, err := b.apiClient.GetArtifactData(ctx.Ctx(), artifact.ID)
		if err != nil {
			return errors.Wrap(err, "error getting data")
		}
		defer reader.Close()
		// TODO this is very permissive, but as we have no idea what the original permissions were this is
		//  the most likely to work. Perhaps we should be capturing permission (+user/group) information for
		//  each part in an artifacts paths so we can restore it correctly?
		err = os.MkdirAll(filepath.Dir(absolutePath), 0777)
		if err != nil {
			return fmt.Errorf("error creating artifact directory: %w", err)
		}
		file, err := os.Create(absolutePath)
		if err != nil {
			return errors.Wrap(err, "error opening artifact file for writing")
		}
		// TODO verify md5 sum
		_, err = io.Copy(file, reader)
		if err != nil {
			return errors.Wrap(err, "error writing artifact file")
		}
	}
	return nil
}

// checkAndVerifyArtifact verifies that if a file exists at the artifact path that it is
// the same file that was saved as an artifact. Returns true if a matching file exists or
// an error if a mismatched file exists.
func (b *ArtifactManager) checkAndVerifyArtifact(artifact *models.Artifact) (bool, error) {
	absolutePath := filepath.Join(b.hostWorkspaceDir, artifact.Path)
	stat, err := os.Stat(absolutePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // File doesn't exist - all good
		}
		return false, fmt.Errorf("error stating artifact: %w", err)
	}
	if uint64(stat.Size()) != artifact.Size {
		return false, errors.New("error artifact size mismatch")
	}
	file, err := os.Open(absolutePath)
	if err != nil {
		return false, err
	}
	var hash hash.Hash
	switch artifact.HashType {
	case models.HashTypeBlake2b:
		hash, err = blake2b.New256(nil)
		if err != nil {
			return false, err
		}
	case models.HashTypeMD5:
		hash = md5.New()
	default:
		return false, fmt.Errorf("error unsupported hash type: %s", artifact.HashType)
	}
	_, err = io.Copy(hash, file)
	if err != nil {
		return false, fmt.Errorf("error reading artifact file: %w", err)
	}
	hashStr := hex.EncodeToString(hash.Sum(nil))
	if artifact.Hash != hashStr {
		return false, errors.New("error artifact hash mismatch")
	}
	return true, nil // File does exist but it matches the artifact - all good
}

// uploadArtifact uploads a single artifact.
func (b *ArtifactManager) uploadArtifact(ctx *JobBuildContext, uploadLogger *logging.StructuredLogger, groupName models.ResourceName, absolutePath string) error {
	stat, err := os.Stat(absolutePath)
	if err != nil {
		return errors.Wrapf(err, "error stating artifact file at path %s", absolutePath)
	}
	if stat.IsDir() {
		return nil
	}
	file, err := os.Open(absolutePath)
	if err != nil {
		return errors.Wrap(err, "error opening artifact file for reading")
	}
	defer file.Close()
	if !b.local {
		uploadLogger.WriteLinef("Uploading artifact %s (%d bytes) from path %s...", groupName, stat.Size(), absolutePath)
	}
	relativePath, err := filepath.Rel(b.hostWorkspaceDir, absolutePath)
	if err != nil {
		return errors.Wrap(err, "error making relative path")
	}
	_, err = b.apiClient.CreateArtifact(
		ctx.Ctx(),
		ctx.Job().Job.ID,
		groupName,
		relativePath,
		file)
	if err != nil {
		return errors.Wrap(err, "error creating artifact")
	}
	return nil
}
