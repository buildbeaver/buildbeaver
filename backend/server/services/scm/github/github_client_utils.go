package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v28/github"
)

const githubResultsPageSize = 30

type githubListReposFunction func(ctx context.Context, ghClient *github.Client, listOptions github.ListOptions) ([]*github.Repository, *github.Response, error)

// ListAllReposForCurrentUser lists all repos in GitHub for the currently authenticated user.
// All pages of results are read and combined into a single list.
func ListAllReposForCurrentUser(ctx context.Context, ghClient *github.Client) ([]*github.Repository, error) {
	return listAllRepos(ctx, ghClient,
		func(ctx context.Context, ghClient *github.Client, listOptions github.ListOptions) ([]*github.Repository, *github.Response, error) {
			return ghClient.Repositories.List(ctx, "", &github.RepositoryListOptions{
				Visibility:  "all",
				Affiliation: "owner",
				ListOptions: listOptions,
			})
		},
	)
}

// ListAllReposForCompany lists all repos in GitHub for the company with the specified GitHub login.
// All pages of results are read and combined into a single list.
func ListAllReposForCompany(ctx context.Context, ghClient *github.Client, companyGithubLogin string) ([]*github.Repository, error) {
	return listAllRepos(ctx, ghClient,
		func(ctx context.Context, ghClient *github.Client, listOptions github.ListOptions) ([]*github.Repository, *github.Response, error) {
			return ghClient.Repositories.ListByOrg(ctx, companyGithubLogin, &github.RepositoryListByOrgOptions{
				ListOptions: listOptions,
			})
		},
	)
}

// ListAllReposForInstallation lists all repos in GitHub accessible to the supplied installation client.
// All pages of results are read and combined into a single list.
func ListAllReposForInstallation(ctx context.Context, installationClient *github.Client) ([]*github.Repository, error) {
	return listAllRepos(ctx, installationClient,
		func(ctx context.Context, installationClient *github.Client, listOptions github.ListOptions) ([]*github.Repository, *github.Response, error) {
			return installationClient.Apps.ListRepos(ctx, &listOptions)
		},
	)
}

func listAllRepos(ctx context.Context, ghClient *github.Client, fn githubListReposFunction) ([]*github.Repository, error) {
	var (
		results   []*github.Repository
		morePages = true
		page      = 1
	)
	for morePages {
		nextRepos, response, err := fn(ctx, ghClient, github.ListOptions{
			Page:    page,
			PerPage: githubResultsPageSize,
		})
		if err != nil {
			return nil, fmt.Errorf("error listing repos from GitHub: %w", err)
		}
		results = append(results, nextRepos...)
		if response.NextPage != 0 {
			page++
		} else {
			morePages = false
		}
	}
	return results, nil
}

// ListAllOrganizationsForCurrentUser lists all GitHub organizations the currently authenticated user is a member of.
// All pages of results are read and combined into a single list.
func ListAllOrganizationsForCurrentUser(ctx context.Context, ghClient *github.Client) ([]*github.Organization, error) {
	var (
		results   []*github.Organization
		morePages = true
		page      = 1
	)
	for morePages {
		nextOrgs, response, err := ghClient.Organizations.List(ctx, "", &github.ListOptions{
			Page:    page,
			PerPage: githubResultsPageSize,
		})
		if err != nil {
			return nil, fmt.Errorf("error listing user organizations from GitHub: %w", err)
		}
		results = append(results, nextOrgs...)
		if response.NextPage != 0 {
			page++
		} else {
			morePages = false
		}
	}
	return results, nil
}

