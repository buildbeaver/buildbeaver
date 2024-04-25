package fake_scm

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/store"
)

const (
	FakeSCMName = models.SystemName("fake-scm")
)

type FakeSCMAuthentication struct {
	userID UserID
}

func (m *FakeSCMAuthentication) Name() models.SystemName {
	return FakeSCMName
}

type UserID int64
type CompanyID int64
type RepoID int64
type GroupID int64

// fakeSCMState contains the information stored in the SCM. The data structure is loosely modelled on GitHub.
// This information can be set for testing purposes and returned through the standard SCM interface.
type fakeSCMState struct {
	users         map[UserID]*fakeSCMUser
	companies     map[CompanyID]*fakeSCMCompany
	nextUserID    UserID
	nextCompanyID CompanyID
	nextRepoID    RepoID
	nextGroupID   GroupID
}

type fakeSCMUser struct {
	id               UserID
	legalEntity      *models.LegalEntityData
	usingBuildBeaver bool
	repos            map[RepoID]*fakeSCMRepo
}

type fakeSCMCompany struct {
	id               CompanyID
	legalEntity      *models.LegalEntityData
	usingBuildBeaver bool
	repos            map[RepoID]*fakeSCMRepo
	members          map[UserID]bool
	groups           map[GroupID]*fakeSCMGroup
	groupsByName     map[models.ResourceName]*fakeSCMGroup // same objects as the 'groups' map but indexed by name
}

type fakeSCMGroup struct {
	id          GroupID
	name        models.ResourceName
	isCustom    bool // true for custom group, false for standard group
	members     map[UserID]bool
	permissions map[RepoID]fakeSCMRepoPermission
}

type fakeSCMRepoPermission struct {
	read  bool
	write bool
	admin bool
}

type fakeSCMRepo struct {
	id           RepoID
	name         models.ResourceName
	sshPublicKey []byte
}

// FakeSCMService is an implementation of the SCM interface designed for testing. It is loosely based on GitHub,
// but allows complete control of the data on the SCM.
type FakeSCMService struct {
	db                 *store.DB
	repoStore          store.RepoStore
	commitStore        store.CommitStore
	legalEntityService services.LegalEntityService
	state              *fakeSCMState
	logger.Log
}

func NewFakeSCMService(
	db *store.DB,
	repoStore store.RepoStore,
	commitStore store.CommitStore,
	legalEntityService services.LegalEntityService,
	logFactory logger.LogFactory,
) *FakeSCMService {
	return &FakeSCMService{
		db:                 db,
		repoStore:          repoStore,
		commitStore:        commitStore,
		legalEntityService: legalEntityService,
		state: &fakeSCMState{
			users:         make(map[UserID]*fakeSCMUser),
			companies:     make(map[CompanyID]*fakeSCMCompany),
			nextUserID:    1,
			nextCompanyID: 1,
			nextRepoID:    1,
			nextGroupID:   1,
		},
		Log: logFactory("FakeSCMService"),
	}
}

func (s *FakeSCMService) CreateUser(userName string, usingBuildBeaver bool) (UserID, models.ExternalResourceID) {
	userID := s.state.nextUserID
	s.state.nextUserID++

	externalID := userIDToExternalResourceID(userID)
	newUser := &fakeSCMUser{
		id: userID,
		legalEntity: models.NewPersonLegalEntityData(
			fakeLegalEntityName(userName),
			userName,
			"test-email@test.domain.com",
			&externalID,
			"",
		),
		usingBuildBeaver: usingBuildBeaver,
		repos:            make(map[RepoID]*fakeSCMRepo),
	}
	s.state.users[userID] = newUser
	return userID, externalID
}

func (s *FakeSCMService) CreateAuthForUser(userID UserID) (models.SCMAuth, error) {
	if _, ok := s.state.users[userID]; !ok {
		return nil, fmt.Errorf("error: user ID %d not found", userID)
	}
	auth := &FakeSCMAuthentication{
		userID: userID,
	}
	return auth, nil
}

