package models

var GrantCreateOperation = &Operation{
	Name:         "create",
	ResourceKind: GrantResourceKind,
}

var GrantReadOperation = &Operation{
	Name:         "read",
	ResourceKind: GrantResourceKind,
}

var GrantUpdateOperation = &Operation{
	Name:         "update",
	ResourceKind: GrantResourceKind,
}

var GrantDeleteOperation = &Operation{
	Name:         "delete",
	ResourceKind: GrantResourceKind,
}

var GrantAccessControlOperations = []*Operation{
	GrantReadOperation,
	GrantUpdateOperation,
	GrantDeleteOperation,
}
