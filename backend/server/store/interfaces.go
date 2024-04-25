package store

import (
	"context"

	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/models/search"
)

type LegalEntityStore interface {
	// Create a new legal entity.
	// Returns store.ErrAlreadyExists if a legal entity with matching unique properties already exists.
	Create(ctx context.Context, txOrNil *Tx, legalEntityData *models.LegalEntityData) (*models.LegalEntity, error)
	// Read an existing legal entity, looking it up by ID.
	// Returns models.ErrNotFound if the legal entity does not exist.
	Read(ctx context.Context, txOrNil *Tx, id models.LegalEntityID) (*models.LegalEntity, error)
	// ReadByName reads an existing legal entity, looking it up by its name.
	// Returns models.ErrNotFound if the legal entity does not exist.
	ReadByName(ctx context.Context, txOrNil *Tx, name models.ResourceName) (*models.LegalEntity, error)
	// ReadByExternalID reads an existing legal entity, looking it up by its external id.
	// Returns models.ErrNotFound if the legal entity does not exist.
	ReadByExternalID(ctx context.Context, txOrNil *Tx, externalID models.ExternalResourceID) (*models.LegalEntity, error)
	// FindOrCreate creates a legal entity if no legal entity with the same External ID already exists,
	// otherwise it reads and returns the existing legal entity.
	// Returns the legal entity as it is in the database, and true iff a new legal entity was created.
	FindOrCreate(ctx context.Context, txOrNil *Tx, legalEntityData *models.LegalEntityData) (legalEntity *models.LegalEntity, created bool, err error)
	// Update an existing legal entity with optimistic locking. Overrides all previous values using the supplied model.
	// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
	Update(ctx context.Context, txOrNil *Tx, legalEntity *models.LegalEntity) error
	// Upsert creates a legal entity if no legal entity with the same External ID already exists, otherwise it updates
	// the existing legal entity's data if it differs from the supplied data.
	// Returns the LegalEntity as it exists in the database after the create or update, and
	// true,false if the resource was created, false,true if the resource was updated, or false,false if
	// neither create nor update was necessary.
	Upsert(ctx context.Context, txOrNil *Tx, legalEntityData *models.LegalEntityData) (*models.LegalEntity, bool, bool, error)
	// ListParentLegalEntities lists all legal entities a legal entity is a member of. Use cursor to page through results, if any.
	ListParentLegalEntities(ctx context.Context, txOrNil *Tx, legalEntityID models.LegalEntityID, pagination models.Pagination) ([]*models.LegalEntity, *models.Cursor, error)
	// ListMemberLegalEntities lists all legal entities that are members of a parent legal entity. Use cursor to page through results, if any.
	ListMemberLegalEntities(ctx context.Context, txOrNil *Tx, parentLegalEntityID models.LegalEntityID, pagination models.Pagination) ([]*models.LegalEntity, *models.Cursor, error)
	// ListAllLegalEntities lists all legal entities in the system. Use cursor to page through results, if any.
	ListAllLegalEntities(ctx context.Context, txOrNil *Tx, pagination models.Pagination) ([]*models.LegalEntity, *models.Cursor, error)
}

type LegalEntityMembershipStore interface {
	// Create a new legal entity membership.
	// Returns store.ErrAlreadyExists if a legal entity membership with matching unique properties already exists.
	Create(ctx context.Context, txOrNil *Tx, membership *models.LegalEntityMembership) error
	// ReadByMember reads an existing legal entity membership, looking it up by (parent) legal entity
	// and member legal entity.
	// Returns models.ErrNotFound if the legal entity membership does not exist.
	ReadByMember(ctx context.Context, txOrNil *Tx, legalEntityID models.LegalEntityID, memberLegalEntityID models.LegalEntityID) (*models.LegalEntityMembership, error)
	// FindOrCreate finds and returns the legal entity membership with the (parent) legal entity
	// and member legal entity specified in the supplied membership data.
	// If no such membership exists then a new one is created and returned, and true is returned for 'created'.
	FindOrCreate(ctx context.Context, txOrNil *Tx, membershipData *models.LegalEntityMembership) (membership *models.LegalEntityMembership, created bool, err error)
	// DeleteByMember removes a member legal entity from a (parent) legal entity by deleting the relevant
	// membership record. This method is idempotent.
	DeleteByMember(ctx context.Context, txOrNil *Tx, legalEntityID models.LegalEntityID, memberLegalEntityID models.LegalEntityID) error
}

