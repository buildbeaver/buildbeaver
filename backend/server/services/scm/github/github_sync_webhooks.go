package github

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/go-github/v28/github"
	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/models"
)

func (s *GitHubService) handleInstallationEvent(ctx context.Context, payload []byte) error {
	event := &github.InstallationEvent{}
	err := json.Unmarshal(payload, event)
	if err != nil {
		return errors.Wrap(err, "error unmarshalling event")
	}

	installationID := event.GetInstallation().GetID()
	if installationID == 0 {
		return fmt.Errorf("error processing GitHub Installation event: installation ID not supplied")
	}

	// Read extra info about the user or organization for the GitHub account that BuildBeaver was installed on.
	// Convert to legal entity data that can be used to find or create a legal entity in the database.
	ghAccount := event.GetInstallation().GetAccount()
	var accountLegalEntityData *models.LegalEntityData
	// Don't do this if the installation was deleted since an installation client will no longer work
	if event.GetAction() != "deleted" {
		accountLegalEntityData, err = s.readAccountDetailsForInstallation(
			ctx,
			ghAccount.GetID(),
			ghAccount.GetLogin(),
			ghAccount.GetType(),
			installationID,
		)
		if err != nil {
			return fmt.Errorf("error processing GitHub Installation event (action %s): %w", event.GetAction(), err)
		}
	}

	s.Infof("Received a GitHub Installation event for account ID %d, login %s, installation ID %d, action %s",
		event.GetInstallation().GetAccount().GetID(), event.GetInstallation().GetAccount().GetLogin(), installationID, event.GetAction())

	switch event.GetAction() {
	case "created":
		// Perform a full sync for the account to discover its repos, teams and permissions
		_, _, err = s.syncService.SyncLegalEntity(ctx, accountLegalEntityData, 0)
		if err != nil {
			return fmt.Errorf("error setting up new installation of GitHub app from GitHub Installation created event: %w", err)
		}
	case "deleted":
		s.Warnf("GitHub app was uninstalled from GitHub account '%s'", event.GetInstallation().GetAccount().GetLogin())
		err = s.syncService.RemoveInstallationForLegalEntity(ctx, accountLegalEntityData)
		if err != nil {
			return fmt.Errorf("error removing installation of GitHub app in response to GitHub Installation deleted event: %w", err)
		}
	case "suspend", "unsuspend", "new_permissions_accepted":
		s.Infof("Nothing to do for GitHub Installation event with action %s", event.GetAction())
	default:
		s.Infof("Ignoring GitHub Installation event with unknown action value %s", event.GetAction())
	}
	return nil
}

func (s *GitHubService) handleInstallationTargetEvent(ctx context.Context, payload []byte) error {
	// TODO: Why is there no type for this event in the client library? Do we need to upgrade to newer client?
	// TODO: Is this the event that would be received when the base permissions for repos in an organization
	// TODO: changes? It doesn't seem to be sent out in this circumstance.
	//event := &github.InstallationTargetEvent{}
	//err := json.Unmarshal(payload, event)
	//if err != nil {
	//	return errors.Wrap(err, "error unmarshalling event")
	//}
	s.Infof("Received Installation Target event, not implemented so discarding event")
	return nil
}

func (s *GitHubService) handleInstallationRepositoriesEvent(ctx context.Context, payload []byte) error {
	event := &github.InstallationRepositoriesEvent{}
	err := json.Unmarshal(payload, event)
	if err != nil {
		return errors.Wrap(err, "error unmarshalling event")
	}

	installationID := event.GetInstallation().GetID()
	if installationID == 0 {
		return fmt.Errorf("error processing GitHub 'Installation Repositories' event: installation ID not supplied")
	}
	accountID := event.GetInstallation().GetAccount().GetID()
	accountLogin := event.GetInstallation().GetAccount().GetLogin()
	accountLegalEntity, err := s.readWebhookLegalEntity(ctx, accountID, installationID)
	if err != nil {
		return fmt.Errorf("error processing GitHub 'Installation Repositories' event (action %s): %w", event.GetAction(), err)
	}

	s.Infof("Received a GitHub 'Installation Repositories' event for account ID %d, org login %s, action %s",
		accountID, accountLogin, event.GetAction())

	switch event.GetAction() {
	case "added", "removed":
		// Don't get too clever; just re-sync all repos for the legal entity when something changes
		_, err = s.syncService.SyncReposForLegalEntity(ctx, s, accountLegalEntity)
		if err != nil {
			return fmt.Errorf("error syncing repos for GitHub 'Installation Repositories' event (action %s): %w", event.GetAction(), err)
		}

	default:
		s.Infof("Ignoring GitHub 'Installation Repositories' event with unknown action=%q", event.GetAction())
	}

	return nil
}

