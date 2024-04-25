package models

var BuildCreateOperation = &Operation{
	Name:         "create",
	ResourceKind: BuildResourceKind,
}

var BuildReadOperation = &Operation{
	Name:         "read",
	ResourceKind: BuildResourceKind,
}

var BuildUpdateOperation = &Operation{
	Name:         "update",
	ResourceKind: BuildResourceKind,
}

var BuildAccessControlOperations = []*Operation{
	BuildReadOperation,
	BuildUpdateOperation,
}