// authenticateUser returns the fake SCM user authenticated with auth.
// NOTE: The returned pointer points to the 'live' user object within the fake SCM state, so any changes are stored.
func (s *FakeSCMService) authenticateUser(auth models.SCMAuth) (*fakeSCMUser, error) {
	fakeAuth, ok := auth.(*FakeSCMAuthentication)
	if !ok {
		return nil, fmt.Errorf("unrecognized auth type: %T", auth)
	}
	user, ok := s.state.users[fakeAuth.userID]
	if !ok {
		return nil, fmt.Errorf("error: user ID %d (from auth token) not found", fakeAuth.userID)
	}
	return user, nil
}

func (s *FakeSCMService) CreateCompany(companyName string) (CompanyID, models.ExternalResourceID) {
	companyID := s.state.nextCompanyID
	s.state.nextCompanyID++
	externalID := companyIDToExternalResourceID(companyID)
	newCompany := &fakeSCMCompany{
		id: companyID,
		legalEntity: models.NewCompanyLegalEntityData(
			fakeLegalEntityName(companyName),
			companyName,
			"admin@test-company.domain.com",
			&externalID,
			"",
		),
		usingBuildBeaver: true, // TODO: Allow this to be passed in
		repos:            make(map[RepoID]*fakeSCMRepo),
		members:          make(map[UserID]bool),
		groups:           make(map[GroupID]*fakeSCMGroup),
		groupsByName:     make(map[models.ResourceName]*fakeSCMGroup),
	}
	s.state.companies[companyID] = newCompany

	// Add standard groups to company; do this after company is added to state
	s.createGroupForCompany(companyID, models.AdminStandardGroup.Name, false)
	s.createGroupForCompany(companyID, models.ReadOnlyUserStandardGroup.Name, false)
	s.createGroupForCompany(companyID, models.UserStandardGroup.Name, false)
	return companyID, externalID
}

func (s *FakeSCMService) CreateRepoForUser(userID UserID, repoName string) (RepoID, models.ExternalResourceID, error) {
	user, ok := s.state.users[userID]
	if !ok {
		return 0, models.ExternalResourceID{}, fmt.Errorf("error: user ID %d not found", userID)
	}
	repoID := s.state.nextRepoID
	s.state.nextRepoID++
	repoResourceName := fakeRepoName(repoName)
	newRepo := &fakeSCMRepo{
		id:   repoID,
		name: repoResourceName,
	}
	user.repos[repoID] = newRepo
	return repoID, repoIDToExternalResourceID(repoID), nil
}

func (s *FakeSCMService) CreateRepoForCompany(companyID CompanyID, repoName string) (RepoID, models.ExternalResourceID, error) {
	company, ok := s.state.companies[companyID]
	if !ok {
		return 0, models.ExternalResourceID{}, fmt.Errorf("error: company ID %d not found", companyID)
	}
	repoID := s.state.nextRepoID
	s.state.nextRepoID++
	repoResourceName := fakeRepoName(repoName)
	newRepo := &fakeSCMRepo{
		id:           repoID,
		name:         repoResourceName,
		sshPublicKey: nil,
	}
	company.repos[repoID] = newRepo
	return repoID, repoIDToExternalResourceID(repoID), nil
}

// DeleteRepo delete the repo with the specified ID, from whichever user or company it was created under.
// This method is idempotent so it doesn't need to return an error.
func (s *FakeSCMService) DeleteRepo(repoID RepoID) {
	for _, user := range s.state.users {
		delete(user.repos, repoID)
	}
	for _, company := range s.state.companies {
		delete(company.repos, repoID)
	}
}

// CreateCustomGroupForCompany adds a new custom access control group for the company.
// Make sure the groupName does not conflict with the name of any standard group - for this reason
// it is recommended to prefix the group name with "fakescm-".
func (s *FakeSCMService) CreateCustomGroupForCompany(companyID CompanyID, groupName string) (GroupID, error) {
	return s.createGroupForCompany(companyID, fakeGroupName(groupName), true)
}