func (s *GitHubService) handleOrganizationEvent(ctx context.Context, payload []byte) error {
	event := &github.OrganizationEvent{}
	err := json.Unmarshal(payload, event)
	if err != nil {
		return errors.Wrap(err, "error unmarshalling event")
	}

	orgLegalEntity, err := s.readWebhookLegalEntity(ctx, event.GetOrganization().GetID(), event.GetInstallation().GetID())
	if err != nil {
		return fmt.Errorf("error processing GitHub Organization event (action %s): %w", event.GetAction(), err)
	}
	orgLegalEntityMetadata, err := GetLegalEntityMetadata(orgLegalEntity)
	if err != nil {
		return err
	}

	// When a member is added or removed, convert the GitHub user to legal entity data that can be used to
	// find or create a legal entity in the database
	var memberData *models.LegalEntityData
	if event.GetAction() == "member_added" || event.GetAction() == "member_removed" {
		memberData, err = s.legalEntityDataFromGitHubUser(event.GetMembership().GetUser(), legalEntityInstallationUnset)
		if err != nil {
			return fmt.Errorf("error processing GitHub Membership event (action %s): error making legal entity from GitHub user: %w",
				event.GetAction(), err)
		}
	}

	s.Infof("Received a GitHub Organization event for org ID %d, org login %s, action %s",
		event.GetOrganization().GetID(), event.GetOrganization().GetLogin(), event.GetAction())

	switch event.GetAction() {
	case "member_added":
		// Add the new member to the company legal entity
		err = s.syncService.AddCompanyMember(ctx, nil, s, orgLegalEntity, memberData)
		if err != nil {
			return fmt.Errorf("error adding new company member from GitHub Organization event: %w", err)
		}

		// Add the new member to appropriate standard groups; otherwise they won't have any permissions
		ghOrgRoleStr := event.GetMembership().GetRole()
		// Use DefaultRepoPermission from the legal entity in our database; this field is not populated
		// in the organization in the incoming event data
		ghOrgDefaultRepoPermission := orgLegalEntityMetadata.OrgDefaultRepoPermission
		standardGroupName := findGroupForGitHubPermissions(ghOrgRoleStr, ghOrgDefaultRepoPermission)
		if standardGroupName != "" {
			err = s.syncService.AddStandardGroupMember(ctx, nil, s, orgLegalEntity, standardGroupName, memberData)
			if err != nil {
				return fmt.Errorf("error adding new company member to standard group %s in response to GitHub Organization event: %w", standardGroupName, err)
			}
		} else {
			s.Warnf("User %s added to org %s but not added to any standard groups: GitHub org role '%s', org default repo permission '%s'",
				memberData.Name, orgLegalEntity.Name, ghOrgRoleStr, ghOrgDefaultRepoPermission)
		}

	case "member_removed":
		err = s.syncService.RemoveCompanyMember(ctx, nil, s, orgLegalEntity, memberData)
		if err != nil {
			return fmt.Errorf("error removing company member based on GitHub Organization event: %w", err)
		}

	case "renamed":
		newLoginName := event.GetOrganization().GetLogin()
		// Re-read all the organization data and tell sync to update the organization in the database
		newLegalEntityData, err := s.readAccountDetailsForInstallation(
			ctx,
			event.GetOrganization().GetID(),
			newLoginName,
			"Organization",
			event.GetInstallation().GetID(),
		)
		if err != nil {
			return fmt.Errorf("error reading new organization data based on GitHub Organization 'renamed' event: %w", err)
		}
		_, err = s.syncService.UpsertLegalEntity(ctx, nil, newLegalEntityData)
		if err != nil {
			return fmt.Errorf("error updating company name details on GitHub Organization 'renamed' event: %w", err)
		}

	case "deleted":
		s.Warnf("GitHub organization using BuildBeaver was deleted, org '%s'", event.GetOrganization().GetLogin())
		err = s.syncService.RemoveInstallationForLegalEntity(ctx, &orgLegalEntity.LegalEntityData)
		if err != nil {
			return fmt.Errorf("error removing installation of GitHub app in response to GitHub Organization deleted event: %w", err)
		}
	case "member_invited":
		s.Tracef("Nothing to do for GitHub Organization event with action=%q", event.GetAction())

	default:
		s.Infof("Ignoring GitHub Organization event with unknown action=%q", event.GetAction())
	}

	return nil
}

