package dump

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/cmd/bb-tools/cli"
	"github.com/buildbeaver/buildbeaver/server/cmd/bb-tools/commands"
	"github.com/buildbeaver/buildbeaver/server/store"
	"github.com/buildbeaver/buildbeaver/server/store/grants"
	"github.com/buildbeaver/buildbeaver/server/store/group_memberships"
	"github.com/buildbeaver/buildbeaver/server/store/groups"
	"github.com/buildbeaver/buildbeaver/server/store/identities"
	"github.com/buildbeaver/buildbeaver/server/store/legal_entities"
)

const defaultSQLiteConnectionString = "file:/var/lib/buildbeaver/db/sqlite.db?cache=shared"

func init() {
	dumpRootCmd.PersistentFlags().StringVar(
		&dumpCmdConfig.databaseDriver,
		"driver",
		string(store.Sqlite),
		"The Database Driver to use for fetching data (i.e sqlite3|postgres)")
	dumpRootCmd.PersistentFlags().StringVar(
		&dumpCmdConfig.databaseConnectionString,
		"connection",
		defaultSQLiteConnectionString,
		"The connection string for the database to use for fetching data")
	dumpRootCmd.PersistentFlags().BoolVarP(
		&dumpCmdConfig.verbose,
		"verbose",
		"v",
		false,
		"Enable verbose log output")

	commands.RootCmd.AddCommand(dumpRootCmd)
	dumpRootCmd.AddCommand(dumpAllLegalEntitiesCmd)
	dumpRootCmd.AddCommand(dumpLegalEntityCmd)
	dumpRootCmd.AddCommand(dumpLegalEntityGroupsCmd)
}

var dumpCmdConfig = struct {
	databaseConfig           store.DatabaseConfig
	databaseDriver           string
	databaseConnectionString string
	verbose                  bool
	logFactory               logger.LogFactory
	db                       *store.DB
	dbCleanup                func()
	legalEntityStore         store.LegalEntityStore
	groupStore               store.GroupStore
	groupMembershipStore     store.GroupMembershipStore
	grantStore               store.GrantStore
	identityStore            store.IdentityStore
}{}

var dumpRootCmd = &cobra.Command{
	Use:   "dump (command)",
	Short: "Dumps the data from of all objects of the specified type from the database",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		dumpCmdConfig.databaseConfig = store.DatabaseConfig{
			ConnectionString:   store.DatabaseConnectionString(dumpCmdConfig.databaseConnectionString),
			Driver:             store.DBDriver(dumpCmdConfig.databaseDriver),
			MaxIdleConnections: store.DefaultDatabaseMaxIdleConnections,
			MaxOpenConnections: store.DefaultDatabaseMaxOpenConnections,
		}

		// stores need a log factory; use a very plain log format
		logRegistry, err := logger.NewLogRegistry("")
		if err != nil {
			return err
		}
		logFactory := logger.MakeLogrusLogFactoryStdOutPlain(logRegistry)
		dumpCmdConfig.logFactory = logFactory

		// open the database but do not perform migrations
		db, cleanup, err := store.NewDatabase(context.Background(), dumpCmdConfig.databaseConfig, nil)
		if err != nil {
			return fmt.Errorf("error opening %s database for dump: %w", dumpCmdConfig.databaseConfig.Driver, err)
		}
		dumpCmdConfig.db = db
		dumpCmdConfig.dbCleanup = cleanup

		// make some stores we might need for dumping database data
		dumpCmdConfig.legalEntityStore = legal_entities.NewStore(db, logFactory)
		dumpCmdConfig.groupStore = groups.NewStore(db, logFactory)
		dumpCmdConfig.groupMembershipStore = group_memberships.NewStore(db, logFactory)
		dumpCmdConfig.grantStore = grants.NewStore(db, logFactory)
		dumpCmdConfig.identityStore = identities.NewStore(db, logFactory)

		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if dumpCmdConfig.dbCleanup != nil {
			dumpCmdConfig.dbCleanup()
			dumpCmdConfig.dbCleanup = nil
		}
	},
}