func (s *FakeSCMService) createGroupForCompany(companyID CompanyID, groupName models.ResourceName, groupIsCustom bool) (GroupID, error) {
	company, ok := s.state.companies[companyID]
	if !ok {
		return 0, fmt.Errorf("error: company ID %d not found", companyID)
	}
	groupID := s.state.nextGroupID
	s.state.nextGroupID++
	newGroup := &fakeSCMGroup{
		id:          groupID,
		name:        groupName,
		isCustom:    groupIsCustom,
		members:     make(map[UserID]bool),
		permissions: make(map[RepoID]fakeSCMRepoPermission),
	}
	company.groups[groupID] = newGroup
	company.groupsByName[groupName] = newGroup
	return groupID, nil
}

// DeleteCustomGroupForCompany removes a custom access control group for a company.
// Make sure the groupName does not conflict with the name of any standard group - for this reason
// it is recommended to prefix the group name with "fakescm-".
func (s *FakeSCMService) DeleteCustomGroupForCompany(companyID CompanyID, groupName models.ResourceName) error {
	company, ok := s.state.companies[companyID]
	if !ok {
		return fmt.Errorf("error: company ID %d not found", companyID)
	}
	group, ok := company.groupsByName[groupName]
	if !ok {
		return fmt.Errorf("error: group %q not found for company ID %d", groupName, companyID)
	}

	delete(company.groups, group.id)
	delete(company.groupsByName, groupName)
	return nil
}

func (s *FakeSCMService) AddUserToCompany(companyID CompanyID, userID UserID) error {
	company, ok := s.state.companies[companyID]
	if !ok {
		return fmt.Errorf("error: company ID %d not found", companyID)
	}
	if _, ok = s.state.users[userID]; !ok {
		return fmt.Errorf("error: user ID %d not found", userID)
	}
	company.members[userID] = true
	return nil
}

func (s *FakeSCMService) RemoveUserFromCompany(companyID CompanyID, userID UserID) error {
	company, ok := s.state.companies[companyID]
	if !ok {
		return fmt.Errorf("error: company ID %d not found", companyID)
	}
	if _, ok = s.state.users[userID]; !ok {
		return fmt.Errorf("error: user ID %d not found", userID)
	}
	delete(company.members, userID)

	// Also remove the user from any groups within the company; these are independent of membership of the company itself
	for groupName, _ := range company.groupsByName {
		s.RemoveUserFromGroup(companyID, groupName, userID) // this is a no-op if user isn't in the group
	}

	return nil
}

func (s *FakeSCMService) AddUserToGroup(companyID CompanyID, groupName models.ResourceName, userID UserID) error {
	company, ok := s.state.companies[companyID]
	if !ok {
		return fmt.Errorf("error: company ID %d not found", companyID)
	}
	group, ok := company.groupsByName[groupName]
	if !ok {
		return fmt.Errorf("error: group %q not found for company ID %d", groupName, companyID)
	}
	if _, ok = s.state.users[userID]; !ok {
		return fmt.Errorf("error: user ID %d not found", userID)
	}
	group.members[userID] = true
	return nil
}

func (s *FakeSCMService) RemoveUserFromGroup(companyID CompanyID, groupName models.ResourceName, userID UserID) error {
	company, ok := s.state.companies[companyID]
	if !ok {
		return fmt.Errorf("error: company ID %d not found", companyID)
	}
	group, ok := company.groupsByName[groupName]
	if !ok {
		return fmt.Errorf("error: group %q not found for company ID %d", groupName, companyID)
	}
	if _, ok = s.state.users[userID]; !ok {
		return fmt.Errorf("error: user ID %d not found", userID)
	}
	delete(group.members, userID)
	return nil
}

