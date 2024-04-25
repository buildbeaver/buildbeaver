package scm

import (
	"context"
	"net/http"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

// OrgToGroupsMap is a map specifying a set of access control Groups in various organizations, sourced from an SCM.
// An organization is specified as the organization's Legal Entity ExternalResourceID from the SCM.
// A group is specified as the ResourceName for the standard group that most closely corresponds
// to the group or role in the SCM.
type OrgToGroupsMap map[models.ExternalResourceID][]models.ResourceName

type SCM interface {
	// Name returns the unique name of the SCM.
	Name() models.SystemName
	// WebhookHandler returns the http handler func that should be invoked when
	// the SCM service receives a webhook, or an error if the service does not
	// support webhooks.
	WebhookHandler() (http.HandlerFunc, error)
	// EnableRepo is called when a repo is enabled within the system - this is the SCM's opportunity to
	// to do any setup required to close the loop and make this work. Public key identifies the key that
	// the system will use when cloning the repo.
	EnableRepo(ctx context.Context, repo *models.Repo, publicKey []byte) error
	// DisableRepo is called when a repo is disabled in the system - this is the SCM's opportunity to do any required
	// teardown such as deleting webhooks or deployment keys etc.
	DisableRepo(ctx context.Context, repo *models.Repo) error
	// BuildRepoLatestCommit will kick off a new build for the latest commit for a ref, if required.
	// The ref can be a branch or a tag. The supplied ref is read from GitHub to determine the latest commit.
	// If no ref is supplied then the head of the main/master branch for the repo will be used.
	// If there is no build underway or complete for the latest commit then a new build will be queued.
	// If all completed builds for this commit failed then a new build will be queued.
	// Older builds for previous commits for this ref may be cancelled or elided from the queue, since they
	// are out of date.
	BuildRepoLatestCommit(ctx context.Context, repo *models.Repo, ref string) error
	// NotifyBuildUpdated is called when the status of a build is updated.
	// Allows the SCM to notify users or take other actions when a build has progressed or finished.
	NotifyBuildUpdated(ctx context.Context, txOrNil *store.Tx, build *models.Build, repo *models.Repo) error
	// GetUserLegalEntityData returns legal entity data representing the user currently authenticated with auth.
	GetUserLegalEntityData(ctx context.Context, auth models.SCMAuth) (*models.LegalEntityData, error)
	// IsLegalEntityRegisteredAsUser returns true if the specified Legal Entity is registered as a user of this
	// build system on this SCM. The meaning of 'registered as a user' is dependent on the SCM.
	IsLegalEntityRegisteredAsUser(ctx context.Context, legalEntity *models.LegalEntity) (bool, error)
	// ListLegalEntitiesRegisteredAsUsers lists all Legal Entities from the SCM that are registered as using this
	// build system for any of their repos.
	ListLegalEntitiesRegisteredAsUsers(ctx context.Context) ([]*models.LegalEntityData, error)
	// ListReposRegisteredForLegalEntity lists all repos belonging to a legal entity that are registered as using
	// the build system.
	ListReposRegisteredForLegalEntity(ctx context.Context, legalEntity *models.LegalEntity) ([]*models.Repo, error)
	// ListAllCompanyMembers returns a list of all users who are members of the specified company.
	ListAllCompanyMembers(ctx context.Context, company *models.LegalEntity) ([]*models.LegalEntityData, error)
	// ListCompanyCustomGroups returns a list of custom groups that can be used for access control for a company.
	// These can corresponding to teams, roles, or any other way to grant access to a group of people.
	// These groups will be created in addition to the standard groups that are created for each company.
	ListCompanyCustomGroups(ctx context.Context, company *models.LegalEntity) ([]*models.Group, error)
	// ListCompanyGroupMembers returns a list of users who are members of the specified group within the specified
	// company. The group is either a standard or a custom group that corresponds to the group of users
	// (e.g a role or team) in the SCM.
	ListCompanyGroupMembers(ctx context.Context, company *models.LegalEntity, group *models.Group) ([]*models.LegalEntityData, error)
	// ListCompanyCustomGroupPermissions returns a list of permissions that a custom group for a company on the SCM
	// should have. The group must be a custom group that corresponds to a group of users (e.g. team) in the SCM.
	ListCompanyCustomGroupPermissions(ctx context.Context, company *models.LegalEntity, group *models.Group) ([]*models.Grant, error)
}
