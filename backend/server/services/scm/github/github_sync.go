package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v28/github"
	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/models"
)

// GetUserLegalEntityData returns an SCM legal entity representing the user currently authenticated with auth.
// The returned legal entity will include external metadata for GitHub that includes the installation ID for the
// BuildBeaver GitHub app, if it is installed for this user.
func (s *GitHubService) GetUserLegalEntityData(ctx context.Context, auth models.SCMAuth) (*models.LegalEntityData, error) {
	// Read the currently authenticated user using the OAuth token (or can be a GitHub Personal Access Token)
	oAuthClient, err := s.makeGitHubOAuthClient(ctx, auth)
	if err != nil {
		return nil, fmt.Errorf("error making github client: %w", err)
	}
	user, _, err := oAuthClient.Users.Get(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("error getting user: %w", err)
	}

	// See if there is a GitHub BuildBeaver app installation ID for the user
	appClient, err := s.makeGitHubAppClient()
	if err != nil {
		return nil, fmt.Errorf("error making github app client: %w", err)
	}
	installation, res, err := appClient.Apps.FindUserInstallation(ctx, user.GetLogin())
	if err != nil {
		if res != nil && (res.StatusCode == http.StatusNotFound) {
			// No installation of the BuildBeaver GitHub app for this user; this is not an error
			installation = nil
			err = nil
		} else {
			return nil, fmt.Errorf("error checking current user for GitHub BuildBeaver app installation: %w", err)
		}
	}
	installationID := repoInstallationUnset
	if installation != nil {
		installationID = installation.GetID()
	}

	return s.legalEntityDataFromGitHubUser(user, installationID)
}

// IsLegalEntityRegisteredAsUser returns true if the specified Legal Entity has the GitHub app installed for
// themselves or any of their repos.
func (s *GitHubService) IsLegalEntityRegisteredAsUser(ctx context.Context, legalEntity *models.LegalEntity) (bool, error) {
	var installed = false

	metadata, err := GetLegalEntityMetadata(legalEntity)
	if err != nil {
		return false, err
	}
	if metadata.InstallationID != legalEntityInstallationUnset && metadata.InstallationID != 0 {
		installed = true
	}

	return installed, nil
}

// ListLegalEntitiesRegisteredAsUsers lists all Legal Entities (GitHub Users and Orgs) that have the
// GitHub app installed for themselves or any of their repos.
func (s *GitHubService) ListLegalEntitiesRegisteredAsUsers(ctx context.Context) ([]*models.LegalEntityData, error) {
	client, err := s.makeGitHubAppClient()
	if err != nil {
		return nil, fmt.Errorf("error creating GitHub app client for repo: %w", err)
	}

	installations, err := ListAllAppInstallations(ctx, client)
	if err != nil {
		return nil, err
	}

	// TODO: Make this a map and convert to list at the end, and warn if we get the same external ID twice
	var legalEntities []*models.LegalEntityData

	for i, installation := range installations {
		account := installation.GetAccount()

		legalEntity, err := s.readAccountDetailsForInstallation(
			ctx,
			account.GetID(),
			account.GetLogin(),
			account.GetType(),
			installation.GetID(),
		)
		if err != nil {
			s.Errorf("Will ignore error reading account details for installation: %s", err)
			continue
		}

		legalEntities = append(legalEntities, legalEntity)
		s.Infof("Installation %d: type %s, accountID %d, login %s, name %s", i+1,
			account.GetType(), account.GetID(), account.GetLogin(), legalEntity.Name)
	}

	return legalEntities, nil
}