func (s *GitHubService) handleTeamEvent(ctx context.Context, payload []byte) error {
	event := &github.TeamEvent{}
	err := json.Unmarshal(payload, event)
	if err != nil {
		return errors.Wrap(err, "error unmarshalling event")
	}

	orgLegalEntity, err := s.readWebhookLegalEntity(ctx, event.GetOrg().GetID(), event.GetInstallation().GetID())
	if err != nil {
		return fmt.Errorf("error processing GitHub Team event (action %s): %w", event.GetAction(), err)
	}

	// When adding or removing a repo from the team, or editing the team's data or permissions,
	// we expect the group (and repo) to already be in the database
	var group *models.Group
	if event.GetAction() == "edited" || event.GetAction() == "added_to_repository" || event.GetAction() == "removed_from_repository" {
		group, err = s.readWebhookGroupForTeam(ctx, event.GetTeam().GetID(), event.GetTeam().GetSlug())
		if err != nil {
			return fmt.Errorf("error processing GitHub Team event (action %s): %w", event.GetAction(), err)
		}
	}

	s.Infof("Received a GitHub Team event for org ID %d, org login %s, team ID %d, group name %s, action %s",
		event.GetOrg().GetID(), event.GetOrg().GetLogin(), event.GetTeam().GetID(), event.GetTeam().GetName(), event.GetAction())

	switch event.GetAction() {
	case "created":
		groupData, err := s.groupDataFromGitHubTeam(event.GetTeam(), orgLegalEntity)
		if err != nil {
			return fmt.Errorf("error processing GitHub Team event (action %s): %w", event.GetAction(), err)
		}
		_, err = s.syncService.UpsertCompanyCustomGroup(ctx, nil, s, orgLegalEntity, groupData)
		if err != nil {
			return fmt.Errorf("error adding new custom group from GitHub Team event (action %s): %w", event.GetAction(), err)
		}

	case "deleted":
		groupExternalID := GitHubIDToExternalResourceID(event.GetTeam().GetID())
		err = s.syncService.RemoveGroupByExternalID(ctx, nil, groupExternalID)
		if err != nil {
			return fmt.Errorf("error removing custom group based on GitHub Team event (action %s): %w", event.GetAction(), err)
		}

	case "edited":
		// Edited actions are sent when the team itself is edited, or when a repo's permission level for the team is changed
		groupData, err := s.groupDataFromGitHubTeam(event.GetTeam(), orgLegalEntity)
		if err != nil {
			return fmt.Errorf("error processing GitHub Team event (action %s): %w", event.GetAction(), err)
		}
		_, err = s.syncService.UpsertCompanyCustomGroup(ctx, nil, s, orgLegalEntity, groupData)
		if err != nil {
			return fmt.Errorf("error updating custom group from GitHub Team event (action %s): %w", event.GetAction(), err)
		}
		if event.GetRepo() != nil {
			// A repo's permission level for the team has (probably) been edited
			// Don't get too clever; just re-sync all permissions for the team
			err = s.syncService.SyncCompanyGroupPermissions(ctx, nil, s, orgLegalEntity, group)
			if err != nil {
				return fmt.Errorf("error syncing team permissions for GitHub Team event (action %s): %w", event.GetAction(), err)
			}
		}

	case "added_to_repository":
		// Don't get too clever; just re-sync all permissions for the team when they change
		err = s.syncService.SyncCompanyGroupPermissions(ctx, nil, s, orgLegalEntity, group)
		if err != nil {
			return fmt.Errorf("error syncing team permissions for GitHub Team event (action %s): %w", event.GetAction(), err)
		}

	case "removed_from_repository":
		// Don't get too clever; just re-sync all permissions for the team when they change
		err = s.syncService.SyncCompanyGroupPermissions(ctx, nil, s, orgLegalEntity, group)
		if err != nil {
			return fmt.Errorf("error syncing team permissions for GitHub Team event (action %s): %w", event.GetAction(), err)
		}

	default:
		s.Infof("Ignoring GitHub Team event with unknown action=%q", event.GetAction())
	}

	return nil
}

