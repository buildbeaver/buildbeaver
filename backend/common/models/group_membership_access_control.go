package models

var GroupMembershipCreateOperation = &Operation{
	Name:         "create",
	ResourceKind: GroupMembershipResourceKind,
}

var GroupMembershipReadOperation = &Operation{
	Name:         "read",
	ResourceKind: GroupMembershipResourceKind,
}

var GroupMembershipUpdateOperation = &Operation{
	Name:         "update",
	ResourceKind: GroupMembershipResourceKind,
}

var GroupMembershipDeleteOperation = &Operation{
	Name:         "delete",
	ResourceKind: GroupMembershipResourceKind,
}

var GroupMembershipAccessControlOperations = []*Operation{
	GroupMembershipReadOperation,
	GroupMembershipUpdateOperation,
	GroupMembershipDeleteOperation,
}