// readAccountDetailsForInstallation reads information about the account (user or organization) that has
// a BuildBeaver GitHub app installation, and returns legal entity data that can be used to create or update
// the Legal Entity for the account. The external metadata for the returned legal entity will be filled out,
// including the InstallationID, and the OrgDefaultRepoPermission field if the account is an organization.
// The data will be read from GitHub using a client created from the supplied installation ID.
func (s *GitHubService) readAccountDetailsForInstallation(
	ctx context.Context,
	ghAccountID int64,
	ghLogin string,
	accountType string,
	installationID int64,
) (*models.LegalEntityData, error) {
	installationClient, err := s.makeGitHubAppInstallationClient(installationID)
	if err != nil {
		return nil, fmt.Errorf("error creating installation Client: %v", err.Error())
	}

	var legalEntityData *models.LegalEntityData
	switch accountType {
	case "Organization":
		org, _, err := installationClient.Organizations.GetByID(ctx, ghAccountID)
		if err != nil {
			return nil, fmt.Errorf("error reading GitHub org with ID %d: %v", ghAccountID, err)
		}
		if org.GetLogin() != ghLogin {
			return nil, fmt.Errorf("error with mismatched Login for GitHub org with ID %d: %v", ghAccountID, err)
		}
		legalEntityData, err = s.legalEntityDataFromGitHubOrg(org, installationID)
		if err != nil {
			return nil, fmt.Errorf("error making legal entity from GitHub org: %v", err)
		}

	case "User":
		user, _, err := installationClient.Users.GetByID(ctx, ghAccountID)
		if err != nil {
			return nil, fmt.Errorf("error reading GitHub user with ID %d: %v", ghAccountID, err)
		}
		if user.GetLogin() != ghLogin {
			return nil, fmt.Errorf("error with mismatched Login for GitHub user with ID %d: %v", ghAccountID, err)
		}
		legalEntityData, err = s.legalEntityDataFromGitHubUser(user, installationID)
		if err != nil {
			return nil, fmt.Errorf("error making legal entity from GitHub user: %v", err)
		}
	default:
		return nil, fmt.Errorf("error: unknown type of GitHub account '%s'", accountType)
	}

	return legalEntityData, nil
}

// ListReposRegisteredForLegalEntity lists all repos belonging to a legal entity that have the GitHub app installed.
func (s *GitHubService) ListReposRegisteredForLegalEntity(ctx context.Context, legalEntity *models.LegalEntity) ([]*models.Repo, error) {
	client, err := s.makeGitHubAppInstallationClientForLegalEntity(legalEntity)
	if err != nil {
		return nil, fmt.Errorf("error making GitHub installation client: %w", err)
	}

	ghRepos, err := ListAllReposForInstallation(ctx, client)
	if err != nil {
		return nil, err
	}

	var repos []*models.Repo
	for _, ghRepo := range ghRepos {
		s.Tracef("Processing repo %s/%s", ghRepo.GetOwner().GetLogin(), ghRepo.GetName())
		repo, err := s.repoDataFromGitHubRepo(ghRepo, legalEntity)
		if err != nil {
			s.Errorf("Ignoring error converting GitHub Repo to BuildBeaver repo: %v", err)
			continue
		}
		repos = append(repos, repo)
	}
	return repos, nil
}

// ListAllCompanyMembers returns a list of all users who are members of the specified company.
func (s *GitHubService) ListAllCompanyMembers(ctx context.Context, company *models.LegalEntity) ([]*models.LegalEntityData, error) {
	legalEntityMetadata, err := GetLegalEntityMetadata(company)
	if err != nil {
		return nil, err
	}
	client, err := s.makeGitHubAppInstallationClientForLegalEntity(company)
	if err != nil {
		return nil, fmt.Errorf("error making GitHub installation client: %w", err)
	}

	ghUsers, err := ListCompanyMembers(ctx, client, legalEntityMetadata.Login, "all") // special value
	if err != nil {
		return nil, fmt.Errorf("error listing members of GitHub Org %s: %w", company.Name, err)
	}

	var members []*models.LegalEntityData
	for _, user := range ghUsers {
		userLegalEntity, err := s.legalEntityDataFromGitHubUser(user, legalEntityInstallationUnset)
		if err != nil {
			s.Errorf("Will ignore error making legal entity from GitHub user: %v", err)
			continue
		}
		members = append(members, userLegalEntity)
	}

	return members, nil
}

