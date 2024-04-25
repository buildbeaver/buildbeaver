package make_admin

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/cmd/bb-tools/cli"
	"github.com/buildbeaver/buildbeaver/server/cmd/bb-tools/commands"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/services/authorization"
	"github.com/buildbeaver/buildbeaver/server/services/group"
	"github.com/buildbeaver/buildbeaver/server/store"
	"github.com/buildbeaver/buildbeaver/server/store/grants"
	"github.com/buildbeaver/buildbeaver/server/store/group_memberships"
	"github.com/buildbeaver/buildbeaver/server/store/groups"
	"github.com/buildbeaver/buildbeaver/server/store/identities"
	"github.com/buildbeaver/buildbeaver/server/store/legal_entities"
	"github.com/buildbeaver/buildbeaver/server/store/ownerships"
)

const defaultSQLiteConnectionString = "file:/var/lib/buildbeaver/db/sqlite.db?cache=shared"

func init() {
	adminRootCmd.PersistentFlags().StringVar(
		&adminCmdConfig.databaseDriver,
		"driver",
		string(store.Sqlite),
		"The Database Driver to use for fetching and writing data (i.e sqlite3|postgres)")
	adminRootCmd.PersistentFlags().StringVar(
		&adminCmdConfig.databaseConnectionString,
		"connection",
		defaultSQLiteConnectionString,
		"The connection string for the database to use for fetching and writing data")
	adminRootCmd.PersistentFlags().BoolVarP(
		&adminCmdConfig.verbose,
		"verbose",
		"v",
		false,
		"Enable verbose log output")

	commands.RootCmd.AddCommand(adminRootCmd)
	adminRootCmd.AddCommand(adminGrantCmd)
	adminRootCmd.AddCommand(adminRevokeCmd)
}

var adminCmdConfig = struct {
	databaseConfig           store.DatabaseConfig
	databaseDriver           string
	databaseConnectionString string
	verbose                  bool
	logFactory               logger.LogFactory
	db                       *store.DB
	dbCleanup                func()
	legalEntityStore         store.LegalEntityStore
	identityStore            store.IdentityStore
	groupService             services.GroupService
}{}

var adminRootCmd = &cobra.Command{
	Use:   "admin grant|revoke",
	Short: "Perform operations on the 'admin' group for a particular company.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		adminCmdConfig.databaseConfig = store.DatabaseConfig{
			ConnectionString:   store.DatabaseConnectionString(adminCmdConfig.databaseConnectionString),
			Driver:             store.DBDriver(adminCmdConfig.databaseDriver),
			MaxIdleConnections: store.DefaultDatabaseMaxIdleConnections,
			MaxOpenConnections: store.DefaultDatabaseMaxOpenConnections,
		}

		// stores need a log factory; use a very plain log format
		logRegistry, err := logger.NewLogRegistry("")
		if err != nil {
			return err
		}
		logFactory := logger.MakeLogrusLogFactoryStdOutPlain(logRegistry)
		adminCmdConfig.logFactory = logFactory

		// open the database but do not perform migrations
		db, cleanup, err := store.NewDatabase(context.Background(), adminCmdConfig.databaseConfig, nil)
		if err != nil {
			return fmt.Errorf("error opening %s database: %w", adminCmdConfig.databaseConfig.Driver, err)
		}
		adminCmdConfig.db = db
		adminCmdConfig.dbCleanup = cleanup

		// make some stores and services we need for database access
		adminCmdConfig.legalEntityStore = legal_entities.NewStore(db, logFactory)
		adminCmdConfig.identityStore = identities.NewStore(db, logFactory)
		adminCmdConfig.groupService = group.NewGroupService(
			db,
			ownerships.NewStore(db, logFactory),
			groups.NewStore(db, logFactory),
			group_memberships.NewStore(db, logFactory),
			grants.NewStore(db, logFactory),
			authorization.NewNoOpAuthorizationService(logFactory),
			logFactory,
		)

		// we need the group service in order to ensure business logic is run when adding/remove user from group

		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if adminCmdConfig.dbCleanup != nil {
			adminCmdConfig.dbCleanup()
			adminCmdConfig.dbCleanup = nil
		}
	},
}

var adminGrantCmd = &cobra.Command{
	Use:           "grant user-name company-name",
	Short:         "Makes the specified user an administrator for a company, by making them a member of the 'admin' standard group.",
	Args:          cobra.ExactArgs(2),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		return adminCmdConfig.db.WithTx(ctx, nil, func(tx *store.Tx) error {
			user, userIdentity, company, adminGroup, err := parseArgsAndRead(ctx, tx, args)
			if err != nil {
				return err
			}

			// Try to read existing membership
			existingMembership, err := adminCmdConfig.groupService.ReadMembership(ctx, tx, adminGroup.ID, userIdentity.ID, models.BuildBeaverToolsSystem)
			if err != nil {
				if gerror.IsNotFound(err) {
					existingMembership = nil
				} else {
					return fmt.Errorf("error attempting to read existing group membership for %s group, user '%s', company '%s': %w", adminGroup.Name, user.Name, company.Name, err)
				}
			}

			if existingMembership == nil {
				// TODO: Who should this membership be added by? A special value for sysadmin?
				addedBy := company.ID

				// Make the user a member of the admin group, if they aren't already
				_, created, err := adminCmdConfig.groupService.FindOrCreateMembership(ctx, tx, models.NewGroupMembershipData(
					adminGroup.ID, userIdentity.ID, models.BuildBeaverToolsSystem, addedBy))
				if err != nil {
					return fmt.Errorf("error adding user '%s' to %s group for company '%s': %w", adminGroup.Name, user.Name, company.Name, err)
				}
				if created {
					cli.Stdout.Printf("Granted.\n")
				} else {
					cli.Stdout.Printf("Not granted: already a member through %s.\n", models.BuildBeaverToolsSystem)
				}
			} else {
				cli.Stdout.Printf("Not granted: already a member through %s.\n", models.BuildBeaverToolsSystem)
			}

			// Always output details of the latest admin permissions for the user
			listMembershipsForUser(ctx, tx, user, userIdentity, company, adminGroup)
			return nil
		})
	},
}