func (s *FakeSCMService) SetGroupPermissionForRepo(
	companyID CompanyID,
	groupName models.ResourceName,
	repoID RepoID,
	canRead bool,
	canWrite bool,
	isAdmin bool,
) error {
	company, ok := s.state.companies[companyID]
	if !ok {
		return fmt.Errorf("error: company ID %d not found", companyID)
	}
	group, ok := company.groupsByName[groupName]
	if !ok {
		return fmt.Errorf("error: group %q not found for company ID %d", groupName, companyID)
	}
	_, ok = company.repos[repoID]
	if !ok {
		return fmt.Errorf("error: repo %d not found for company ID %d", repoID, companyID)
	}

	group.permissions[repoID] = fakeSCMRepoPermission{
		read:  canRead,
		write: canWrite,
		admin: isAdmin,
	}
	return nil
}

const userIDPrefix = "user-"

const companyIDPrefix = "company-"

const repoIDPrefix = "repo-"

const groupIDPrefix = "group-"

func userIDToExternalID(id UserID) string {
	return fmt.Sprintf("%s%d", userIDPrefix, id)
}

func companyIDToExternalID(id CompanyID) string {
	return fmt.Sprintf("%s%d", companyIDPrefix, id)
}

func repoIDToExternalID(id RepoID) string {
	return fmt.Sprintf("%s%d", repoIDPrefix, id)
}

func groupIDToExternalID(id GroupID) string {
	return fmt.Sprintf("%s%d", groupIDPrefix, id)
}

func userIDToExternalResourceID(id UserID) models.ExternalResourceID {
	return models.NewExternalResourceID(FakeSCMName, userIDToExternalID(id))
}

func companyIDToExternalResourceID(id CompanyID) models.ExternalResourceID {
	return models.NewExternalResourceID(FakeSCMName, companyIDToExternalID(id))
}

func repoIDToExternalResourceID(id RepoID) models.ExternalResourceID {
	return models.NewExternalResourceID(FakeSCMName, repoIDToExternalID(id))
}

func groupIDToExternalResourceID(id GroupID) models.ExternalResourceID {
	return models.NewExternalResourceID(FakeSCMName, groupIDToExternalID(id))
}

func userIDFromExternalResourceID(externalID *models.ExternalResourceID) (UserID, error) {
	if externalID == nil {
		return 0, fmt.Errorf("error: empty external ID supplied")
	}
	if externalID.ExternalSystem != FakeSCMName {
		return 0, fmt.Errorf("error: unexpected system name: %v", externalID.ExternalSystem)
	}
	if !strings.HasPrefix(externalID.ResourceID, userIDPrefix) {
		return 0, fmt.Errorf("error: not a user ID")
	}
	idStr := strings.TrimPrefix(externalID.ResourceID, userIDPrefix)
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing user external ID: %w", err)
	}
	return UserID(id), nil
}

func companyIDFromExternalResourceID(externalID *models.ExternalResourceID) (CompanyID, error) {
	if externalID == nil {
		return 0, fmt.Errorf("error: empty external ID supplied")
	}
	if externalID.ExternalSystem != FakeSCMName {
		return 0, fmt.Errorf("error: unexpected system name: %v", externalID.ExternalSystem)
	}
	if !strings.HasPrefix(externalID.ResourceID, companyIDPrefix) {
		return 0, fmt.Errorf("error: not a company ID")
	}
	idStr := strings.TrimPrefix(externalID.ResourceID, companyIDPrefix)
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing company external ID: %w", err)
	}
	return CompanyID(id), nil
}

func repoIDFromExternalResourceID(externalID *models.ExternalResourceID) (RepoID, error) {
	if externalID == nil {
		return 0, fmt.Errorf("error: empty external ID supplied")
	}
	if externalID.ExternalSystem != FakeSCMName {
		return 0, fmt.Errorf("error: unexpected system name: %v", externalID.ExternalSystem)
	}
	if !strings.HasPrefix(externalID.ResourceID, repoIDPrefix) {
		return 0, fmt.Errorf("error: not a repo ID")
	}
	idStr := strings.TrimPrefix(externalID.ResourceID, repoIDPrefix)
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing repo external ID: %w", err)
	}
	return RepoID(id), nil
}