// ListCompanyCustomGroups returns a list of custom groups that can be used for access control for a company.
// These can corresponding to teams, roles, or any other way to grant access to a group of people.
// These groups will be created in addition to the standard groups that are created for each company.
// For GitHub, we return a group for each team within the company.
func (s *GitHubService) ListCompanyCustomGroups(ctx context.Context, company *models.LegalEntity) ([]*models.Group, error) {
	legalEntityMetadata, err := GetLegalEntityMetadata(company)
	if err != nil {
		return nil, err
	}
	client, err := s.makeGitHubAppInstallationClientForLegalEntity(company)
	if err != nil {
		return nil, fmt.Errorf("error making GitHub installation client: %w", err)
	}

	ghTeams, err := ListAllTeamsForOrganization(ctx, client, legalEntityMetadata.Login)
	if err != nil {
		return nil, fmt.Errorf("error listing teams in GitHub Org %s: %w", company.Name, err)
	}

	var groups []*models.Group
	for _, ghTeam := range ghTeams {
		s.Tracef("Processing team %s/%s", ghTeam.GetOrganization().GetLogin(), ghTeam.GetName())
		group, err := s.groupDataFromGitHubTeam(ghTeam, company)
		if err != nil {
			s.Errorf("Ignoring error converting GitHub Team to BuildBeaver group: %v", err)
			continue
		}
		groups = append(groups, group)
	}

	return groups, nil
}

// ListCompanyGroupMembers returns a list of users who are members of the specified group within the specified
// company. The group is either a standard or a custom group that corresponds to the group of users
// (e.g a role or team) in the SCM.
func (s *GitHubService) ListCompanyGroupMembers(ctx context.Context, company *models.LegalEntity, group *models.Group) ([]*models.LegalEntityData, error) {
	legalEntityMetadata, err := GetLegalEntityMetadata(company)
	if err != nil {
		return nil, err
	}
	client, err := s.makeGitHubAppInstallationClientForLegalEntity(company)
	if err != nil {
		return nil, fmt.Errorf("error making GitHub installation client: %w", err)
	}

	// Read the members of the group from GitHub, via an API call appropriate to the type of group
	var ghUsers []*github.User
	if models.IsStandardGroupName(group.Name) {
		// Convert the standard group name we are interested in to a GitHub role to search on GitHub for members
		ghRole := ""
		switch group.Name {
		case models.AdminStandardGroup.Name:
			ghRole = "admin"
		case models.ReadOnlyUserStandardGroup.Name, models.UserStandardGroup.Name:
			// Users in the 'member' role will be in a specific group depending on the GitHub settings
			memberRoleGroupName := findGroupForMemberRole(legalEntityMetadata.OrgDefaultRepoPermission)
			if memberRoleGroupName != group.Name {
				// The group we are interested in is not the group assigned to the 'member' role, so
				// there are no users in the group we are interested in
				s.Infof("ListCompanyGroupMembers for group '%s' does not match legal entity '%s' default role permission - returning empty member list", group.Name, company.Name)
				return []*models.LegalEntityData{}, nil
			}
			ghRole = "member" // users with role 'member' on GitHub should are members of this group
		default:
			s.Warnf("ListCompanyGroupMembers called for an unknown standard group '%s' - returning empty member list", group.Name)
			return []*models.LegalEntityData{}, nil
		}
		s.Infof("ListCompanyGroupMembers looking for members with GitHub role '%s' which corresponds to standard group '%s' for Legal Entity '%s'", ghRole, group.Name, company.Name)
		ghUsers, err = ListCompanyMembers(ctx, client, legalEntityMetadata.Login, ghRole)
		if err != nil {
			return nil, fmt.Errorf("error listing members of GitHub Org %s role %s: %w", company.Name, ghRole, err)
		}
	} else if isGroupNameForTeam(group.Name) && group.ExternalID != nil && group.ExternalID.ExternalSystem == GitHubSCMName {
		// The group corresponds to a team, and the group external ID is the GitHub team ID
		teamID, err := githubIDFromExternalID(group.ExternalID.ResourceID)
		if err != nil {
			return nil, fmt.Errorf("error parsing external id to GitHub team id: %w", err)
		}
		// Ignore the distinction between members and admins for the team; both are just members of the BuildBeaver group
		ghUsers, err = ListTeamMembers(ctx, client, teamID, "all")
		if err != nil {
			return nil, fmt.Errorf("error listing members of GitHub Org %s team ID %d (group %s): %w", company.Name, teamID, group.Name, err)
		}
	} else {
		s.Warnf("ListCompanyGroupMembers called for an unknown group '%s' - returning empty member list", group.Name)
		return []*models.LegalEntityData{}, nil
	}

	var members []*models.LegalEntityData
	for _, user := range ghUsers {
		userLegalEntity, err := s.legalEntityDataFromGitHubUser(user, legalEntityInstallationUnset)
		if err != nil {
			s.Errorf("Will ignore error making legal entity from GitHub user: %v", err)
			continue
		}
		members = append(members, userLegalEntity)
	}

	return members, nil
}

