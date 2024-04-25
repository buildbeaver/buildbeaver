package runner

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/transport"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/runner/logging"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
)

type CheckoutInfo struct {
	Repo        *documents.Repo
	Commit      *documents.Commit
	Ref         string
	RepoSSHKey  []byte
	CheckoutDir string
}

type GitCheckoutManager struct {
	log               logger.Log
	repoLocksMu       sync.Mutex
	repoLocksByRepoID map[models.RepoID]*sync.Mutex
}

func NewGitCheckoutManager(factory logger.LogFactory) *GitCheckoutManager {
	return &GitCheckoutManager{
		repoLocksByRepoID: map[models.RepoID]*sync.Mutex{},
		log:               factory("git"),
	}
}

func (s *GitCheckoutManager) Checkout(ctx context.Context, checkout CheckoutInfo, log logging.LogPipeline) error {
	start := time.Now()
	checkoutLog := log.StructuredLogger().Wrap("job_git_checkout", "Setting up workspace...")
	ref := plumbing.ReferenceName(checkout.Ref)
	hash := plumbing.NewHash(checkout.Commit.SHA)

	mirrorPath, err := s.mirrorWith(ctx, checkout, plumbing.Revision(ref), hash, checkoutLog)
	if err != nil {
		return fmt.Errorf("error getting mirror: %w", err)
	}
	mirrorURL, _ := url.Parse(mirrorPath)
	mirrorURL.Scheme = "file"
	mirrorUri := mirrorURL.String()

	// For Windows, we cannot use the result of url.Parse as it results in a path that we cannot use as a git URL.
	// Instead, we can use the direct mirrorPath as the clone URL as it is already an absolute file path.
	//
	// This is a known bug in golang -- https://github.com/golang/go/issues/32456
	if runtime.GOOS == "windows" {
		mirrorUri = mirrorPath
	}

	checkoutLog.WriteLine("Checking out repo to workspace...")
	_, err = git.PlainCloneContext(ctx, checkout.CheckoutDir, false, &git.CloneOptions{
		URL:           mirrorUri,
		RemoteName:    "origin",
		ReferenceName: ref,
		SingleBranch:  true,
		// TODO ideally we'd be able to set this to 1 to do a much faster shallow clone, but if the job uses
		//  git to locate tags on the ref's lineage (like our version script does) it won't be able to find them.
		//  This behaviour needs to be configurable... for now we use the more compatible slower option.
		//Depth:         1,
		Tags: git.AllTags,
	})
	if err != nil {
		return fmt.Errorf("error cloning repo: %w", err)
	}
	checkoutLog.WriteLinef("Workspace setup completed in: %s", time.Now().Sub(start).Round(time.Millisecond))
	return nil
}

// getRepoMirrorLock returns mutex that should be held when modifying the repo mirror.
func (s *GitCheckoutManager) getRepoMirrorLock(repoID models.RepoID) *sync.Mutex {
	s.repoLocksMu.Lock()
	defer s.repoLocksMu.Unlock()
	repoMu, ok := s.repoLocksByRepoID[repoID]
	if !ok {
		repoMu = &sync.Mutex{}
		s.repoLocksByRepoID[repoID] = repoMu
	}
	return repoMu
}

// mirrorWith does the work necessary to find or create a mirror for the job's repo that contains revision + hash.
// Takes the corresponding mirror lock as needed.
// Returns the mirror's filesystem location.
func (s *GitCheckoutManager) mirrorWith(ctx context.Context, checkout CheckoutInfo, revision plumbing.Revision, hash plumbing.Hash, log *logging.StructuredLogger) (string, error) {
	mu := s.getRepoMirrorLock(checkout.Repo.ID)
	mu.Lock()
	defer mu.Unlock()
	mirror, mirrorPath, err := s.findOrCreateMirror(ctx, checkout, log)
	if err != nil {
		return "", fmt.Errorf("error finding or creating mirror: %w", err)
	}
	_, revisionErr := mirror.ResolveRevision(revision)
	_, hashErr := mirror.CommitObject(hash)
	if revisionErr != nil || hashErr != nil {
		err = s.updateMirror(ctx, checkout.RepoSSHKey, mirror, log)
		if err != nil {
			return "", fmt.Errorf("error updating mirror: %w", err)
		}
	}

	return mirrorPath, nil
}

// findOrCreateMirror ensures a mirror exists for the job's repo on disk.
// Assumes the corresponding mirror lock is held.
// Returns a handle to the mirror and its filesystem location.
func (s *GitCheckoutManager) findOrCreateMirror(ctx context.Context, checkout CheckoutInfo, log *logging.StructuredLogger) (*git.Repository, string, error) {
	path := s.getMirrorPath(checkout.Repo.ID)
	mirror, err := s.findMirror(path, log)
	if err != nil {
		return nil, "", fmt.Errorf("error finding mirror: %w", err)
	}
	if mirror == nil {
		mirror, err = s.createMirror(ctx, checkout, path, log)
		if err != nil {
			return nil, "", fmt.Errorf("error creating mirror: %w", err)
		}
	}
	return mirror, path, nil
}