type IdentityStore interface {
	// Create a new Identity.
	// Returns store.ErrAlreadyExists if an Identity with matching ID already exists.
	Create(ctx context.Context, txOrNil *Tx, identity *models.Identity) error
	// Read an existing Identity, looking it up by IdentityID.
	// Returns models.ErrNotFound if the Identity does not exist.
	Read(ctx context.Context, txOrNil *Tx, id models.IdentityID) (*models.Identity, error)
	// ReadByOwnerResource reads the Identity for an owner resource (e.g. a Legal Entity)
	// Returns models.ErrNotFound if no Identity is associated with the specified resource.
	ReadByOwnerResource(ctx context.Context, txOrNil *Tx, ownerResourceID models.ResourceID) (*models.Identity, error)
	// FindOrCreateByOwnerResource creates an identity if no identity already exists for the specified owner resource,
	// otherwise it reads and returns the existing identity.
	// Returns the new or existing identity, and true iff a new identity was created.
	FindOrCreateByOwnerResource(ctx context.Context, txOrNil *Tx, ownerResourceID models.ResourceID) (identity *models.Identity, created bool, err error)
	// Delete permanently and idempotently deletes an Identity.
	Delete(ctx context.Context, txOrNil *Tx, id models.IdentityID) error
	// DeleteByOwnerResource permanently and idempotently deletes the Identity with the specified owner resource (if any).
	DeleteByOwnerResource(ctx context.Context, txOrNil *Tx, ownerResourceID models.ResourceID) error
}

type RepoStore interface {
	// Create a new repo.
	// Returns store.ErrAlreadyExists if a repo with matching unique properties already exists.
	Create(ctx context.Context, txOrNil *Tx, repo *models.Repo) error
	// Read an existing repo, looking it up by ID.
	// Returns models.ErrNotFound if the repo does not exist.
	Read(ctx context.Context, txOrNil *Tx, id models.RepoID) (*models.Repo, error)
	// ReadByExternalID reads an existing repo, looking it up by its external id.
	// Returns models.ErrNotFound if the repo does not exist.
	ReadByExternalID(ctx context.Context, txOrNil *Tx, externalID models.ExternalResourceID) (*models.Repo, error)
	// Update an existing repo with optimistic locking. Overrides all previous values using the supplied model.
	// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
	Update(ctx context.Context, txOrNil *Tx, repo *models.Repo) error
	// Upsert creates a repo if it does not exist, otherwise it updates its mutable properties
	// if they differ from the in-memory instance. Returns true,false if the resource was created
	// and false,true if the resource was updated. false,false if neither a create or update was necessary.
	// Repo Metadata and selected fields will not be updated (including Enabled and SSHKeySecretID fields).
	Upsert(ctx context.Context, txOrNil *Tx, model *models.Repo) (bool, bool, error)
	// SoftDelete soft deletes an existing repo.
	// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
	SoftDelete(ctx context.Context, txOrNil *Tx, repo *models.Repo) error
	// Search all repos. If searcher is set, the results will be limited to repos the searcher is authorized to
	// see (via the read:repo permission). Use cursor to page through results, if any.
	Search(ctx context.Context, txOrNil *Tx, searcher models.IdentityID, query search.Query) ([]*models.Repo, *models.Cursor, error)
	// IncrementBuildCounter increments and returns the build counter for the specified repo.
	IncrementBuildCounter(ctx context.Context, txOrNil *Tx, id models.RepoID) (models.BuildNumber, error)
	InitializeBuildCounter(ctx context.Context, txOrNil *Tx, id models.RepoID) error
}

type CommitStore interface {
	// Create a new commit.
	// Returns store.ErrAlreadyExists if a commit with matching unique properties already exists.
	Create(ctx context.Context, txOrNil *Tx, commit *models.Commit) error
	// Read an existing commit, looking it up by ID.
	// Returns models.ErrNotFound if the commit does not exist.
	Read(ctx context.Context, txOrNil *Tx, id models.CommitID) (*models.Commit, error)
	// ReadBySHA reads an existing commit, looking it up by its repo and SHA hash.
	// Returns models.ErrNotFound if the commit does not exist.
	ReadBySHA(ctx context.Context, txOrNil *Tx, repoID models.RepoID, sha string) (*models.Commit, error)
	// Update an existing commit with optimistic locking. Overrides all previous values using the supplied model.
	// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
	Update(ctx context.Context, txOrNil *Tx, commit *models.Commit) error
	// Upsert creates a Commit if it does not exist, otherwise it updates its mutable properties
	// if they differ from the in-memory instance. Returns true,false if the resource was created
	// and false,true if the resource was updated. false,false if neither a create nor update was necessary.
	Upsert(ctx context.Context, txOrNil *Tx, commit *models.Commit) (bool, bool, error)
	// LockRowForUpdate takes out an exclusive row lock on the commit table row for the specified commit.
	// This must be done within a transaction, and will block other transactions from locking, reading or updating
	// the row until this transaction ends.
	LockRowForUpdate(ctx context.Context, tx *Tx, id models.CommitID) error
}