// findGroupForMemberRole finds the standard group to put ordinary members of a GitHub organization into, based on
// the GitHub 'default repo permission' setting for the organization.
func findGroupForMemberRole(orgDefaultRepoPermission string) models.ResourceName {
	switch orgDefaultRepoPermission {
	case "read":
		return models.ReadOnlyUserStandardGroup.Name
	case "write":
		return models.UserStandardGroup.Name
	case "admin":
		return models.UserStandardGroup.Name // no special group for repo admins
	case "none":
		// don't return BaseStandardGroup since all company members are members of this group
		// automatically, so we don't want to sync this group
		return ""
	default:
		return ""
	}
}

// findGroupForMemberRole finds the standard group to put members of a GitHub organization into, based on
// the GitHub role assigned to the member (i.e. whether they are an admin) and the GitHub
// 'default repo permission' setting for the organization which applies for non-admin members.
func findGroupForGitHubPermissions(orgRoleStr string, orgDefaultRepoPermission string) models.ResourceName {
	switch orgRoleStr {
	case "admin":
		return models.AdminStandardGroup.Name
	case "member":
		// For ordinary members, the standard group is based on orgDefaultRepoPermission
		return findGroupForMemberRole(orgDefaultRepoPermission)
	default:
		return ""
	}
}

// ListCompanyCustomGroupPermissions returns a list of permissions that a custom group for a company on the SCM
// should have. The group must be a custom group that corresponds to a group of users (e.g. team) in the SCM.
func (s *GitHubService) ListCompanyCustomGroupPermissions(ctx context.Context, company *models.LegalEntity, group *models.Group) ([]*models.Grant, error) {
	client, err := s.makeGitHubAppInstallationClientForLegalEntity(company)
	if err != nil {
		return nil, fmt.Errorf("error making GitHub installation client: %w", err)
	}

	// Check that the group is a custom group corresponding to a team on GitHub, and find the Team ID
	if !isGroupNameForTeam(group.Name) {
		return nil, fmt.Errorf("error listing permissions of GitHub Org %s: group '%s' does not correspond to a team", company.Name, group.Name)
	}
	if group.ExternalID == nil || group.ExternalID.ExternalSystem != GitHubSCMName {
		return nil, fmt.Errorf("error listing permissions of GitHub Org %s: group '%s' does not have a GitHub external ID", company.Name, group.Name)
	}
	teamID, err := githubIDFromExternalID(group.ExternalID.ResourceID)
	if err != nil {
		return nil, fmt.Errorf("error parsing group external id '%s' to GitHub team id: %w", group.ExternalID.ResourceID, err)
	}

	// Read the permissions for the team
	ghRepos, err := ListTeamRepoPermissions(ctx, client, teamID)
	if err != nil {
		return nil, fmt.Errorf("error listing repos for GitHub Org %s team ID %d (group %s): %w", company.Name, teamID, group.Name, err)
	}

	var grants []*models.Grant
	now := models.NewTime(time.Now())
	grantedBy := company.ID // let's say the permissions are granted by the company legal entity
	for _, ghRepo := range ghRepos {
		// Find the repo in the database
		repoExternalID := GitHubIDToExternalResourceID(ghRepo.GetID())
		repo, err := s.repoStore.ReadByExternalID(ctx, nil, repoExternalID)
		if err != nil {
			if gerror.ToNotFound(err) != nil {
				// NOTE: It is possible to see team repo permissions on GitHub for repos that the BuildBeaver app
				// doesn't have access to, if the app was only granted access to specific repos; this isn't an error
				s.Infof("repo '%s' that legal entity %s team %s has access to was not found in database; skipping granting permission",
					ghRepo.GetName(), company.Name, group.Name)
				continue
			} else {
				return nil, errors.Wrap(err, "error looking for repo in the database")
			}
		}

		// Make a set of grants that correspond to the set of GitHub permissions for the repo
		ghPermissions := permissionSetToList(ghRepo.GetPermissions())
		operations := findOperationsForGitHubRepoPermissions(ghPermissions)
		for _, operation := range operations {
			grant := models.NewGroupGrant(now, grantedBy, group.ID, *operation, repo.ID.ResourceID)
			grants = append(grants, grant)
		}
	}

	return grants, nil
}

