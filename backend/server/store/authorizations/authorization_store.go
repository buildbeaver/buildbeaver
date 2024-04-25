package authorizations

import (
	"context"

	"github.com/doug-martin/goqu/v9"
	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

const queryCountGrantsForOperation = `

WITH RECURSIVE ownership_hierarchy AS (
	SELECT
		access_control_ownership_owned_resource_id AS access_control_anchor_id,
		access_control_ownership_id,
		access_control_ownership_owner_resource_id,
		access_control_ownership_owned_resource_id
	FROM
		access_control_ownerships
	WHERE
		access_control_ownership_owned_resource_id = :access_control_target_resource_id
	UNION ALL
		SELECT
			child.access_control_anchor_id,
			parent.access_control_ownership_id,
			parent.access_control_ownership_owner_resource_id,
			parent.access_control_ownership_owned_resource_id
		FROM
			access_control_ownerships AS parent
		INNER JOIN
			ownership_hierarchy
			AS
				child
			ON
				child.access_control_ownership_owner_resource_id = parent.access_control_ownership_owned_resource_id
			AND
				child.access_control_ownership_id != parent.access_control_ownership_id
)

SELECT
	COUNT(*)
FROM
	access_control_grants
INNER JOIN
	ownership_hierarchy
	AS
		owned_resource
	ON
		owned_resource.access_control_ownership_owned_resource_id = access_control_grant_target_resource_id
WHERE
	access_control_grant_operation_name = :access_control_operation_name
AND
	access_control_grant_operation_resource_kind = :access_control_operation_resource_kind
AND (
	-- The identity was granted permission directly
	access_control_grant_authorized_identity_id = :access_control_authorized_identity_id

	-- Or the legal entity is a member of a group that was granted permission
	OR (
		SELECT
			access_control_group_membership_id
		FROM
			access_control_group_memberships
		WHERE
			access_control_group_membership_group_id = access_control_grant_authorized_group_id
		AND
			access_control_group_membership_member_identity_id = :access_control_authorized_identity_id
		LIMIT
			1
	) IS NOT NULL
)
`

// WithIsAuthorizedListFilter filters the supplied dataset to resources that the specified identity is
// authorized to perform operation on. Set resourceIDColumnName to the name of the id column of the
// resource table being searched.
//
// NOTE: It's important to add this filter to your query immediately after declaring the select/from. This is because
// this filter derives off of the supplied dataset, which will copy all WHERE and JOIN clauses that have already been
// set. But why derive if it creates this ordering problem, you ask? It's because we *want* to copy the dialect
// that's set on the dataset (this filter does some things (e.g. the union) that have different syntax depending
// on the underlying database).
func WithIsAuthorizedListFilter(
	dataset *goqu.SelectDataset,
	identityID models.IdentityID,
	operation models.Operation,
	resourceIDColumnName string) *goqu.SelectDataset {

	return dataset.WithRecursive(
		"ownership_hierarchy",
		dataset.From("access_control_ownerships").
			Select(
				goqu.I("access_control_ownership_owned_resource_id").As("access_control_anchor_id"),
				goqu.I("access_control_ownership_id"),
				goqu.I("access_control_ownership_owner_resource_id"),
				goqu.I("access_control_ownership_owned_resource_id"),
			).
			UnionAll(
				dataset.From(goqu.T("access_control_ownerships").As("parent")).
					Select(
						goqu.I("child.access_control_anchor_id"),
						goqu.I("parent.access_control_ownership_id"),
						goqu.I("parent.access_control_ownership_owner_resource_id"),
						goqu.I("parent.access_control_ownership_owned_resource_id")).
					InnerJoin(goqu.T("ownership_hierarchy").As("child"),
						goqu.On(
							goqu.I("child.access_control_ownership_owner_resource_id").Eq(goqu.I("parent.access_control_ownership_owned_resource_id")),
							goqu.I("child.access_control_ownership_id").Neq(goqu.I("parent.access_control_ownership_id")),
						),
					),
			),
	).InnerJoin(
		goqu.Select(
			goqu.I("owned_resource.access_control_anchor_id").As("access_control_anchor_id")).
			From(goqu.T("access_control_grants")).
			InnerJoin(goqu.T("ownership_hierarchy").As("owned_resource"),
				goqu.On(
					goqu.I("access_control_grant_target_resource_id").
						Eq(goqu.I("owned_resource.access_control_ownership_owned_resource_id")),
				),
			).
			LeftJoin(goqu.T("access_control_group_memberships"),
				goqu.On(
					goqu.I("access_control_grant_authorized_group_id").
						Eq(goqu.I("access_control_group_memberships.access_control_group_membership_group_id")),
				),
			).
			Where(goqu.I("access_control_grant_operation_name").Eq(operation.Name)).
			Where(goqu.I("access_control_grant_operation_resource_kind").Eq(operation.ResourceKind)).
			Where(
				goqu.Or(
					// The legal entity was granted permission directly
					goqu.I("access_control_grant_authorized_identity_id").Eq(identityID),
					// Or the legal entity is a member of a group that was granted permission
					goqu.I("access_control_group_memberships.access_control_group_membership_member_identity_id").Eq(identityID),
				)).As("access_control"),
		goqu.On(goqu.I(resourceIDColumnName).Eq(goqu.I("access_control.access_control_anchor_id"))),
	).Distinct() // do not produce duplicate results if there are multiple ways to gain access to a resource
}

type AuthorizationStore struct {
	db *store.DB
}

func NewStore(db *store.DB) *AuthorizationStore {
	return &AuthorizationStore{
		db: db,
	}
}

// CountGrantsForOperation counts the number of grants that an identity has for the specified operation
// against the specified resource. All pathways are explored to locate the grants, including direct,
// group membership and inheritance.
func (d *AuthorizationStore) CountGrantsForOperation(
	ctx context.Context,
	txOrNil *store.Tx,
	identityID models.IdentityID,
	operation *models.Operation,
	resourceID models.ResourceID,
) (int, error) {

	var count int

	err := d.db.Read(txOrNil, func(queryer store.Queryer, binder store.Binder) error {

		params := map[string]interface{}{
			"access_control_authorized_identity_id":  identityID,
			"access_control_target_resource_id":      resourceID,
			"access_control_operation_name":          operation.Name,
			"access_control_operation_resource_kind": operation.ResourceKind,
		}

		query, args, err := binder.BindNamed(queryCountGrantsForOperation, params)
		if err != nil {
			return errors.Wrap(err, "error binding query params")
		}

		row := queryer.QueryRowContext(ctx, query, args...)

		err = row.Scan(&count)
		if err != nil {
			return errors.Wrap(err, "error in scan")
		}

		return err
	})
	if err != nil {
		return -1, errors.Wrap(err, "error counting grants")
	}

	return count, nil
}