type BuildStore interface {
	// Create a new build.
	// Returns store.ErrAlreadyExists if a build with matching unique properties already exists.
	Create(ctx context.Context, txOrNil *Tx, build *models.Build) error
	// Read an existing build, looking it up by ID.
	// Returns models.ErrNotFound if the build does not exist.
	Read(ctx context.Context, txOrNil *Tx, id models.BuildID) (*models.Build, error)
	// Update an existing build with optimistic locking. Overrides all previous values using the supplied model.
	// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
	Update(ctx context.Context, txOrNil *Tx, build *models.Build) error
	// LockRowForUpdate takes out an exclusive row lock on the build table row for the specified build.
	// This function must be called within a transaction, and will block other transactions from locking, updating
	// or deleting the row until this transaction ends.
	LockRowForUpdate(ctx context.Context, tx *Tx, id models.BuildID) error
	// Search all builds. If searcher is set, the results will be limited to builds the searcher is authorized to
	// see (via the read:build permission). Use cursor to page through results, if any.
	Search(ctx context.Context, txOrNil *Tx, searcher models.IdentityID, search *models.BuildSearch) ([]*models.BuildSearchResult, *models.Cursor, error)
	// UniversalSearch searches all builds. If searcher is set, the results will be limited to builds the searcher is authorized to
	// see (via the read:build permission). Use cursor to page through results, if any.
	UniversalSearch(ctx context.Context, txOrNil *Tx, searcher models.IdentityID, search search.Query) ([]*models.BuildSearchResult, *models.Cursor, error)
}

type JobStore interface {
	// Create a new job.
	// Returns store.ErrAlreadyExists if a job with matching unique properties already exists.
	Create(ctx context.Context, txOrNil *Tx, job *models.Job) error
	// Read an existing job, looking it up by ID.
	// Returns models.ErrNotFound if the job does not exist.
	Read(ctx context.Context, txOrNil *Tx, id models.JobID) (*models.Job, error)
	// ReadByName reads an existing job, looking it up by build, workflow and job name.
	// Returns models.ErrNotFound if the job is not found.
	ReadByName(ctx context.Context, txOrNil *Tx, buildID models.BuildID, workflow models.ResourceName, jobName models.ResourceName) (*models.Job, error)
	// ReadByFingerprint reads the most recent successful job inside a repo with a matching workflow, name
	// and fingerprint. Returns models.ErrNotFound if the job does not exist.
	ReadByFingerprint(
		ctx context.Context,
		txOrNil *Tx,
		repoID models.RepoID,
		workflow models.ResourceName,
		jobName models.ResourceName,
		jobFingerprint string,
		jobFingerprintHashType *models.HashType) (*models.Job, error)
	// Update an existing job with optimistic locking. Overrides all previous values using the supplied model.
	// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
	Update(ctx context.Context, txOrNil *Tx, job *models.Job) error
	// ListByBuildID gets all jobs that are associated with the specified build id.
	ListByBuildID(ctx context.Context, txOrNil *Tx, id models.BuildID) ([]*models.Job, error)
	// ListByStatus returns all jobs that have the specified status, regardless of who owns the jobs or which build
	// they are part of. Use cursor to page through results, if any.
	ListByStatus(ctx context.Context, txOrNil *Tx, status models.WorkflowStatus, pagination models.Pagination) ([]*models.Job, *models.Cursor, error)
	// ListDependencies lists all jobs that the specified job depends on.
	// Deferred dependencies (on jobs in other workflows that don't yet exist) will not be listed.
	ListDependencies(ctx context.Context, txOrNil *Tx, jobID models.JobID) ([]*models.Job, error)
	// CreateDependency records a dependency between jobs where source depends on target.
	CreateDependency(ctx context.Context, txOrNil *Tx, buildID models.BuildID, sourceJobID models.JobID, targetJobID models.JobID) error
	// CreateDeferredDependency records a dependency between a job and another job in another workflow
	// which does not yet exist.
	CreateDeferredDependency(ctx context.Context, txOrNil *Tx, buildID models.BuildID, sourceJobID models.JobID, targetWorkflow models.ResourceName, targetJobName models.ResourceName) error
	// UpdateDeferredDependencies updates any dependencies that refer to the target job's workflow and job name,
	// clearing those fields and setting target job ID instead. This has the effect of converting all dependencies
	// on the target job from deferred dependencies into 'real' dependencies.
	UpdateDeferredDependencies(ctx context.Context, txOrNil *Tx, targetJob *models.Job) error
	// CreateLabel records a label against a job.
	CreateLabel(ctx context.Context, txOrNil *Tx, jobID models.JobID, label models.Label) error
	// FindQueuedJob locates a queued job that the runner is capable of running, and which is ready for
	// execution (e.g all dependencies are completed).
	FindQueuedJob(ctx context.Context, txOrNil *Tx, runner *models.Runner) (*models.Job, error)
}