func groupIDFromExternalResourceID(externalID *models.ExternalResourceID) (GroupID, error) {
	if externalID == nil {
		return 0, fmt.Errorf("error: empty external ID supplied")
	}
	if externalID.ExternalSystem != FakeSCMName {
		return 0, fmt.Errorf("error: unexpected system name: %v", externalID.ExternalSystem)
	}
	if !strings.HasPrefix(externalID.ResourceID, groupIDPrefix) {
		return 0, fmt.Errorf("error: not a group ID")
	}
	idStr := strings.TrimPrefix(externalID.ResourceID, groupIDPrefix)
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing group external ID: %w", err)
	}
	return GroupID(id), nil
}

func fakeLegalEntityName(name string) models.ResourceName {
	return models.ResourceName(strings.ToLower(name))
}

func fakeRepoName(name string) models.ResourceName {
	return models.ResourceName(strings.ToLower(name))
}

func fakeGroupName(name string) models.ResourceName {
	return models.ResourceName(strings.ToLower(name))
}

func (s *FakeSCMService) findRepo(repoID RepoID) (*fakeSCMRepo, error) {
	for _, user := range s.state.users {
		repo, found := user.repos[repoID]
		if found {
			return repo, nil
		}
	}
	for _, company := range s.state.companies {
		repo, found := company.repos[repoID]
		if found {
			return repo, nil
		}
	}
	return nil, gerror.NewErrNotFound(fmt.Sprintf("error: repo not found, ID %d", repoID))
}

func (s *FakeSCMService) findRepoByExternalID(repoExternalID *models.ExternalResourceID) (*fakeSCMRepo, error) {
	repoID, err := repoIDFromExternalResourceID(repoExternalID)
	if err != nil {
		return nil, err
	}
	return s.findRepo(repoID)
}

// Name returns the unique name of the SCM.
func (s *FakeSCMService) Name() models.SystemName {
	return FakeSCMName
}

// WebhookHandler returns the http handler func that should be invoked when
// the fake SCM service receives a webhook, or an error if the service does not
// support webhooks.
func (s *FakeSCMService) WebhookHandler() (http.HandlerFunc, error) {
	return nil, fmt.Errorf("FakeSCMService does not support Webhooks")
}

// NotifyBuildUpdated is called when the status of a build is updated.
// Allows the SCM to notify users or take other actions when a build has progressed or finished.
func (s *FakeSCMService) NotifyBuildUpdated(ctx context.Context, txOrNil *store.Tx, build *models.Build, repo *models.Repo) error {
	// Verify the repo is actually on the fake SCM
	fakeSCMRepo, err := s.findRepoByExternalID(repo.ExternalID)
	if err != nil {
		return err
	}

	s.Tracef("Received notification that build %q has been updated for repo repo %d, name %q (database repo %q ID %d)",
		build.Name, fakeSCMRepo.id, fakeSCMRepo.name, repo.Name, repo.ID)
	return nil // This is a no-op
}

// EnableRepo is called when a repo is enabled within BuildBeaver - this is the SCM's opportunity
// to do any setup required to close the loop and make this work. Public key identifies the key that
// BuildBeaver will use when cloning the repo.
func (s *FakeSCMService) EnableRepo(ctx context.Context, repo *models.Repo, publicKey []byte) error {
	// Verify the repo is actually on the fake SCM
	fakeSCMRepo, err := s.findRepoByExternalID(repo.ExternalID)
	if err != nil {
		return err
	}
	if len(publicKey) == 0 {
		return fmt.Errorf("error: public key not provided when enabling repo %d, name %q", fakeSCMRepo.id, fakeSCMRepo.name)
	}

	s.Tracef("Enabling repo %d, name %q", fakeSCMRepo.id, fakeSCMRepo.name)
	fakeSCMRepo.sshPublicKey = publicKey
	return nil
}