// handleMembershipEvent handles GitHub webhook events relating to team membership.
func (s *GitHubService) handleMembershipEvent(ctx context.Context, payload []byte) error {
	event := &github.MembershipEvent{}
	err := json.Unmarshal(payload, event)
	if err != nil {
		return errors.Wrap(err, "error unmarshalling event")
	}

	orgLegalEntity, err := s.readWebhookLegalEntity(ctx, event.GetOrg().GetID(), event.GetInstallation().GetID())
	if err != nil {
		return fmt.Errorf("error processing GitHub Membership event (action %s): %w", event.GetAction(), err)
	}

	group, err := s.readWebhookGroupForTeam(ctx, event.GetTeam().GetID(), event.GetTeam().GetSlug())
	if err != nil {
		if event.GetAction() == "removed" {
			// When a GitHub team is deleted, sometimes Membership 'removed' events arrive after a Team event has been
			// delivered that causes the team's group to be deleted, so we need to tolerate not finding the group.
			s.Infof("Unable to find group to remove user when processing GitHub Membership event (action %s); this is expected when teams are deleted from GitHub: %s", event.GetAction(), err)
			return nil
		} else {
			return fmt.Errorf("error processing GitHub Membership event (action %s): %w", event.GetAction(), err)
		}
	}

	// Convert the GitHub user to legal entity data that can be used to find or create a legal entity in the database
	memberData, err := s.legalEntityDataFromGitHubUser(event.GetMember(), legalEntityInstallationUnset)
	if err != nil {
		return fmt.Errorf("error processing GitHub Membership event (action %s): error making legal entity from GitHub user: %w",
			event.GetAction(), err)
	}

	s.Infof("Received a GitHub Membership event for org ID %d, org login %s, team ID %d, group name %s, action %s, user name %s",
		event.GetOrg().GetID(), event.GetOrg().GetLogin(), event.GetTeam().GetID(), group.Name, event.GetAction(), memberData.Name)

	switch event.GetAction() {
	case "added":
		err = s.syncService.AddCompanyGroupMember(ctx, nil, s, orgLegalEntity, group, memberData)
		if err != nil {
			return fmt.Errorf("error adding new member to group in GitHub Membership event: %w", err)
		}
	case "removed":
		err = s.syncService.RemoveCompanyGroupMember(ctx, nil, s, orgLegalEntity, group, memberData)
		if err != nil {
			return fmt.Errorf("error removing member from group in GitHub Membership event: %w", err)
		}
	default:
		s.Infof("Ignoring GitHub GitHub Membership event with unknown action '%s'", event.GetAction())
	}
	return nil
}

func (s *GitHubService) handleRepositoryEvent(ctx context.Context, payload []byte) error {
	event := &github.RepositoryEvent{}
	err := json.Unmarshal(payload, event)
	if err != nil {
		return errors.Wrap(err, "error unmarshalling event")
	}

	var ownerLegalEntity *models.LegalEntity
	if event.GetRepo().GetOrganization() != nil {
		ownerLegalEntity, err = s.readWebhookLegalEntity(ctx, event.GetRepo().GetOrganization().GetID(), event.GetInstallation().GetID())
		if err != nil {
			return fmt.Errorf("error processing GitHub Repository event (action %s): %w", event.GetAction(), err)
		}
	} else {
		ownerLegalEntity, err = s.readWebhookLegalEntity(ctx, event.GetRepo().GetOwner().GetID(), event.GetInstallation().GetID())
		if err != nil {
			return fmt.Errorf("error processing GitHub Repository event (action %s): %w", event.GetAction(), err)
		}
	}

	// Convert the GitHub repo to BuildBeaver repo data that can be used to find or create a repo in the database
	repoData, err := s.repoDataFromGitHubRepo(event.GetRepo(), ownerLegalEntity)
	if err != nil {
		return fmt.Errorf("error processing GitHub Repository event (action %s): %w", event.GetAction(), err)
	}

	s.Infof("Received a GitHub Repository event for owner legal entity '%s' (ID '%s'), action %s, repo %s (external ID %s)",
		ownerLegalEntity.Name, ownerLegalEntity.ID, event.GetAction(), repoData.Name, repoData.ExternalID)

	switch event.GetAction() {
	case "created":
		err = s.syncService.UpsertRepo(ctx, nil, repoData)
		if err != nil {
			return fmt.Errorf("error adding new repo from GitHub Repository event: %w", err)
		}
	case "deleted":
		err = s.syncService.RemoveRepoByExternalID(ctx, nil, *repoData.ExternalID)
		if err != nil {
			return fmt.Errorf("error removing repo based on GitHub Repository event: %w", err)
		}
	case "edited", "renamed":
		err = s.syncService.UpsertRepo(ctx, nil, repoData)
		if err != nil {
			return fmt.Errorf("error updating repo from GitHub Repository event: %w", err)
		}
	case "archived", "unarchived", "transferred", "publicized", "privatized":
		s.Tracef("Nothing to do for GitHub Repository event with action %s", event.GetAction())
	default:
		s.Infof("Ignoring GitHub Repository event with unknown action %s", event.GetAction())
	}
	return nil
}