var dumpAllLegalEntitiesCmd = &cobra.Command{
	Use:           "all-legal-entities",
	Short:         "Dumps a list of all legal entities in the database",
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		return dumpCmdConfig.db.WithTx(ctx, nil, func(tx *store.Tx) error {
			cli.Stdout.Printf("\nALL LEGAL ENTITIES\n\n")
			count := 0
			pagination := models.NewPagination(models.DefaultPaginationLimit, nil)
			for moreResults := true; moreResults; {
				legalEntities, cursor, err := dumpCmdConfig.legalEntityStore.ListAllLegalEntities(ctx, tx, pagination)
				if err != nil {
					return fmt.Errorf("error reading list of all Legal Entities: %w", err)
				}
				for _, entity := range legalEntities {
					count++
					cli.Stdout.Printf("%d: Name '%s', type '%s', ID '%s':\n", count, entity.Name, entity.Type.String(), entity.ID)
				}
				if cursor != nil && cursor.Next != nil {
					pagination.Cursor = cursor.Next // move on to next page of results
				} else {
					moreResults = false
				}
			}
			cli.Stdout.Printf("\n")
			return nil
		})
	},
}

var dumpLegalEntityCmd = &cobra.Command{
	Use:           "legal-entity name",
	Short:         "Dumps the contents of the legal entity with the specified name, from the database",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if len(name) == 0 {
			return fmt.Errorf("error: legal entity name must be specified")
		}

		legalEntity, err := dumpCmdConfig.legalEntityStore.ReadByName(context.Background(), nil, models.ResourceName(name))
		if err != nil {
			return fmt.Errorf("error reading Legal Entity with name '%s': %w", name, err)
		}

		cli.Stdout.Printf("Legal Entity '%s':\n", name)
		cli.Stdout.Printf("  ID: %s", legalEntity.ID)
		cli.Stdout.Printf("  Created At: %s", legalEntity.CreatedAt.String())
		cli.Stdout.Printf("  Updated At: %s", legalEntity.UpdatedAt.String())
		cli.Stdout.Printf("  Name: %s", legalEntity.Name)
		cli.Stdout.Printf("  LegalName: %s", legalEntity.LegalName)
		cli.Stdout.Printf("  Type: %s", legalEntity.Type.String())
		cli.Stdout.Printf("  EmailAddress: %s", legalEntity.EmailAddress)
		cli.Stdout.Printf("  ExternalID: %s", legalEntity.ExternalID.String())
		cli.Stdout.Printf("  ExternalMetadata: %s", legalEntity.ExternalMetadata)

		return nil
	},
}