type StepStore interface {
	// Create a new step.
	// Returns store.ErrAlreadyExists if a step with matching unique properties already exists.
	Create(ctx context.Context, txOrNil *Tx, step *models.Step) error
	// Read an existing step, looking it up by ID.
	// Returns models.ErrNotFound if the step does not exist.
	Read(ctx context.Context, txOrNil *Tx, id models.StepID) (*models.Step, error)
	// Update an existing step with optimistic locking. Overrides all previous values using the supplied model.
	// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
	Update(ctx context.Context, txOrNil *Tx, step *models.Step) error
	// ListByJobID gets all steps that are associated with the specified job id.
	ListByJobID(ctx context.Context, txOrNil *Tx, id models.JobID) ([]*models.Step, error)
}

type SecretStore interface {
	// Create a new secret.
	// Returns store.ErrAlreadyExists if a secret with matching unique properties already exists.
	Create(ctx context.Context, txOrNil *Tx, secret *models.Secret) error
	// Read an existing secret, looking it up by ID.
	// Returns models.ErrNotFound if the secret does not exist.
	Read(ctx context.Context, txOrNil *Tx, id models.SecretID) (*models.Secret, error)
	// Update an existing secret with optimistic locking. Overrides all previous values using the supplied model.
	// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
	Update(ctx context.Context, txOrNil *Tx, secret *models.Secret) error
	// Delete permanently and idempotently deletes a secret, identifying it by id.
	Delete(ctx context.Context, txOrNil *Tx, id models.SecretID) error
	// ListByRepoID lists all secrets for a repo. Use cursor to page through results, if any.
	ListByRepoID(ctx context.Context, txOrNil *Tx, repoID models.RepoID, pagination models.Pagination) ([]*models.Secret, *models.Cursor, error)
}

type GroupStore interface {
	// Create a new access control Group.
	// Returns store.ErrAlreadyExists if a group with matching unique properties already exists.
	Create(ctx context.Context, txOrNil *Tx, group *models.Group) error
	// Read an existing access control Group, looking it up by ResourceID.
	// Returns models.ErrNotFound if the Group does not exist.
	Read(ctx context.Context, txOrNil *Tx, id models.GroupID) (*models.Group, error)
	// ReadByName reads an existing access control Group, looking it up by group name and the ID of the
	// legal entity that owns the group. Returns models.ErrNotFound if the group does not exist.
	ReadByName(ctx context.Context, txOrNil *Tx, ownerLegalEntityID models.LegalEntityID, groupName models.ResourceName) (*models.Group, error)
	// ReadByExternalID reads an existing group, looking it up by its unique external id.
	// Returns models.ErrNotFound if the group does not exist.
	ReadByExternalID(ctx context.Context, txOrNil *Tx, externalID models.ExternalResourceID) (*models.Group, error)
	// FindOrCreateByName finds and returns the access control Group with the name and legal entity specified in
	// the supplied group data.
	// If no such group exists then a new group is created and returned, and true is returned for 'created'.
	FindOrCreateByName(ctx context.Context, txOrNil *Tx, groupData *models.Group) (group *models.Group, created bool, err error)
	// Update an existing group with optimistic locking. Overrides all previous values using the supplied model.
	// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
	Update(ctx context.Context, txOrNil *Tx, group *models.Group) error
	// UpsertByExternalID creates a group if no group with the same External ID already exists, otherwise it updates
	// the existing group's mutable properties if they differ from the in-memory instance.
	// Returns true,false if the resource was created, false,true if the resource was updated, or false,false if
	// neither create nor update was necessary.
	// Returns an error if no External ID is filled out in the supplied Group.
	// In all cases group.ID will be filled out in the supplied group object.
	UpsertByExternalID(ctx context.Context, txOrNil *Tx, group *models.Group) (bool, bool, error)
	// Delete permanently and idempotently deletes an access control group.
	// The caller is responsible for ensuring that all memberships and grants for the group have previously been deleted.
	Delete(ctx context.Context, txOrNil *Tx, id models.GroupID) error
	// ListGroups returns a list of groups. Use cursor to page through results, if any.
	// If groupParent is provided then only groups owned by the supplied parent legal entity will be returned.
	// If memberID is provided then only groups that include the provided identity as a member will be returned.
	ListGroups(ctx context.Context, txOrNil *Tx, groupParent *models.LegalEntityID, memberID *models.IdentityID, pagination models.Pagination) ([]*models.Group, *models.Cursor, error)
}