// readWebhookOrgLegalEntity reads the Legal Entity that corresponds to a GitHub Organization or User from a Webhook.
// ghAccountID is the GitHub ID of the user or organization (also known as the 'account ID').
// The legal entity must already exist in the database, or an error is returned.
// The legal entity must have an installation ID recorded against it that matches the specified value from the event,
// or an error is returned because we are receiving Webhook events for the wrong installation.
func (s *GitHubService) readWebhookLegalEntity(ctx context.Context, ghAccountID int64, eventInstallationID int64) (*models.LegalEntity, error) {
	if ghAccountID == 0 {
		return nil, fmt.Errorf("error: event has zero for user/org/account ID")
	}
	orgExternalID := GitHubIDToExternalResourceID(ghAccountID)
	legalEntity, err := s.legalEntityService.ReadByExternalID(ctx, nil, orgExternalID)
	if err != nil {
		return nil, fmt.Errorf("error finding legal entity with externalID %s: %w", orgExternalID, err)
	}
	legalEntityMetadata, err := GetLegalEntityMetadata(legalEntity)
	if err != nil {
		return nil, err
	}
	if legalEntityMetadata.InstallationID != eventInstallationID {
		return nil, fmt.Errorf("error: Legal Entity %s has GitHub Installation ID %d, does not match event Installation ID %d",
			legalEntity.Name, legalEntityMetadata.InstallationID, eventInstallationID)
	}
	s.Tracef("Found legal entity name %s, ID %s for external ID %s", legalEntity.Name, legalEntity.ID, orgExternalID)
	return legalEntity, nil
}

// readWebhookGroupForTeam reads the access control Group that corresponds to a GitHub Team from a Webhook.
// The Group must already exist in the database or an error is returned.
// As a double check, the team slug must also match the group name in the database, or an error is returned.
func (s *GitHubService) readWebhookGroupForTeam(ctx context.Context, ghTeamID int64, ghTeamSlug string) (*models.Group, error) {
	if ghTeamID == 0 {
		return nil, fmt.Errorf("error: event has zero Team ID")
	}
	groupExternalID := GitHubIDToExternalResourceID(ghTeamID)
	group, err := s.groupService.ReadByExternalID(ctx, nil, groupExternalID)
	if err != nil {
		return nil, fmt.Errorf("error finding group for GitHub team with externalID %s: %w", groupExternalID, err)
	}
	groupNameFromSlug := gitHubGroupNameForTeam(ghTeamSlug) // make group name from the slug, not the name
	if group.Name != groupNameFromSlug {
		return nil, fmt.Errorf("error finding group for GitHub team: group name %s does not match name derived from team slug %s",
			group.Name, groupNameFromSlug)
	}
	s.Tracef("Found group name %s, ID %s for external ID %s", group.Name, group.ID, groupExternalID)
	return group, nil
}

// readWebhookRepo reads the BuildBeaver database repo oup that corresponds to a GitHub repo from a Webhook.
// The Repo must already exist in the database or an error is returned.
// As a double check, the repo name must also match the name in the database, or an error is returned.
func (s *GitHubService) readWebhookRepo(ctx context.Context, ghRepoID int64, ghRepoName string) (*models.Repo, error) {
	if ghRepoID == 0 {
		return nil, fmt.Errorf("error: event has zero Repo ID")
	}
	repoExternalID := GitHubIDToExternalResourceID(ghRepoID)
	repo, err := s.repoStore.ReadByExternalID(ctx, nil, repoExternalID)
	if err != nil {
		return nil, fmt.Errorf("error finding repo with externalID %s: %w", repoExternalID, err)
	}
	repoNameFromGitHub := gitHubRepoName(ghRepoName)
	if repo.Name != repoNameFromGitHub {
		return nil, fmt.Errorf("error finding repo: repo name %s does not match name derived from GitHub name %s",
			repo.Name, ghRepoName)
	}
	s.Tracef("Found repo name %s, ID %s for repo external ID %s", repo.Name, repo.ID, repoExternalID)
	return repo, nil
}