var dumpLegalEntityGroupsCmd = &cobra.Command{
	Use:           "legal-entity-groups name",
	Short:         "Dumps info about the groups and members of the legal entity with the specified name, from the database",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		name := args[0]
		if len(name) == 0 {
			return fmt.Errorf("error: legal entity name must be specified")
		}

		legalEntity, err := dumpCmdConfig.legalEntityStore.ReadByName(ctx, nil, models.ResourceName(name))
		if err != nil {
			return fmt.Errorf("error reading Legal Entity with name '%s': %w", name, err)
		}
		cli.Stdout.Printf("\nLEGAL ENTITY GROUPS AND MEMBERS REPORT\n\n")
		cli.Stdout.Printf("Name '%s', type '%s', ID '%s'\n", legalEntity.Name, legalEntity.Type.String(), legalEntity.ID)

		if legalEntity.Type == models.LegalEntityTypeCompany {
			err = dumpCmdConfig.db.WithTx(ctx, nil, func(tx *store.Tx) error {
				cli.Stdout.Printf("\nLegal Entity Members:\n")
				count := 0
				pagination := models.NewPagination(models.DefaultPaginationLimit, nil)
				for moreResults := true; moreResults; {
					members, cursor, err := dumpCmdConfig.legalEntityStore.ListMemberLegalEntities(ctx, tx, legalEntity.ID, pagination)
					if err != nil {
						return err
					}
					for _, member := range members {
						count++
						cli.Stdout.Printf("   Member %d: Name '%s', type '%s', ID '%s':\n", count, member.Name, member.Type.String(), member.ID)
					}
					if cursor != nil && cursor.Next != nil {
						pagination.Cursor = cursor.Next // move on to next page of results
					} else {
						moreResults = false
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
		}

		err = dumpCmdConfig.db.WithTx(ctx, nil, func(tx *store.Tx) error {
			cli.Stdout.Printf("\nAccess Control Groups:\n")
			count := 0
			pagination := models.NewPagination(models.DefaultPaginationLimit, nil)
			for moreResults := true; moreResults; {
				groups, cursor, err := dumpCmdConfig.groupStore.ListGroups(ctx, tx, &legalEntity.ID, nil, pagination)
				if err != nil {
					return err
				}
				for _, group := range groups {
					count++
					err = dumpGroup(tx, count, group)
					if err != nil {
						return err
					}
				}
				if cursor != nil && cursor.Next != nil {
					pagination.Cursor = cursor.Next // move on to next page of results
				} else {
					moreResults = false
				}
			}
			return nil
		})
		if err != nil {
			return err
		}

		return nil
	},
}

func dumpGroup(txOrNil *store.Tx, groupNr int, group *models.Group) error {
	db := dumpCmdConfig.db
	ctx := context.Background()

	cli.Stdout.Printf("   Group %d: Name '%s', ID '%s', Internal: %v, External-ID '%s'\n",
		groupNr, group.Name, group.ID, group.IsInternal, group.ExternalID)
	if group.Description != "" {
		cli.Stdout.Printf("      Desc '%s'\n", group.Description)
	}

	// Dump group members
	db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		count := 0
		pagination := models.NewPagination(models.DefaultPaginationLimit, nil)
		for moreResults := true; moreResults; {
			groupMemberships, cursor, err := dumpCmdConfig.groupMembershipStore.ListGroupMemberships(ctx, tx, &group.ID, nil, nil, pagination)
			if err != nil {
				return err
			}
			for _, groupMembership := range groupMemberships {
				count++
				if dumpCmdConfig.verbose {
					cli.Stdout.Printf("         Group membership %d (source system '%s'): %s\n",
						count, groupMembership.SourceSystem, identityToString(txOrNil, groupMembership.MemberIdentityID))
				}
			}
			if cursor != nil && cursor.Next != nil {
				pagination.Cursor = cursor.Next // move on to next page of results
			} else {
				moreResults = false
			}
		}
		cli.Stdout.Printf("      Group has %d group members\n", count)
		return nil
	})

	// Dump access control grants for the group
	db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		count := 0
		pagination := models.NewPagination(models.DefaultPaginationLimit, nil)
		for moreResults := true; moreResults; {
			grants, cursor, err := dumpCmdConfig.grantStore.ListGrantsForGroup(ctx, tx, group.ID, pagination)
			if err != nil {
				return err
			}
			for _, grant := range grants {
				count++
				if dumpCmdConfig.verbose {
					cli.Stdout.Printf("         Group Access Control Grant %d: %s for %s\n",
						count, grant.GetOperation().String(), grant.TargetResourceID.String())
				}
			}
			if cursor != nil && cursor.Next != nil {
				pagination.Cursor = cursor.Next // move on to next page of results
			} else {
				moreResults = false
			}
		}
		cli.Stdout.Printf("      Group has %d access control grants\n", count)
		return nil
	})

	cli.Stdout.Printf("\n")
	return nil
}

func identityToString(txOrNil *store.Tx, identityID models.IdentityID) string {
	ctx := context.Background()

	// Read the identity, check whether it is owned by a Legal Entity
	identity, err := dumpCmdConfig.identityStore.Read(ctx, txOrNil, identityID)
	if err != nil {
		return fmt.Sprintf("Identity ID '%s': error reading Identity: %s", identityID, err.Error())
	}
	if identity.OwnerResourceID.Kind() != models.LegalEntityResourceKind {
		return fmt.Sprintf("Identity ID '%s' for resource '%s'", identityID, identity.OwnerResourceID)
	}

	// Read legal entity that owns the identity
	legalEntityID := models.LegalEntityIDFromResourceID(identity.OwnerResourceID)
	legalEntity, err := dumpCmdConfig.legalEntityStore.Read(ctx, txOrNil, legalEntityID)
	if err != nil {
		return fmt.Sprintf("Identity ID '%s': error reading Legal Entity '%s' that owns resource: %s", identityID, legalEntityID, err.Error())
	}

	return fmt.Sprintf("Legal Entity '%s', type '%s' (IDs '%s', '%s')",
		legalEntity.Name, legalEntity.Type.String(), legalEntity.ID, identityID)
}