// findMirror attempts to open the mirror at the specified path. If the mirror cannot be opened it is deleted.
// Assumes the corresponding mirror lock is held.
func (s *GitCheckoutManager) findMirror(mirrorPath string, log *logging.StructuredLogger) (*git.Repository, error) {
	fs := osfs.New(mirrorPath)
	storage := filesystem.NewStorage(fs, cache.NewObjectLRUDefault())
	mirror, err := git.Open(storage, fs)
	if err == nil {
		return mirror, nil
	} else if _, err := os.Stat(mirrorPath); !os.IsNotExist(err) {
		s.log.Warnf("Error opening mirror; Destroying: %v", err)
		err = s.deleteDirectory(mirrorPath)
		if err != nil {
			return nil, fmt.Errorf("error deleting bad mirror: %w", err)
		}
	}
	return nil, nil
}

// createMirror creates a new mirror for the job's repo at mirrorPath.
// Assumes the corresponding mirror lock is held.
func (s *GitCheckoutManager) createMirror(ctx context.Context, checkout CheckoutInfo, mirrorPath string, log *logging.StructuredLogger) (*git.Repository, error) {
	err := os.MkdirAll(mirrorPath, 0744)
	if err != nil {
		return nil, fmt.Errorf("error creating mirror path: %w", err)
	}
	auth, err := s.getRepoAuth(checkout.RepoSSHKey)
	if err != nil {
		return nil, fmt.Errorf("error getting repo auth: %w", err)
	}
	log.WriteLine("Cloning repo...")
	fs := osfs.New(mirrorPath)
	storage := filesystem.NewStorage(fs, cache.NewObjectLRUDefault())
	mirror, err := git.CloneContext(ctx, storage, nil, &git.CloneOptions{
		URL:        checkout.Repo.SSHURL,
		Auth:       auth,
		RemoteName: "origin",
		NoCheckout: true,
		Progress:   &progressWriter{log: log},
	})
	if err != nil {
		return nil, fmt.Errorf("error cloning mirror: %w", err)
	}
	return mirror, nil
}

// updateMirror fetches updates to all refs for a mirror repo.
// Assumes the corresponding mirror lock is held.
func (s *GitCheckoutManager) updateMirror(ctx context.Context, repoSSHKey []byte, repo *git.Repository, log *logging.StructuredLogger) error {
	auth, err := s.getRepoAuth(repoSSHKey)
	if err != nil {
		return fmt.Errorf("error getting repo auth: %w", err)
	}
	log.WriteLine("Fetching changes...")
	err = repo.FetchContext(ctx, &git.FetchOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{"refs/*:refs/*"},
		Auth:       auth,
		Force:      true,
		Progress:   &progressWriter{log: log},
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("error fetching mirror updates: %w", err)
	}
	return nil
}

// getRepoAuth returns the auth object that should be used to authenticate to GitHub
// when interacting with the job's repo.
func (s *GitCheckoutManager) getRepoAuth(repoSSHKey []byte) (transport.AuthMethod, error) {
	sshAuth, err := gitssh.NewPublicKeys("git", repoSSHKey, "")
	if err != nil {
		return nil, fmt.Errorf("error loading repo private key: %w", err)
	}
	sshAuth.HostKeyCallback = ssh.InsecureIgnoreHostKey() // TODO
	return sshAuth, nil
}

// deleteDirectory deletes a directory from disk (if it exists)
func (s *GitCheckoutManager) deleteDirectory(path string) error {
	err := os.RemoveAll(path)
	if err != nil && err != os.ErrNotExist {
		return fmt.Errorf("error deleting directory: %w", err)
	}
	return nil
}

// getMirrorBasePath makes a deterministic path on disk that can be used to store a git mirror.
func (s *GitCheckoutManager) getMirrorPath(repoID models.RepoID) string {
	return filepath.Join(os.TempDir(), "buildbeaver", "git-mirrors", models.SanitizeFilePathID(repoID.ResourceID))
}

type progressWriter struct {
	log   *logging.StructuredLogger
	count int
}

func (w *progressWriter) Write(p []byte) (int, error) {
	if strings.Contains(string(p), "objects:") && !strings.Contains(string(p), "100%") {
		if w.count%10 == 0 {
			w.log.WriteLine(string(p))
		}
		w.count++
	} else {
		w.log.WriteLine(string(p))
	}
	return len(p), nil
}