type GroupMembershipStore interface {
	// Create a new access control group membership.
	// Returns store.ErrAlreadyExists if a group membership with matching unique properties already exists.
	Create(ctx context.Context, txOrNil *Tx, groupMembershipData *models.GroupMembershipData) (*models.GroupMembership, error)
	// Read an existing access control group membership, looking it up by ResourceID.
	// Returns models.ErrNotFound if the group membership does not exist.
	Read(ctx context.Context, txOrNil *Tx, id models.GroupMembershipID) (*models.GroupMembership, error)
	// ReadByMember reads an existing access control group membership, looking it up by group, member identity and
	// source system. Returns models.ErrNotFound if the group membership does not exist.
	ReadByMember(ctx context.Context, txOrNil *Tx, groupID models.GroupID, memberID models.IdentityID, sourceSystem models.SystemName) (*models.GroupMembership, error)
	// FindOrCreate finds and returns the access control group membership with the group, member identity and
	// source system specified in the supplied group membership data.
	// If no such group membership exists then a new one is created and returned, and true is returned for 'created'.
	FindOrCreate(ctx context.Context, txOrNil *Tx, membershipData *models.GroupMembershipData) (membership *models.GroupMembership, created bool, err error)
	// DeleteByMember removes a member identity from an access control group by deleting the relevant membership record(s).
	// If sourceSystem is not nil then only the record matching the source system will be deleted; otherwise
	// records from all source systems for the member will be deleted.
	// This method is idempotent.
	DeleteByMember(ctx context.Context, txOrNil *Tx, groupID models.GroupID, memberID models.IdentityID, sourceSystem *models.SystemName) error
	// DeleteAllMembersOfGroup removes all members from an access control group by deleting all membership records for that
	// group. This method is idempotent.
	DeleteAllMembersOfGroup(ctx context.Context, txOrNil *Tx, groupID models.GroupID) error
	// ListGroupMemberships returns a list of group memberships. Use cursor to page through results, if any.
	// If groupID is provided then only memberships of the specified group will be returned.
	// If memberID is provided then only groups that include the provided identity as a member will be returned.
	// If sourceSystem is provided then only memberships with matching source system values will be returned.
	ListGroupMemberships(
		ctx context.Context,
		txOrNil *Tx,
		groupID *models.GroupID,
		memberID *models.IdentityID,
		sourceSystem *models.SystemName,
		pagination models.Pagination,
	) ([]*models.GroupMembership, *models.Cursor, error)
}

type GrantStore interface {
	// Create a new grant.
	// Returns store.ErrAlreadyExists if a grant with matching unique properties already exists.
	Create(ctx context.Context, txOrNil *Tx, model *models.Grant) error
	// Read an existing grant, looking it up by ID.
	// Returns models.ErrNotFound if the grant does not exist.
	Read(ctx context.Context, txOrNil *Tx, id models.GrantID) (*models.Grant, error)
	// ReadByAuthorizedOperation reads an existing grant, looking it up by requiring the following fields to match:
	// - OperationResourceType
	// - OperationName
	// - TargetResourceID
	// - either AuthorizedIdentityID or AuthorizedGroupID must match, whichever one is not nil
	// Returns models.ErrNotFound if the grant does not exist.
	ReadByAuthorizedOperation(ctx context.Context, txOrNil *Tx, model *models.Grant) (*models.Grant, error)
	// Update an existing grant with optimistic locking. Overrides all previous values using the supplied model.
	// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
	Update(ctx context.Context, txOrNil *Tx, model *models.Grant) error
	// Delete permanently and idempotently deletes a grant, identifying it by id.
	Delete(ctx context.Context, txOrNil *Tx, id models.GrantID) error
	// FindOrCreate finds and returns a grant with the data specified in the supplied grant data.
	// The readByAuthorizedOperation function is used to find matching grants.
	// If no such grant exists then a new one is created and returned, and true is returned for 'created'.
	FindOrCreate(ctx context.Context, txOrNil *Tx, grantData *models.Grant) (grant *models.Grant, created bool, err error)
	// ListGrantsForGroup finds and returns all grants that give permissions to the specified group.
	ListGrantsForGroup(ctx context.Context, txOrNil *Tx, groupID models.GroupID, pagination models.Pagination) ([]*models.Grant, *models.Cursor, error)
	// DeleteAllGrantsForGroup permanently and idempotently deletes all grants for the specified group.
	DeleteAllGrantsForGroup(ctx context.Context, txOrNil *Tx, groupID models.GroupID) error
	// DeleteAllGrantsForIdentity permanently and idempotently deletes all grants for the specified identity.
	DeleteAllGrantsForIdentity(ctx context.Context, txOrNil *Tx, identityID models.IdentityID) error
}