// DisableRepo is called when a repo is disabled in BuildBeaver - this is the SCM's opportunity to do any required
// teardown such as deleting webhooks or deployment keys etc.
func (s *FakeSCMService) DisableRepo(ctx context.Context, repo *models.Repo) error {
	// Verify the repo is actually on the fake SCM
	fakeSCMRepo, err := s.findRepoByExternalID(repo.ExternalID)
	if err != nil {
		return err
	}

	s.Tracef("Disabling repo %d, name %q", fakeSCMRepo.id, fakeSCMRepo.name)
	fakeSCMRepo.sshPublicKey = nil
	return nil
}

// BuildRepoLatestCommit will kick off a new build for the latest commit for a ref, if required.
func (s *FakeSCMService) BuildRepoLatestCommit(ctx context.Context, repo *models.Repo, ref string) error {
	// Verify the repo is actually on the fake SCM
	fakeSCMRepo, err := s.findRepoByExternalID(repo.ExternalID)
	if err != nil {
		return err
	}

	s.Tracef("Received call to BuildRepoLatestCommit() for repo repo %d, name %q (database repo %q ID %d) - no actual build will be queued",
		fakeSCMRepo.id, fakeSCMRepo.name, repo.Name, repo.ID)
	return nil // This is a no-op
}

// GetUserLegalEntityData returns an SCM legal entity representing the user currently authenticated with auth.
func (s *FakeSCMService) GetUserLegalEntityData(ctx context.Context, auth models.SCMAuth) (*models.LegalEntityData, error) {
	user, err := s.authenticateUser(auth)
	if err != nil {
		return nil, err
	}
	// Make a copy so the caller can't directly change the state of the SCM
	var legalEntityCopy = *user.legalEntity
	return &legalEntityCopy, nil
}

// IsLegalEntityRegisteredAsUser returns true if the specified Legal Entity is registered as a user of this
// build system on this SCM. The meaning of 'registered as a user' is dependent on the SCM.
func (s *FakeSCMService) IsLegalEntityRegisteredAsUser(ctx context.Context, legalEntity *models.LegalEntity) (bool, error) {
	userID, err := userIDFromExternalResourceID(legalEntity.ExternalID)
	if err == nil {
		user, ok := s.state.users[userID]
		if !ok {
			return false, fmt.Errorf("error: user ID %d not found", userID)
		}
		return user.usingBuildBeaver, nil
	} else {
		// External ID not a user ID; try a company ID
		companyID, err := companyIDFromExternalResourceID(legalEntity.ExternalID)
		if err != nil {
			return false, fmt.Errorf("error: unknown external ID %q in legal entity", legalEntity.ExternalID.String())
		}
		company, ok := s.state.companies[companyID]
		if !ok {
			return false, fmt.Errorf("error: company ID %d not found", companyID)
		}
		return company.usingBuildBeaver, nil
	}
}

// ListLegalEntitiesRegisteredAsUsers lists all Legal Entities from the SCM that are registered as using this
// build system for any of their repos.
func (s *FakeSCMService) ListLegalEntitiesRegisteredAsUsers(ctx context.Context) ([]*models.LegalEntityData, error) {
	var legalEntities []*models.LegalEntityData

	// Add all users that are using BuildBeaver
	for _, user := range s.state.users {
		if user.usingBuildBeaver {
			orgCopy := *user.legalEntity
			legalEntities = append(legalEntities, &orgCopy)
		}
	}

	// Add all companies that are using BuildBeaver
	for _, company := range s.state.companies {
		if company.usingBuildBeaver {
			orgCopy := *company.legalEntity
			legalEntities = append(legalEntities, &orgCopy)
		}
	}
	return legalEntities, nil
}

