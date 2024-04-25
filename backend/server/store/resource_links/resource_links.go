package resource_links

import (
	"context"
	"fmt"

	"github.com/doug-martin/goqu/v9"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

func init() {
	store.MustDBModel(&models.ResourceLinkFragment{})
}

type ResourceLinkStore struct {
	table *store.ResourceTable
}

func NewStore(db *store.DB, logFactory logger.LogFactory) *ResourceLinkStore {
	return &ResourceLinkStore{
		table: store.NewResourceTableWithTableName(db, logFactory, "resource_link_fragments", &models.ResourceLinkFragment{}),
	}
}

// read an existing resource link fragment, looking it up by ResourceID.
// Returns models.ErrNotFound if the fragment does not exist.
func (d *ResourceLinkStore) read(ctx context.Context, txOrNil *store.Tx, id models.ResourceID) (*models.ResourceLinkFragment, error) {
	named := &models.ResourceLinkFragment{}
	return named, d.table.ReadByID(ctx, txOrNil, id, named)
}

// create a new resource link fragment.
// Returns store.ErrAlreadyExists if a fragment with matching unique properties already exists.
func (d *ResourceLinkStore) create(ctx context.Context, txOrNil *store.Tx, named *models.ResourceLinkFragment) error {
	return d.table.Create(ctx, txOrNil, named)
}

// update an existing resource link fragment with optimistic locking. Overrides all previous values using the supplied model.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (d *ResourceLinkStore) update(ctx context.Context, txOrNil *store.Tx, named *models.ResourceLinkFragment) error {
	return d.table.UpdateByID(ctx, txOrNil, named)
}

// Upsert creates a resource link fragment if it does not exist, otherwise it updates its mutable properties
// if they differ from the in-memory instance. Returns true,false if the resource was created
// and false,true if the resource was updated. false,false if neither a create or update was necessary.
func (d *ResourceLinkStore) Upsert(ctx context.Context, txOrNil *store.Tx, namedResource models.NamedResource) (bool, bool, error) {
	named := &models.ResourceLinkFragment{
		ID:        namedResource.GetID(),
		CreatedAt: namedResource.GetCreatedAt(),
		Name:      namedResource.GetName(),
		ParentID:  namedResource.GetParentID(),
		Kind:      namedResource.GetKind(),
	}
	return d.table.Upsert(ctx, txOrNil,
		func(tx *store.Tx) (models.Resource, error) {
			return d.read(ctx, tx, named.GetID())
		}, func(tx *store.Tx) error {
			return d.create(ctx, tx, named)
		}, func(tx *store.Tx, obj models.Resource) (bool, error) {
			return true, d.update(ctx, tx, named)
		})
}

// Delete permanently and idempotently deletes a resource link fragment for a resource, ensuring its name can now be reused.
func (d *ResourceLinkStore) Delete(ctx context.Context, txOrNil *store.Tx, resourceID models.ResourceID) error {
	return d.table.DeleteWhere(ctx, txOrNil,
		goqu.Ex{"resource_link_fragment_id": resourceID})
}

// Resolve the leaf resource fragment in a resource link.
// Returns models.ErrNotFound if the fragment does not exist.
func (d *ResourceLinkStore) Resolve(ctx context.Context, txOrNil *store.Tx, link models.ResourceLink) (*models.ResourceLinkFragment, error) {
	/*
		Example:
			SELECT "leaf".* FROM "resource_names" AS "leaf"
			INNER JOIN "resource_names" AS "p1" ON ("p1"."resource_name_id" = "leaf"."resource_name_parent_id")
			INNER JOIN "resource_names" AS "p2" ON ("p2"."resource_name_id" = "p1"."resource_name_parent_id")
			WHERE (("leaf"."resource_name_kind" = 'secret')
			AND ("leaf"."resource_name_name" = 'password')
			AND ("p1"."resource_name_kind" = 'repo')
			AND ("p1"."resource_name_name" = 'buildbeaver')
			AND ("p2"."resource_name_kind" = 'legal_entity')
			AND ("p2"."resource_name_name" = 'github-user')) LIMIT 1;
	*/
	var (
		leafAlias = "leaf"
		child     = leafAlias
		leaf      = link[len(link)-1]
		parents   = link[:len(link)-1]
	)
	s := goqu.
		From(goqu.T(d.table.TableName()).As(leafAlias)).
		Select(goqu.I(d.colAlias(leafAlias, "*"))).
		Where(goqu.I(d.colAlias(leafAlias, "resource_link_fragment_kind")).Eq(leaf.Kind)).
		Where(goqu.I(d.colAlias(leafAlias, "resource_link_fragment_name")).Eq(leaf.Name))
	j := 1
	for i := len(parents) - 1; i >= 0; i-- {
		parent := parents[i]
		alias := d.parentAlias(j)
		s = s.Join(goqu.T(d.table.TableName()).As(alias),
			goqu.On(goqu.Ex{d.colAlias(alias, "resource_link_fragment_id"): goqu.I(d.colAlias(child, "resource_link_fragment_parent_id"))}))
		s = s.Where(goqu.I(d.colAlias(alias, "resource_link_fragment_kind")).Eq(parent.Kind)).
			Where(goqu.I(d.colAlias(alias, "resource_link_fragment_name")).Eq(parent.Name))
		child = alias
		j++
	}
	named := &models.ResourceLinkFragment{}
	return named, d.table.ReadIn(ctx, txOrNil, named, s)
}

func (d *ResourceLinkStore) parentAlias(i int) string {
	return fmt.Sprintf("p%d", i)
}

func (d *ResourceLinkStore) colAlias(alias string, col string) string {
	return fmt.Sprintf("%s.%s", alias, col)
}