func permissionSetToList(ghPermissions map[string]bool) []string {
	var results []string
	for ghPermission, granted := range ghPermissions {
		if granted {
			results = append(results, ghPermission)
		}
	}
	return results
}

func mergeOperations(opSet map[*models.Operation]bool, newOperations []*models.Operation) {
	for _, newOp := range newOperations {
		_, inSet := opSet[newOp]
		if !inSet {
			opSet[newOp] = true
		}
	}
}

// findOperationsForGitHubRepoPermissions finds a set of BuildBeaver access control operations that correspond to
// a set of permissions for a GitHub repo.
func findOperationsForGitHubRepoPermissions(ghPermissions []string) []*models.Operation {
	// Combine the operations granted by each GitHub permission into a set
	opSet := make(map[*models.Operation]bool)

	for _, permission := range ghPermissions {
		switch permission {
		case "admin":
			// From GitHub: "Admin: Can read, clone and push to this repository. Can also manage issues, pull requests,
			//               and repository settings, including adding collaborators."
			mergeOperations(opSet, []*models.Operation{
				models.RepoReadOperation,
				models.RepoUpdateOperation,
				models.RepoDeleteOperation,
				models.BuildCreateOperation,
				models.SecretCreateOperation,
				models.BuildReadOperation,
				models.BuildUpdateOperation,
				models.ArtifactCreateOperation,
				models.ArtifactReadOperation,
				models.ArtifactUpdateOperation,
				models.ArtifactDeleteOperation,
			})
		case "maintain":
			// From GitHub: "Maintain: Can read, clone and push to this repository. They can also manage issues,
			//               pull requests, and some repository settings"
			mergeOperations(opSet, []*models.Operation{
				models.RepoReadOperation,
				models.RepoUpdateOperation,
				models.BuildCreateOperation,
				models.SecretCreateOperation,
				models.BuildReadOperation,
				models.BuildUpdateOperation,
				models.ArtifactReadOperation,
				models.ArtifactDeleteOperation,
			})
		case "push":
			// From GitHub: "Write: Can read, clone and push to this repository. Can also manage issues and pull requests."
			mergeOperations(opSet, []*models.Operation{
				models.RepoReadOperation,
				models.RepoUpdateOperation,
				models.BuildCreateOperation,
				models.SecretCreateOperation,
				models.BuildReadOperation,
				models.ArtifactReadOperation,
				models.ArtifactDeleteOperation,
			})
		case "triage":
			// From GitHub: "Triage: Can read and clone this repository. Can also manage issues and pull requests."
			mergeOperations(opSet, []*models.Operation{
				models.RepoReadOperation,
				models.BuildReadOperation,
				models.ArtifactReadOperation,
			})
		case "pull":
			// From GitHub: "Read: Can read and clone this repository. Can also open and comment on issues and pull requests."
			mergeOperations(opSet, []*models.Operation{
				models.RepoReadOperation,
				models.BuildReadOperation,
				models.ArtifactReadOperation,
			})
		default:
			// don't add any operations
		}
	}

	// Convert the map (i.e. set) back to a list to return
	results := make([]*models.Operation, 0, len(opSet))
	for op := range opSet {
		results = append(results, op)
	}
	return results
}

func (s *GitHubService) legalEntityDataFromGitHubUser(
	user *github.User,
	installationID int64,
) (*models.LegalEntityData, error) {
	name := gitHubLegalEntityName(user.GetLogin())
	externalID := GitHubIDToExternalResourceID(user.GetID())

	metadata := &LegalEntityMetadata{
		Login:                    user.GetLogin(),
		InstallationID:           installationID,
		OrgDefaultRepoPermission: "", // not an organization, so leave this empty
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("error marshaling legal entity metadata JSON for GitHub User: %w", err)
	}

	return models.NewPersonLegalEntityData(
		name,
		user.GetName(),
		user.GetEmail(),
		&externalID,
		string(metadataJSON)), nil
}

func (s *GitHubService) legalEntityDataFromGitHubOrg(
	org *github.Organization,
	installationID int64,
) (*models.LegalEntityData, error) {
	name := gitHubLegalEntityName(org.GetLogin())
	externalID := GitHubIDToExternalResourceID(org.GetID())

	metadata := &LegalEntityMetadata{
		Login:                    org.GetLogin(),
		InstallationID:           installationID,
		OrgDefaultRepoPermission: org.GetDefaultRepoPermission(),
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("error marshaling legal entity metadata JSON for GitHub Organization: %w", err)
	}

	return models.NewCompanyLegalEntityData(
		name,
		org.GetName(),
		org.GetEmail(),
		&externalID,
		string(metadataJSON)), nil
}