type OwnershipStore interface {
	// Create a new ownership.
	// Returns store.ErrAlreadyExists if an ownership with matching unique properties already exists.
	Create(ctx context.Context, txOrNil *Tx, ownership *models.Ownership) error
	// Read an existing ownership, looking it up by ID.
	// Returns models.ErrNotFound if the ownership does not exist.
	Read(ctx context.Context, txOrNil *Tx, id models.OwnershipID) (*models.Ownership, error)
	// Update an existing ownership with optimistic locking. Overrides all previous values using the supplied model.
	// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
	Update(ctx context.Context, txOrNil *Tx, ownership *models.Ownership) error
	// Upsert creates an ownership if it does not exist, otherwise it updates its mutable properties
	// if they differ from the in-memory instance. Returns true,false if the resource was created
	// and false,true if the resource was updated. false,false if neither a create or update was necessary.
	Upsert(ctx context.Context, txOrNil *Tx, ownership *models.Ownership) (bool, bool, error)
	// Delete permanently and idempotently deletes an ownership, identifying it by owned resource id.
	Delete(ctx context.Context, txOrNil *Tx, ownedResourceID models.ResourceID) error
}

type AuthorizationStore interface {
	// CountGrantsForOperation counts the number of grants that an identity has for the specified operation
	// against the specified resource. All pathways are explored to locate the grants, including direct,
	// group membership and inheritance.
	CountGrantsForOperation(ctx context.Context, txOrNil *Tx, identityID models.IdentityID, operation *models.Operation, resourceID models.ResourceID) (int, error)
}

type CredentialStore interface {
	// Create a new credential.
	// Returns store.ErrAlreadyExists if a credential with matching unique properties already exists.
	Create(ctx context.Context, txOrNil *Tx, credential *models.Credential) error
	// Read an existing credential, looking it up by ID.
	// Returns models.ErrNotFound if the credential does not exist.
	Read(ctx context.Context, txOrNil *Tx, id models.CredentialID) (*models.Credential, error)
	// ReadBySharedSecretID reads an existing shared secret credential, looking it up by shared secret ID.
	// Returns models.ErrNotFound if the credential does not exist.
	ReadBySharedSecretID(ctx context.Context, txOrNil *Tx, sharedSecretID string) (*models.Credential, error)
	// ReadByGitHubUserID reads an existing GitHub credential, looking it up by the GitHub user ID.
	// Returns models.ErrNotFound if the credential does not exist.
	ReadByGitHubUserID(ctx context.Context, txOrNil *Tx, gitHubUserID int64) (*models.Credential, error)
	// ReadByPublicKey reads an existing client certificate credential, looking it up by from the supplied public key.
	// Returns models.ErrNotFound if the credential does not exist.
	ReadByPublicKey(ctx context.Context, txOrNil *Tx, publicKey certificates.PublicKeyData) (*models.Credential, error)
	// Delete permanently and idempotently deletes a credential.
	Delete(ctx context.Context, txOrNil *Tx, id models.CredentialID) error
	// ListCredentialsForIdentity returns a list of all credentials for the specified identity ID.
	// Use cursor to page through results, if any.
	ListCredentialsForIdentity(
		ctx context.Context,
		txOrNil *Tx,
		identityID models.IdentityID,
		pagination models.Pagination,
	) ([]*models.Credential, *models.Cursor, error)
}

type ArtifactStore interface {
	// Create a new artifact.
	// Returns store.ErrAlreadyExists if an artifact with matching unique properties already exists.
	Create(ctx context.Context, txOrNil *Tx, artifact *models.ArtifactData) (*models.Artifact, error)
	// FindOrCreate creates an artifact if no artifact with the same unique values exist,
	// otherwise it reads and returns the existing artifact.
	// Returns the artifact as it is in the database, and true iff a new artifact was created.
	FindOrCreate(ctx context.Context, txOrNil *Tx, artifact *models.ArtifactData) (*models.Artifact, bool, error)
	// Read an existing artifact, looking it up by ID.
	// Returns models.ErrNotFound if the artifact does not exist.
	Read(ctx context.Context, txOrNil *Tx, id models.ArtifactID) (*models.Artifact, error)
	// FindByUniqueFields returns a matching artifact from the fields that are unique within our store.
	FindByUniqueFields(ctx context.Context, txOrNil *Tx, artifact *models.ArtifactData) (*models.Artifact, error)
	// Update an existing artifact with optimistic locking. Overrides all previous values using the supplied model.
	// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
	Update(ctx context.Context, txOrNil *Tx, artifact *models.Artifact) error
	// Search all artifacts. If searcher is set, the results will be limited to artifacts the searcher is authorized to
	// see (via the read:artifact permission). Use cursor to page through results, if any.
	Search(ctx context.Context, txOrNil *Tx, searcher models.IdentityID, search models.ArtifactSearch) ([]*models.Artifact, *models.Cursor, error)
}