var adminRevokeCmd = &cobra.Command{
	Use:           "revoke user-name company-name",
	Short:         "Revokes the specified user's special administrator rights for a company, by removing their special membership of the 'admin' standard group. NOTE: The user may still be an admin if they have inherited rights from GitHub.",
	Args:          cobra.ExactArgs(2),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		return adminCmdConfig.db.WithTx(ctx, nil, func(tx *store.Tx) error {
			user, userIdentity, company, adminGroup, err := parseArgsAndRead(ctx, tx, args)
			if err != nil {
				return err
			}

			// Try to read existing membership
			existingMembership, err := adminCmdConfig.groupService.ReadMembership(ctx, tx, adminGroup.ID, userIdentity.ID, models.BuildBeaverToolsSystem)
			if err != nil {
				if gerror.IsNotFound(err) {
					existingMembership = nil
				} else {
					return fmt.Errorf("error attempting to read existing group membership for %s group, user '%s', company '%s': %w", adminGroup.Name, user.Name, company.Name, err)
				}
			}

			if existingMembership != nil {
				// Remove the user from the admin group
				system := models.BuildBeaverToolsSystem
				err = adminCmdConfig.groupService.RemoveMembership(ctx, tx, adminGroup.ID, userIdentity.ID, &system)
				if err != nil {
					return fmt.Errorf("error removing user '%s' from %s group for company '%s': %w", user.Name, adminGroup.Name, company.Name, err)
				}
				cli.Stdout.Printf("Revoked.\n")
			} else {
				cli.Stdout.Printf("Not revoked: not a member through %s.\n", models.BuildBeaverToolsSystem)
			}

			// Always output details of the latest admin permissions for the user
			listMembershipsForUser(ctx, tx, user, userIdentity, company, adminGroup)
			return nil
		})
	},
}

// parseArgsAndRead parses the supplied arguments expecting the name of a user and the name of a company.
// Reads and returns the user's Legal Entity and Identity, and the company's Legal Entity and admin group from the database.
func parseArgsAndRead(ctx context.Context, txOrNil *store.Tx, args []string) (user *models.LegalEntity, userIdentity *models.Identity, company *models.LegalEntity, adminGroup *models.Group, err error) {
	userName := args[0]
	if len(userName) == 0 {
		return nil, nil, nil, nil, fmt.Errorf("error: user's Legal Entity name must be specified")
	}
	companyName := args[1]
	if len(companyName) == 0 {
		return nil, nil, nil, nil, fmt.Errorf("error: company Legal Entity name must be specified")
	}

	user, err = adminCmdConfig.legalEntityStore.ReadByName(ctx, txOrNil, models.ResourceName(userName))
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("error: Unable to find user with name '%s': %w", userName, err)
	}
	if user.Type != models.LegalEntityTypePerson {
		return nil, nil, nil, nil, fmt.Errorf("error: The specified user must be of type '%s' (found '%s')", models.LegalEntityTypePerson, user.Type)
	}
	userIdentity, err = adminCmdConfig.identityStore.ReadByOwnerResource(ctx, txOrNil, user.ID.ResourceID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("error: Unable to find user identity for user name '%s': %w", userName, err)
	}

	company, err = adminCmdConfig.legalEntityStore.ReadByName(ctx, txOrNil, models.ResourceName(companyName))
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("error: Unable to find company with name '%s': %w", companyName, err)
	}
	if company.Type != models.LegalEntityTypeCompany {
		return nil, nil, nil, nil, fmt.Errorf("error: The specified company must be of type '%s' (found '%s')", models.LegalEntityTypeCompany, company.Type)
	}

	adminGroup, err = adminCmdConfig.groupService.ReadByName(ctx, txOrNil, company.ID, models.AdminStandardGroup.Name)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("error: Unable to find %s group for company '%s': %w", models.AdminStandardGroup.Name, company.Name, err)
	}

	return user, userIdentity, company, adminGroup, nil
}

func listMembershipsForUser(ctx context.Context, txOrNil *store.Tx, user *models.LegalEntity, userIdentity *models.Identity, company *models.LegalEntity, adminGroup *models.Group) {
	db := adminCmdConfig.db

	db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		count := 0
		pagination := models.NewPagination(models.DefaultPaginationLimit, nil)
		for moreResults := true; moreResults; {
			groupMemberships, cursor, err := adminCmdConfig.groupService.ListGroupMemberships(ctx, tx, &adminGroup.ID, &userIdentity.ID, nil, pagination)
			if err != nil {
				return err
			}
			if len(groupMemberships) > 0 && count == 0 {
				cli.Stdout.Printf("User %s is a member of %s group for company '%s' through the following membership(s):\n",
					user.Name, adminGroup.Name, company.Name)
			}
			for _, groupMembership := range groupMemberships {
				count++
				cli.Stdout.Printf("    Membership %d source system '%s': ID %s\n", count, groupMembership.SourceSystem, groupMembership.ID)
			}
			if cursor != nil && cursor.Next != nil {
				pagination.Cursor = cursor.Next // move on to next page of results
			} else {
				moreResults = false
			}
		}
		if count == 0 {
			cli.Stdout.Printf("User %s is NOT a member of %s group for company '%s'\n",
				user.Name, adminGroup.Name, company.Name)
		}
		return nil
	})

}