// ListReposRegisteredForLegalEntity lists all repos belonging to a legal entity that are registered as using
// the build system.
func (s *FakeSCMService) ListReposRegisteredForLegalEntity(ctx context.Context, legalEntity *models.LegalEntity) ([]*models.Repo, error) {
	var repoMap map[RepoID]*fakeSCMRepo
	userID, err := userIDFromExternalResourceID(legalEntity.ExternalID)
	if err == nil {
		user, ok := s.state.users[userID]
		if !ok {
			return nil, fmt.Errorf("error: user ID %d not found", userID)
		}
		repoMap = user.repos
	} else {
		// External ID not a user ID; try a company ID
		companyID, err := companyIDFromExternalResourceID(legalEntity.ExternalID)
		if err != nil {
			return nil, fmt.Errorf("error: unknown external ID %q in legal entity", legalEntity.ExternalID.String())
		}
		company, ok := s.state.companies[companyID]
		if !ok {
			return nil, fmt.Errorf("error: company ID %d not found", companyID)
		}
		repoMap = company.repos
	}

	var repos []*models.Repo
	var now = models.NewTime(time.Now())
	for _, fakeRepo := range repoMap {
		// TODO: Check to see whether the repo should be accessible to BuildBeaver

		externalResourceID := repoIDToExternalResourceID(fakeRepo.id)
		repo := models.NewRepo(
			now,
			fakeRepo.name,
			legalEntity.ID,
			fmt.Sprintf("This is fake repo %d", fakeRepo.id),
			"ssh://fake-ssh-url",
			"https://fake-clone-url",
			"https://fake-html-url",
			"main",
			false,
			true,
			nil,
			&externalResourceID,
			"",
		)
		repos = append(repos, repo)
	}
	return repos, nil
}

// ListAllCompanyMembers returns a list of all users who are members of the specified company.
func (s *FakeSCMService) ListAllCompanyMembers(ctx context.Context, company *models.LegalEntity) ([]*models.LegalEntityData, error) {
	// External ID for Legal Entity must be a company ID
	companyID, err := companyIDFromExternalResourceID(company.ExternalID)
	if err != nil {
		return nil, fmt.Errorf("error: unknown external ID %q for company legal entity", company.ExternalID.String())
	}
	companyData, ok := s.state.companies[companyID]
	if !ok {
		return nil, fmt.Errorf("error: company ID %d not found", companyID)
	}

	// Return the legal entities for each user who is a member of the company
	var members []*models.LegalEntityData
	for memberUserID, _ := range companyData.members {
		user, ok := s.state.users[memberUserID]
		if !ok {
			return nil, fmt.Errorf("error: member user ID %d not found", memberUserID)
		}
		members = append(members, user.legalEntity)
	}
	return members, nil
}

// ListCompanyCustomGroups returns a list of custom groups that can be used for access control for a company.
// These can corresponding to teams, roles, or any other way to grant access to a group of people.
// These groups will be created in addition to the standard groups that are created for each company.
func (s *FakeSCMService) ListCompanyCustomGroups(ctx context.Context, company *models.LegalEntity) ([]*models.Group, error) {
	// External ID for Legal Entity must be a company ID
	companyID, err := companyIDFromExternalResourceID(company.ExternalID)
	if err != nil {
		return nil, fmt.Errorf("error: unknown external ID %q for company legal entity", company.ExternalID.String())
	}
	companyData, ok := s.state.companies[companyID]
	if !ok {
		return nil, fmt.Errorf("error: company ID %d not found", companyID)
	}

	// Return the group names for each group in the company
	var groups []*models.Group
	now := models.NewTime(time.Now())
	for _, fakeGroup := range companyData.groups {
		if fakeGroup.isCustom {
			externalResourceID := groupIDToExternalResourceID(fakeGroup.id)
			newGroup := models.NewGroup(
				now,
				company.ID,
				fakeGroup.name,
				"A test SCM group",
				false,
				&externalResourceID)
			groups = append(groups, newGroup)
		}
	}
	return groups, nil
}