type RunnerStore interface {
	// Create a new runner.
	// Returns store.ErrAlreadyExists if a runner with matching unique properties already exists.
	Create(ctx context.Context, txOrNil *Tx, runner *models.Runner) error
	// Read an existing runner, looking it up by ID.
	// Returns models.ErrNotFound if the runner does not exist.
	Read(ctx context.Context, txOrNil *Tx, id models.RunnerID) (*models.Runner, error)
	// ReadByName reads an existing runner, looking it up by name and the ID of the legal entity that owns the runner.
	// Returns models.ErrNotFound if the runner is not found.
	ReadByName(ctx context.Context, txOrNil *Tx, legalEntityID models.LegalEntityID, name models.ResourceName) (*models.Runner, error)
	// Update an existing runner.
	// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
	Update(ctx context.Context, txOrNil *Tx, runner *models.Runner) error
	// LockRowForUpdate takes out an exclusive row lock on the runner table row for the specified runner.
	// This must be done within a transaction, and will block other transactions from locking or updating
	// the row until this transaction ends.
	LockRowForUpdate(ctx context.Context, tx *Tx, id models.RunnerID) error
	// SoftDelete soft deletes an existing runner.
	// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
	SoftDelete(ctx context.Context, txOrNil *Tx, runner *models.Runner) error
	// CreateLabel records a label against a runner.
	CreateLabel(ctx context.Context, txOrNil *Tx, runnerID models.RunnerID, label models.Label) error
	// DeleteLabel deletes an existing label from a runner.
	DeleteLabel(ctx context.Context, txOrNil *Tx, runnerID models.RunnerID, label models.Label) error
	// CreateSupportedJobType records a supported job type against a runner.
	CreateSupportedJobType(ctx context.Context, txOrNil *Tx, runnerID models.RunnerID, kind models.JobType) error
	// DeleteSupportedJobType deletes an existing supported job type from a runner.
	DeleteSupportedJobType(ctx context.Context, txOrNil *Tx, runnerID models.RunnerID, kind models.JobType) error
	// RunnerCompatibleWithJob returns true if a runner exists that is capable of running job.
	RunnerCompatibleWithJob(ctx context.Context, txOrNil *Tx, job *models.Job) (bool, error)
	// Search all runners. If searcher is set, the results will be limited to runners the searcher is authorized to
	// see (via the read:runner permission). Use cursor to page through results, if any.
	Search(ctx context.Context, txOrNil *Tx, searcher models.IdentityID, search models.RunnerSearch) ([]*models.Runner, *models.Cursor, error)
}

type ResourceLinkStore interface {
	// Upsert creates a resource link fragment if it does not exist, otherwise it updates its mutable properties
	// if they differ from the in-memory instance. Returns true,false if the resource was created
	// and false,true if the resource was updated. false,false if neither a create or update was necessary.
	Upsert(ctx context.Context, txOrNil *Tx, namedResource models.NamedResource) (bool, bool, error)
	// Resolve the leaf resource fragment in a resource link.
	// Returns models.ErrNotFound if the fragment does not exist.
	Resolve(ctx context.Context, txOrNil *Tx, link models.ResourceLink) (*models.ResourceLinkFragment, error)
	// Delete permanently and idempotently deletes a resource link fragment for a resource, ensuring its name can now be reused.
	Delete(ctx context.Context, txOrNil *Tx, resourceID models.ResourceID) error
}

type LogStore interface {
	// Create a new logs.
	// Returns store.ErrAlreadyExists if a logs with matching unique properties already exists.
	Create(ctx context.Context, txOrNil *Tx, container *models.LogDescriptor) error
	// Read an existing logs, looking it up by ID.
	// Returns models.ErrNotFound if the logs does not exist.
	Read(ctx context.Context, txOrNil *Tx, id models.LogDescriptorID) (*models.LogDescriptor, error)
	// Update an existing logs.
	// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
	Update(ctx context.Context, txOrNil *Tx, container *models.LogDescriptor) error
	// Delete permanently and idempotently deletes a log.
	Delete(ctx context.Context, txOrNil *Tx, id models.LogDescriptorID) error
	// Search all log descriptors. If searcher is set, the results will be limited to log descriptors the searcher
	// is authorized to see (via the read:build permission). Use cursor to page through results, if any.
	Search(ctx context.Context, txOrNil *Tx, searcher models.IdentityID, search models.LogDescriptorSearch) ([]*models.LogDescriptor, *models.Cursor, error)
}

type PullRequestStore interface {
	// Create a new Pull Request.
	// Returns store.ErrAlreadyExists if a Pull Request with matching unique properties already exists.
	Create(ctx context.Context, txOrNil *Tx, pullRequest *models.PullRequest) error
	// Read an existing Pull Request, looking it up by ID.
	// Returns models.ErrNotFound if the Pull Request does not exist.
	Read(ctx context.Context, txOrNil *Tx, id models.PullRequestID) (*models.PullRequest, error)
	// Update an existing pull request with optimistic locking. Overrides all previous values using the supplied model.
	// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
	Update(ctx context.Context, txOrNil *Tx, pullRequest *models.PullRequest) error
	// Upsert creates a pull request for a given External ID if it does not exist, otherwise it updates its
	// mutable properties if they differ from the in-memory instance. Returns true,false if the resource was
	// created and false,true if the resource was updated. false,false if neither a create nor update was necessary.
	Upsert(ctx context.Context, txOrNil *Tx, pullRequest *models.PullRequest) (bool, bool, error)
}