// repoDataFromGitHubRepo converts a GitHub Repo (owned by a specified BuildBeaver Legal Entity) to a BuildBeaver
// repo object. The returned repo object contains data but no ID, and may or may not match a record in the database.
func (s *GitHubService) repoDataFromGitHubRepo(
	ghRepo *github.Repository,
	ownerLegalEntity *models.LegalEntity,
) (*models.Repo, error) {
	// Use the installation ID from the owner, which is in the GitHub metadata
	legalEntityMetadata, err := GetLegalEntityMetadata(ownerLegalEntity)
	if err != nil {
		return nil, err
	}

	now := models.NewTime(time.Now())
	repoExternalID := GitHubIDToExternalResourceID(ghRepo.GetID())

	repoName := ghRepo.GetName()
	repoOwnerLogin := ghRepo.GetOwner().GetLogin() // use owner.GetLogin() rather than owner.GetName()
	if repoOwnerLogin != legalEntityMetadata.Login {
		return nil, fmt.Errorf("error expected repo to belong to %q, found %q", legalEntityMetadata.Login, repoOwnerLogin)
	}
	repoMetadata := NewRepoMetadata(legalEntityMetadata.InstallationID, repoName, repoOwnerLogin)
	repoMetadataJSON, err := json.Marshal(repoMetadata)
	if err != nil {
		return nil, fmt.Errorf("error marshaling repo metadata JSON: %v", err)
	}

	repo := models.NewRepo(
		now,
		gitHubRepoName(repoName),
		ownerLegalEntity.ID,
		ghRepo.GetDescription(),
		ghRepo.GetSSHURL(),
		ghRepo.GetCloneURL(),
		ghRepo.GetHTMLURL(),
		ghRepo.GetDefaultBranch(),
		ghRepo.GetPrivate(),
		false, // this field won't overwrite any existing values
		nil,   // this field won't overwrite any existing values
		&repoExternalID,
		string(repoMetadataJSON),
	)

	return repo, nil
}

// groupDataFromGitHubTeam converts a GitHub Team (owned by the specified BuildBeaver Legal Entity) to a BuildBeaver
// group object that can be used to represent the team within BuildBeaver.
// The returned group object contains data but no ID, and may or may not match a record in the database.
func (s *GitHubService) groupDataFromGitHubTeam(
	ghTeam *github.Team,
	orgLegalEntity *models.LegalEntity,
) (*models.Group, error) {
	now := models.NewTime(time.Now())
	// Use the team ID as an external ID for the group
	groupExternalID := GitHubIDToExternalResourceID(ghTeam.GetID())
	// Make group name from the slug, not the name
	groupName := gitHubGroupNameForTeam(ghTeam.GetSlug())

	// NOTE: ghTeam.Organization is not filled out when teams are read via the API and also not filled out
	// in events delivered via Webhooks, so it can't be checked against orgLegalEntity

	groupForTeam := models.NewGroup(
		now,
		orgLegalEntity.ID,
		groupName,
		ghTeam.GetDescription(),
		false,
		&groupExternalID,
	)

	return groupForTeam, nil
}

func gitHubLegalEntityName(name string) models.ResourceName {
	return models.ResourceName(strings.ToLower(name))
}

func gitHubRepoName(name string) models.ResourceName {
	return models.ResourceName(strings.ToLower(name))
}

func gitHubGroupNameForTeam(teamSlug string) models.ResourceName {
	// TODO: Should we be calling Validate() and making sure the team slug (and resulting name) is not too long?
	// We need a prefix for groups representing GitHub teams to prevent namespace clashes
	// with standard groups and groups from other SCMs
	return models.ResourceName(groupNamePrefixForGitHubTeam + strings.ToLower(teamSlug))
}

// isGroupNameForTeam returns true if the specified group name represents the name of an access control group
// created to represent a GitHub team.
func isGroupNameForTeam(groupName models.ResourceName) bool {
	return strings.HasPrefix(groupName.String(), groupNamePrefixForGitHubTeam)
}