// ListCompanyGroupMembers returns a list of users who are members of the specified group within the specified
// company. The group is either a standard or a custom group that corresponds to the group of users
// (e.g a role or team) in the SCM.
func (s *FakeSCMService) ListCompanyGroupMembers(ctx context.Context, company *models.LegalEntity, group *models.Group) ([]*models.LegalEntityData, error) {
	// External ID for Legal Entity must be a company ID
	companyID, err := companyIDFromExternalResourceID(company.ExternalID)
	if err != nil {
		return nil, fmt.Errorf("error: unknown external ID %q for company legal entity", company.ExternalID.String())
	}
	companyData, ok := s.state.companies[companyID]
	if !ok {
		return nil, fmt.Errorf("error: company ID %d not found", companyID)
	}

	fakeGroup, ok := companyData.groupsByName[group.Name]
	if !ok {
		return nil, fmt.Errorf("error: group '%s' not found for company ID %d", group.Name, companyID)
	}

	// Return the legal entities for each user who is a member of the group
	var members []*models.LegalEntityData
	for memberUserID, _ := range fakeGroup.members {
		user, ok := s.state.users[memberUserID]
		if !ok {
			return nil, fmt.Errorf("error: group member user ID %d not found", memberUserID)
		}
		members = append(members, user.legalEntity)
	}
	return members, nil
}

// ListCompanyCustomGroupPermissions returns a list of permissions that a custom group for a company on the SCM
// should have. The group must be a custom group that corresponds to a group of users (e.g. team) in the SCM.
func (s *FakeSCMService) ListCompanyCustomGroupPermissions(ctx context.Context, company *models.LegalEntity, group *models.Group) ([]*models.Grant, error) {
	// External ID for Legal Entity must be a company ID
	companyID, err := companyIDFromExternalResourceID(company.ExternalID)
	if err != nil {
		return nil, fmt.Errorf("error: unknown external ID %q for company legal entity", company.ExternalID.String())
	}
	companyData, ok := s.state.companies[companyID]
	if !ok {
		return nil, fmt.Errorf("error: company ID %d not found", companyID)
	}

	fakeGroup, ok := companyData.groupsByName[group.Name]
	if !ok {
		return nil, fmt.Errorf("error: group '%s' not found for company ID %d", group.Name, companyID)
	}

	// Return grants for the permissions of the group
	var grants []*models.Grant
	now := models.NewTime(time.Now())
	grantedBy := company.ID // let's say the permissions are granted by the company legal entity
	for fakeRepoID, permission := range fakeGroup.permissions {
		// Find the repo in the BuildBeaver database
		repoExternalResourceID := repoIDToExternalResourceID(fakeRepoID)
		repo, err := s.repoStore.ReadByExternalID(ctx, nil, repoExternalResourceID)
		if err != nil {
			if gerror.ToNotFound(err) != nil {
				s.Infof("repo with ID '%s' that legal entity %s group %s has access to was not found in database; skipping granting permission",
					fakeRepoID, company.Name, group.Name)
				continue
			} else {
				return nil, errors.Wrap(err, "error looking for repo in the database")
			}
		}

		operations := findOperationsForFakeSCMRepoPermission(permission)
		for _, operation := range operations {
			grant := models.NewGroupGrant(now, grantedBy, group.ID, *operation, repo.ID.ResourceID)
			grants = append(grants, grant)
		}
	}

	return grants, nil
}

// findOperationsForFakeSCMRepoPermission finds a set of BuildBeaver access control operations that correspond to
// a permission object for a fake SCM repo.
func findOperationsForFakeSCMRepoPermission(permission fakeSCMRepoPermission) []*models.Operation {
	// Combine the operations granted by each field within the permission into a set
	opSet := make(map[*models.Operation]bool)
	if permission.read {
		mergeOperations(opSet, []*models.Operation{
			models.RepoReadOperation,
			models.BuildReadOperation,
			models.ArtifactReadOperation,
		})
	}
	if permission.write {
		mergeOperations(opSet, []*models.Operation{
			models.RepoReadOperation,
			models.RepoUpdateOperation,
			models.BuildCreateOperation,
			models.SecretCreateOperation,
			models.BuildReadOperation,
			models.ArtifactReadOperation,
			models.ArtifactDeleteOperation,
		})
	}
	if permission.admin {
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
	}
	// Convert the map (i.e. set) back to a list to return
	results := make([]*models.Operation, 0, len(opSet))
	for op, _ := range opSet {
		results = append(results, op)
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