// ListAllOrganizationMembershipsForCurrentUser lists all GitHub organization memberships (including the role
// for each membership) for the currently authenticated user, across all organizations.
// All pages of results are read and combined into a single list.
func ListAllOrganizationMembershipsForCurrentUser(ctx context.Context, ghClient *github.Client) ([]*github.Membership, error) {
	var (
		results   []*github.Membership
		morePages = true
		page      = 1
	)
	for morePages {
		nextMemberships, response, err := ghClient.Organizations.ListOrgMemberships(ctx, &github.ListOrgMembershipsOptions{
			State: "active",
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: githubResultsPageSize,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("error reading org memberships for authenticated user in GitHub: %w", err)
		}
		results = append(results, nextMemberships...)
		if response.NextPage != 0 {
			page++
		} else {
			morePages = false
		}
	}
	return results, nil
}

// ListAllDeployKeysForRepo lists all GitHub deployment keys for the specified repo.
// All pages of results are read and combined into a single list.
func ListAllDeployKeysForRepo(ctx context.Context, ghClient *github.Client, ghRepo *github.Repository) ([]*github.Key, error) {
	var (
		results   []*github.Key
		morePages = true
		page      = 1
	)
	for morePages {
		nextKeys, response, err := ghClient.Repositories.ListKeys(ctx, ghRepo.Owner.GetLogin(), ghRepo.GetName(), &github.ListOptions{
			Page:    page,
			PerPage: githubResultsPageSize,
		})
		if err != nil {
			return nil, fmt.Errorf("error listing repo keys from GitHub: %w", err)
		}
		results = append(results, nextKeys...)
		if response.NextPage != 0 {
			page++
		} else {
			morePages = false
		}
	}
	return results, nil
}

// ListAllAppInstallations lists all installations of the BuildBeaver GitHub app.
// All pages of results are read and combined into a single list.
func ListAllAppInstallations(ctx context.Context, ghClient *github.Client) ([]*github.Installation, error) {
	var (
		results   []*github.Installation
		morePages = true
		page      = 1
	)
	for morePages {
		nextInstallations, response, err := ghClient.Apps.ListInstallations(ctx, &github.ListOptions{
			Page:    page,
			PerPage: githubResultsPageSize,
		})
		if err != nil {
			return nil, fmt.Errorf("error listing BuildBeaver app installations from GitHub: %w", err)
		}
		results = append(results, nextInstallations...)
		if response.NextPage != 0 {
			page++
		} else {
			morePages = false
		}
	}
	return results, nil
}

// ListCompanyMembers reads members of a company, with a specified role.
// If "all" is specified as the role then all company members will be listed.
// As of Nov 2022, possible values for role are "all", "admin" or "member".
func ListCompanyMembers(ctx context.Context, ghClient *github.Client, ghOrgName string, ghRole string) ([]*github.User, error) {
	var (
		results   []*github.User
		morePages = true
		page      = 1
	)
	for morePages {
		nextMembers, response, err := ghClient.Organizations.ListMembers(ctx, ghOrgName, &github.ListMembersOptions{
			PublicOnly: false,
			Filter:     "all",
			Role:       ghRole,
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: githubResultsPageSize,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("error listing GitHub Org members for org %s, role %s: %w", ghOrgName, ghRole, err)
		}
		results = append(results, nextMembers...)
		if response.NextPage != 0 {
			page++
		} else {
			morePages = false
		}
	}
	return results, nil
}

// ListTeamMembers reads members of a GitHub team (specified by team ID) within a company.
// If role is supplied then only team members with that role will be listed.
// As of Nov 2022, possible values for role are "all", "maintainer" or "member".
func ListTeamMembers(ctx context.Context, ghClient *github.Client, teamID int64, ghRole string) ([]*github.User, error) {
	var (
		results   []*github.User
		morePages = true
		page      = 1
	)
	for morePages {
		nextMembers, response, err := ghClient.Teams.ListTeamMembers(ctx, teamID, &github.TeamListTeamMembersOptions{
			Role: ghRole,
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: githubResultsPageSize,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("error listing GitHub team members for team %d, role %s: %w", teamID, ghRole, err)
		}
		results = append(results, nextMembers...)
		if response.NextPage != 0 {
			page++
		} else {
			morePages = false
		}
	}
	return results, nil
}

// ListTeamRepoPermissions reads the repos that GitHub team (specified by team ID) has access to, including the
// access rights the team has for each repo.
// If role is supplied then only team members with that role will be listed.
// As of Nov 2022, possible values for role are "all", "maintainer" or "member".
func ListTeamRepoPermissions(ctx context.Context, ghClient *github.Client, teamID int64) ([]*github.Repository, error) {
	var (
		results   []*github.Repository
		morePages = true
		page      = 1
	)
	for morePages {
		nextRepos, response, err := ghClient.Teams.ListTeamRepos(ctx, teamID, &github.ListOptions{
			Page:    page,
			PerPage: githubResultsPageSize,
		})
		if err != nil {
			return nil, fmt.Errorf("error listing GitHub team repos and permissions for team %d: %w", teamID, err)
		}
		results = append(results, nextRepos...)
		if response.NextPage != 0 {
			page++
		} else {
			morePages = false
		}
	}
	return results, nil
}

// ListAllTeamsForOrganization lists all GitHub teams within the specified organization.
// All pages of results are read and combined into a single list.
func ListAllTeamsForOrganization(ctx context.Context, ghClient *github.Client, ghOrgName string) ([]*github.Team, error) {
	var (
		results   []*github.Team
		morePages = true
		page      = 1
	)
	for morePages {
		nextTeams, response, err := ghClient.Teams.ListTeams(ctx, ghOrgName, &github.ListOptions{
			Page:    page,
			PerPage: githubResultsPageSize,
		})
		if err != nil {
			return nil, fmt.Errorf("error listing teams for organization '%s' from GitHub: %w", ghOrgName, err)
		}
		results = append(results, nextTeams...)
		if response.NextPage != 0 {
			page++
		} else {
			morePages = false
		}
	}
	return results, nil
}

type githubListFunction func(ctx context.Context, ghClient *github.Client, listOptions *github.ListOptions) (items []interface{}, response *github.Response, err error)

// listAll calls a GitHub API function and pages through all results to combine them into a single list.
// NOTE: Writing specific functions to call this is very ugly without generics, so it's easier to duplicate the logic.
func listAll(ctx context.Context, ghClient *github.Client, fn githubListFunction) (items []interface{}, err error) {
	var (
		results   []interface{}
		morePages = true
		page      = 1
	)
	for morePages {
		nextResults, response, err := fn(ctx, ghClient, &github.ListOptions{
			Page:    page,
			PerPage: githubResultsPageSize,
		})
		if err != nil {
			return nil, err
		}
		results = append(results, nextResults...)
		if response.NextPage != 0 {
			page++
		} else {
			morePages = false
		}
	}
	return results, nil
}