type WorkItemStore interface {
	// Create a new work item.
	// Returns store.ErrAlreadyExists if a work item with this ID already exists.
	Create(ctx context.Context, txOrNil *Tx, workItem *models.WorkItem) error
	// Read an existing work item, looking it up by ResourceID.
	// Will return models.ErrNotFound if the work item does not exist.
	Read(ctx context.Context, txOrNil *Tx, id models.WorkItemID) (*models.WorkItem, error)
	// Update an existing work item with optimistic locking. Overrides all previous values using the supplied model.
	// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
	Update(ctx context.Context, txOrNil *Tx, workItem *models.WorkItem) error
	// Delete permanently and idempotently deletes a work item.
	Delete(ctx context.Context, txOrNil *Tx, id models.WorkItemID) error
}

type WorkItemStateStore interface {
	// FindOrCreateAndLockRow create a new work item state record if one does not already exist with the same
	// concurrency key, otherwise reads and returns the existing record.
	//
	// A row lock is taken out on the returned record for the duration of the supplied transaction.
	FindOrCreateAndLockRow(ctx context.Context, tx *Tx, state *models.WorkItemState) (*models.WorkItemState, error)
	// Read an existing work item state record, looking it up by ResourceID.
	// Will return models.ErrNotFound if the work item does not exist.
	Read(ctx context.Context, txOrNil *Tx, id models.WorkItemStateID) (*models.WorkItemState, error)
	// Update an existing work item state record with optimistic locking. Overrides all previous values using
	// the supplied model. Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
	Update(ctx context.Context, txOrNil *Tx, state *models.WorkItemState) error
	// LockRowForUpdate takes out an exclusive row lock on the database row for the specified work item state.
	// This must be done within a transaction, and will block other transactions from locking, reading or updating
	// the row until this transaction ends.
	LockRowForUpdate(ctx context.Context, tx *Tx, id models.WorkItemStateID) error
	// Delete permanently and idempotently deletes a work item state record.
	Delete(ctx context.Context, txOrNil *Tx, id models.WorkItemStateID) error
	// CountWorkItems returns the number of work items associated with the specified work item state record.
	// This will include any completed or failed work items which have not been deleted.
	CountWorkItems(ctx context.Context, txOrNil *Tx, workItemStateID models.WorkItemStateID) (int, error)
	// FindQueuedWorkItem reads the next queued work item that is ready to be allocated to a work item processor.
	// A row lock is taken out on the work item state row for the returned work item, for the duration of the
	// supplied transaction.
	//
	// A work item is logically a combination of a WorkItemRecord and a WorkItemState object, and both objects
	// are returned. The WorkItemState row in the table is locked, preventing any other caller from allocating
	// a work item with the same concurrency key (which would share the same WorkItemState row).
	//
	// The now parameter is the current time, for comparison with time values in the database like 'allocated until'.
	//
	// Only work items of the types in the supplied list will be returned.
	// Will return gerror.ErrNotFound if no suitable work item can be found.
	FindQueuedWorkItem(ctx context.Context, tx *Tx, now models.Time, types []models.WorkItemType) (*models.WorkItemRecords, error)
}

type EventStore interface {
	// Create a new event.
	// Returns store.ErrAlreadyExists if an event with this ID or build/sequence number already exists.
	Create(ctx context.Context, txOrNil *Tx, sequenceNumber models.EventNumber, eventData *models.EventData) (*models.Event, error)
	// Read an existing event, looking it up by ResourceID.
	// Will return models.ErrNotFound if the event does not exist.
	Read(ctx context.Context, txOrNil *Tx, id models.EventID) (*models.Event, error)
	// DeleteEventsForBuild permanently and idempotently deletes all events for the specified build.
	DeleteEventsForBuild(ctx context.Context, txOrNil *Tx, buildID models.BuildID) error
	// FindEvents reads the next events for a build.
	// If no matching events are present then an empty list is returned immediately.
	FindEvents(ctx context.Context, txOrNil *Tx, buildID models.BuildID, lastEventNumber models.EventNumber, limit int) ([]*models.Event, error)
	// IncrementEventCounter increments and returns the event counter for the specified build, to provide
	// a sequence number for a new event.
	IncrementEventCounter(ctx context.Context, txOrNil *Tx, buildID models.BuildID) (models.EventNumber, error)
}
