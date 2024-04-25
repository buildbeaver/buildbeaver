package models

var GroupCreateOperation = &Operation{
	Name:         "create",
	ResourceKind: GroupResourceKind,
}

var GroupReadOperation = &Operation{
	Name:         "read",
	ResourceKind: GroupResourceKind,
}

var GroupUpdateOperation = &Operation{
	Name:         "update",
	ResourceKind: GroupResourceKind,
}

var GroupDeleteOperation = &Operation{
	Name:         "delete",
	ResourceKind: GroupResourceKind,
}

var GroupAccessControlOperations = []*Operation{
	GroupReadOperation,
	GroupUpdateOperation,
	GroupDeleteOperation,
	GroupMembershipCreateOperation,
}
